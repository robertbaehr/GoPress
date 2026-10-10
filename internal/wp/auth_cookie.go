// Auth cookie session helpers (P2.4 / P2.22) — parse/validate/set cookies and login.
package wp

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AuthCookieParts is the parsed form of an authentication cookie.
type AuthCookieParts struct {
	Username   string
	Expiration int64
	Token      string
	HMAC       string
	Scheme     string
}

// AuthCookieSet holds cookie values produced by SetAuthCookie / Login.
type AuthCookieSet struct {
	UserID         int64
	Expiration     int64
	Expire         int64 // cookie MaxAge deadline (grace); 0 = session cookie
	Token          string
	Scheme         string
	AuthName       string
	AuthValue      string
	LoggedInName   string
	LoggedInValue  string
	SecureLoggedIn bool
}

// ParseAuthCookie splits a wordpress_* cookie value (wp_parse_auth_cookie).
func ParseAuthCookie(cookie, scheme string) (*AuthCookieParts, error) {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return nil, fmt.Errorf("wp: ParseAuthCookie: empty cookie")
	}
	parts := strings.Split(cookie, "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("wp: ParseAuthCookie: malformed cookie")
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("wp: ParseAuthCookie: expiration: %w", err)
	}
	if scheme == "" {
		scheme = "auth"
	}
	return &AuthCookieParts{
		Username:   parts[0],
		Expiration: exp,
		Token:      parts[2],
		HMAC:       parts[3],
		Scheme:     scheme,
	}, nil
}

// CookieHash returns md5(siteurl) used in WordPress cookie names (COOKIEHASH).
func (c *Context) CookieHash() string {
	if c == nil || c.Options == nil {
		return ""
	}
	siteurl := optionValueString(c.Options.GetOption("siteurl", ""))
	sum := md5.Sum([]byte(siteurl))
	return hex.EncodeToString(sum[:])
}

// AuthCookieName returns the HTTP cookie name for a scheme.
func (c *Context) AuthCookieName(scheme string) string {
	h := c.CookieHash()
	switch scheme {
	case "secure_auth":
		return "wordpress_sec_" + h
	case "logged_in":
		return "wordpress_logged_in_" + h
	default:
		return "wordpress_" + h
	}
}

// ValidateAuthCookie validates cookie contents and returns the user (wp_validate_auth_cookie).
// now is Unix seconds (use 0 for time.Now). When ajaxOrPOST is true, a 1-hour grace applies.
// Session tokens are verified against usermeta when Meta is available.
func (c *Context) ValidateAuthCookie(cookie, scheme string, now int64, ajaxOrPOST bool) (*User, error) {
	if c == nil || c.Auth == nil || c.Users == nil {
		return nil, fmt.Errorf("wp: ValidateAuthCookie: nil Context")
	}
	parts, err := ParseAuthCookie(cookie, scheme)
	if err != nil {
		return nil, err
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	expired := parts.Expiration
	if ajaxOrPOST {
		expired += 3600
	}
	if expired < now {
		return nil, fmt.Errorf("wp: ValidateAuthCookie: expired")
	}

	user, err := c.GetUserBy("login", parts.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("wp: ValidateAuthCookie: bad username")
	}

	want := c.Auth.GenerateAuthCookie(user.Login, user.Pass, parts.Expiration, parts.Scheme, parts.Token)
	wantParts, err := ParseAuthCookie(want, parts.Scheme)
	if err != nil {
		return nil, err
	}
	if !hmacEqual(wantParts.HMAC, parts.HMAC) {
		return nil, fmt.Errorf("wp: ValidateAuthCookie: bad hash")
	}

	if c.Meta != nil {
		ok, err := c.VerifySessionToken(user.ID, parts.Token, now)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("wp: ValidateAuthCookie: bad session token")
		}
	}
	return user, nil
}

