// Auth cookie session helpers (P2.4) — parse/validate/set cookies and login.
package wp

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// NonceTick returns the WordPress nonce tick (ceil(time / (nonce_life/2))).
// Default nonce_life is one day (86400).
func NonceTick(now int64, action string) int64 {
	_ = action
	if now <= 0 {
		now = time.Now().Unix()
	}
	const nonceLife = 86400
	half := int64(nonceLife / 2)
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

// Login authenticates login/password, sets CurrentUser, and returns auth cookies.
func (c *Context) Login(login, password string, remember, secure bool, now int64) (*AuthCookieSet, error) {
	if c == nil || c.Users == nil {
		return nil, fmt.Errorf("wp: Login: nil Context")
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
	if user == nil || !user.CheckPassword(password) {
		return nil, fmt.Errorf("wp: Login: invalid credentials")
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
	verifier := sessionTokenVerifier(token)
	sessions[verifier] = map[string]any{
		"expiration": expiration,
		"login":      now,
	}
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
	return exp >= now, nil
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
		out[fmt.Sprint(k)] = val
	}
	return out, nil
}

func (c *Context) saveSessionTokens(userID int64, sessions map[string]any) error {
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