// SetAuthCookie builds auth + logged_in cookie values and records a session token.
// secure selects secure_auth vs auth scheme. remember extends expiration to 14 days.
func (c *Context) SetAuthCookie(user *User, remember, secure bool, now int64, token string) (*AuthCookieSet, error) {
	if c == nil || c.Auth == nil || user == nil {
		return nil, fmt.Errorf("wp: SetAuthCookie: nil Context/user")
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	var expiration int64
	var expire int64
	if remember {
		expiration = now + 14*24*3600
		expire = expiration + 12*3600
	} else {
		expiration = now + 2*24*3600
		expire = 0
	}

	if token == "" {
		var err error
		token, err = c.CreateSessionToken(user.ID, expiration, now)
		if err != nil {
			return nil, err
		}
	} else if c.Meta != nil {
		if err := c.StoreSessionToken(user.ID, token, expiration, now); err != nil {
			return nil, err
		}
	}

	scheme := "auth"
	if secure {
		scheme = "secure_auth"
	}
	authVal := c.Auth.GenerateAuthCookie(user.Login, user.Pass, expiration, scheme, token)
	loggedIn := c.Auth.GenerateAuthCookie(user.Login, user.Pass, expiration, "logged_in", token)
	return &AuthCookieSet{
		UserID:        user.ID,
		Expiration:    expiration,
		Expire:        expire,
		Token:         token,
		Scheme:        scheme,
		AuthName:      c.AuthCookieName(scheme),
		AuthValue:     authVal,
		LoggedInName:  c.AuthCookieName("logged_in"),
		LoggedInValue: loggedIn,
	}, nil
}

// WriteAuthCookies writes SetAuthCookie values onto w (HttpOnly).
func (set *AuthCookieSet) WriteAuthCookies(w http.ResponseWriter, path string) {
	if set == nil || w == nil {
		return
	}
	if path == "" {
		path = "/"
	}
	write := func(name, value string, expire int64) {
		c := &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     path,
			HttpOnly: true,
		}
		if expire > 0 {
			c.Expires = time.Unix(expire, 0)
		}
		http.SetCookie(w, c)
	}
	write(set.AuthName, set.AuthValue, set.Expire)
	write(set.LoggedInName, set.LoggedInValue, set.Expire)
}

// ClearAuthCookies expires auth and logged_in cookies (wp_clear_auth_cookie).
func (c *Context) ClearAuthCookies(w http.ResponseWriter, path string) {
	if c == nil || w == nil {
		return
	}
	if path == "" {
		path = "/"
	}
	expired := time.Unix(1, 0)
	for _, name := range []string{
		c.AuthCookieName("auth"),
		c.AuthCookieName("secure_auth"),
		c.AuthCookieName("logged_in"),
	} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    " ",
			Path:     path,
			Expires:  expired,
			MaxAge:   -1,
			HttpOnly: true,
		})
	}
}

// DestroySessionToken removes one session token from usermeta (wp_destroy_current_session subset).
func (c *Context) DestroySessionToken(userID int64, token string) error {
	if c == nil || c.Meta == nil || userID <= 0 || token == "" {
		return nil
	}
	sessions, err := c.loadSessionTokens(userID)
	if err != nil || sessions == nil {
		return err
	}
	key := sessionTokenVerifier(token)
	delete(sessions, key)
	return c.saveSessionTokens(userID, sessions)
}

// DestroyOtherSessionTokens keeps token_to_keep and removes every other session
// (WP_Session_Tokens::destroy_others).
func (c *Context) DestroyOtherSessionTokens(userID int64, tokenToKeep string) error {
	if c == nil || c.Meta == nil || userID <= 0 {
		return nil
	}
	if tokenToKeep == "" {
		return c.DestroyAllSessionTokens(userID)
	}
	sessions, err := c.loadSessionTokens(userID)
	if err != nil {
		return err
	}
	key := sessionTokenVerifier(tokenToKeep)
	keep, ok := sessions[key]
	if !ok {
		return c.DestroyAllSessionTokens(userID)
	}
	return c.saveSessionTokens(userID, map[string]any{key: keep})
}

// DestroyAllSessionTokens clears session_tokens usermeta for one user
// (WP_User_Meta_Session_Tokens::destroy_all_sessions).
func (c *Context) DestroyAllSessionTokens(userID int64) error {
	if c == nil || c.Meta == nil || userID <= 0 {
		return nil
	}
	return c.saveSessionTokens(userID, map[string]any{})
}

// SessionTokenInfo is one usermeta session_tokens entry (subset of WP session array).
type SessionTokenInfo struct {
	Verifier   string
	Expiration int64
	Login      int64
	IP         string
	UA         string
}

// GetAllSessionTokens returns unexpired sessions for a user (WP_Session_Tokens::get_all).
func (c *Context) GetAllSessionTokens(userID int64, now int64) ([]SessionTokenInfo, error) {
	if c == nil || c.Meta == nil || userID <= 0 {
		return nil, nil
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	sessions, err := c.loadSessionTokens(userID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionTokenInfo, 0, len(sessions))
	for verifier, raw := range sessions {
		info := sessionTokenInfo(verifier, raw)
		if info.Expiration < now {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

// AuthenticateRequest reads the logged_in auth cookie and sets CurrentUser when valid.
// Invalid or missing cookies leave CurrentUser nil (anonymous).
func (c *Context) AuthenticateRequest() {
	if c == nil || c.Req == nil || c.Auth == nil || c.Users == nil {
		return
	}
	name := c.AuthCookieName("logged_in")
	cookie, err := c.Req.Cookie(name)
	if err != nil || cookie == nil || cookie.Value == "" {
		return
	}
	ajaxOrPOST := c.Req.Method == http.MethodPost
	user, err := c.ValidateAuthCookie(cookie.Value, "logged_in", 0, ajaxOrPOST)
	if err != nil || user == nil {
		return
	}
	c.CurrentUser = user
	if parts, err := ParseAuthCookie(cookie.Value, "logged_in"); err == nil && parts != nil {
		c.SessionToken = parts.Token
	}
}

// DefaultNonceLife is DAY_IN_SECONDS — half-life window for wp_nonce_tick.
const DefaultNonceLife int64 = 86400

// NonceTick returns the WordPress nonce tick (ceil(time / (nonce_life/2))).
// Default nonce_life is one day (86400). action is reserved for nonce_life filters.
func NonceTick(now int64, action string) int64 {
	return NonceTickLife(now, action, DefaultNonceLife)
}

// NonceTickLife is NonceTick with an explicit nonce_life (seconds).
func NonceTickLife(now int64, action string, nonceLife int64) int64 {
	_ = action
	if now <= 0 {
		now = time.Now().Unix()
	}
	if nonceLife <= 0 {
		nonceLife = DefaultNonceLife
	}
	half := nonceLife / 2
	if half <= 0 {
		half = 1
	}
	// PHP ceil(time / half)
	return (now + half - 1) / half
}

// CreateNonce builds a wp_create_nonce value for the current user/session.
func (c *Context) CreateNonce(action string) string {
	if c == nil || c.Auth == nil {
		return ""
	}
	uid := 0
	token := ""
	if c.CurrentUser != nil {
		uid = int(c.CurrentUser.ID)
		token = c.SessionToken
	}
	tick := NonceTick(0, action)
	return c.Auth.CreateNonce(tick, action, uid, token)
}

// VerifyNonce implements wp_verify_nonce.
// Returns 1 for the current tick, 2 for the previous tick, and 0 when invalid.
func (c *Context) VerifyNonce(nonce, action string) int {
	if c == nil || c.Auth == nil || nonce == "" {
		return 0
	}
	uid := 0
	token := ""
	if c.CurrentUser != nil {
		uid = int(c.CurrentUser.ID)
		token = c.SessionToken
	}
	tick := NonceTick(0, action)
	if c.Auth.CreateNonce(tick, action, uid, token) == nonce {
		return 1
	}
	if c.Auth.CreateNonce(tick-1, action, uid, token) == nonce {
		return 2
	}
	return 0
}

// AuthError is a typed login failure (wp_authenticate error codes, P6.12).
type AuthError struct {
	Code    string
	Login   string // submitted username/email when relevant
	Message string // optional override
}

func (e *AuthError) Error() string {
	if e == nil {
		return "wp: auth error"
	}
	if e.Message != "" {
		return e.Message
	}
	return "wp: Login: " + e.Code
}

// Login authenticates login/password, sets CurrentUser, and returns auth cookies.
func (c *Context) Login(login, password string, remember, secure bool, now int64) (*AuthCookieSet, error) {
	if c == nil || c.Users == nil {
		return nil, fmt.Errorf("wp: Login: nil Context")
	}
	login = strings.TrimSpace(login)
	if login == "" && password == "" {
		return nil, &AuthError{Code: "empty_username"}
	}
	if login == "" {
		return nil, &AuthError{Code: "empty_username"}
	}
	if password == "" {
		return nil, &AuthError{Code: "empty_password", Login: login}
	}
	user, err := c.GetUserBy("login", login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Try email.
		user, err = c.GetUserBy("email", login)
		if err != nil {
			return nil, err
		}
	}
	if user == nil {
		return nil, &AuthError{Code: "invalid_username", Login: login}
	}
	if !user.CheckPassword(password) {
		return nil, &AuthError{Code: "incorrect_password", Login: login}
	}
	set, err := c.SetAuthCookie(user, remember, secure, now, "")
	if err != nil {
		return nil, err
	}
	c.CurrentUser = user
	return set, nil
}

// CreateSessionToken generates a random session token and stores it.
func (c *Context) CreateSessionToken(userID, expiration, now int64) (string, error) {
	token, err := randomSessionToken(43)
	if err != nil {
		return "", err
	}
	if err := c.StoreSessionToken(userID, token, expiration, now); err != nil {
		return "", err
	}
	return token, nil
}

// StoreSessionToken records token under session_tokens usermeta (sha256 verifier key).
// When c.Req is set, IP and User-Agent are stored like WP_Session_Tokens::create.
func (c *Context) StoreSessionToken(userID int64, token string, expiration, now int64) error {
	if c == nil || c.Meta == nil || userID <= 0 || token == "" {
		return fmt.Errorf("wp: StoreSessionToken: invalid state")
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	sessions, err := c.loadSessionTokens(userID)
	if err != nil {
		return err
	}
	// Drop expired sessions while writing (WP get_sessions filters is_still_valid).
	for k, raw := range sessions {
		if sessionExpiration(raw) < now {
			delete(sessions, k)
		}
	}
	verifier := sessionTokenVerifier(token)
	entry := map[string]any{
		"expiration": expiration,
		"login":      now,
	}
	if c.Req != nil {
		if ip := strings.TrimSpace(c.Req.RemoteAddr); ip != "" {
			// Strip :port from host:port when present.
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip = host
			}
			entry["ip"] = ip
		}
		if ua := strings.TrimSpace(c.Req.UserAgent()); ua != "" {
			entry["ua"] = ua
		}
	}
	sessions[verifier] = entry
	return c.saveSessionTokens(userID, sessions)
}

// VerifySessionToken reports whether token exists and is unexpired.
func (c *Context) VerifySessionToken(userID int64, token string, now int64) (bool, error) {
	if c == nil || c.Meta == nil || userID <= 0 || token == "" {
		return false, nil
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	sessions, err := c.loadSessionTokens(userID)
	if err != nil {
		return false, err
	}
	verifier := sessionTokenVerifier(token)
	sess, ok := sessions[verifier]
	if !ok {
		return false, nil
	}
	exp := sessionExpiration(sess)
	if exp < now {
		// Lazy purge of the expired verifier.
		delete(sessions, verifier)
		_ = c.saveSessionTokens(userID, sessions)
		return false, nil
	}
	return true, nil
}

func (c *Context) loadSessionTokens(userID int64) (map[string]any, error) {
	// Bypass per-Context object cache so concurrent packages sharing the same
	// DB (go test ./...) see each other's session_tokens writes.
	if c.Meta != nil {
		c.Meta.invalidate(MetaUser, userID)
	}
	raw := c.Meta.GetMetadata(MetaUser, userID, "session_tokens", true)
	s, _ := raw.(string)
	if s == "" {
		return map[string]any{}, nil
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return nil, fmt.Errorf("wp: session_tokens: %w", err)
	}
	m, ok := v.(map[any]any)
	if !ok {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(m))
	for k, val := range m {
		out[fmt.Sprint(k)] = prepareSessionToken(val)
	}
	return out, nil
}

func (c *Context) saveSessionTokens(userID int64, sessions map[string]any) error {
	if len(sessions) == 0 {
		_, err := c.Meta.DeleteMetadata(MetaUser, userID, "session_tokens", "")
		return err
	}
	encoded := make(map[any]any, len(sessions))
	for k, v := range sessions {
		encoded[k] = v
	}
	s, err := phpSerialize(encoded)
	if err != nil {
		return err
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaUser, userID, "session_tokens", s, "")
	return err
}

// prepareSessionToken normalizes int expiration sessions to map form (WP prepare_session).
func prepareSessionToken(session any) any {
	switch t := session.(type) {
	case int64:
		return map[string]any{"expiration": t}
	case int:
		return map[string]any{"expiration": int64(t)}
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[fmt.Sprint(k)] = v
		}
		return out
	default:
		return session
	}
}

func sessionTokenInfo(verifier string, raw any) SessionTokenInfo {
	info := SessionTokenInfo{Verifier: verifier, Expiration: sessionExpiration(raw)}
	switch t := raw.(type) {
	case map[string]any:
		info.Login = asInt64OrZero(t["login"])
		info.IP, _ = t["ip"].(string)
		info.UA, _ = t["ua"].(string)
	case map[any]any:
		info.Login = asInt64OrZero(t["login"])
		if v, ok := t["ip"].(string); ok {
			info.IP = v
		}
		if v, ok := t["ua"].(string); ok {
			info.UA = v
		}
	}
	return info
}

func sessionTokenVerifier(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sessionExpiration(sess any) int64 {
	switch t := sess.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case map[any]any:
		return asInt64OrZero(t["expiration"])
	case map[string]any:
		return asInt64OrZero(t["expiration"])
	default:
		return 0
	}
}

func randomSessionToken(n int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = chars[int(buf[i])%len(chars)]
	}
	return string(buf), nil
}

func hmacEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
