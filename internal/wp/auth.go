// Auth salts, hashes, nonces, and password helpers (P2.1).
// Ports wp_salt / wp_hash / wp_create_nonce formula / wp_fast_hash /
// wp_hash_password (bcrypt+$wp) / wp_generate_auth_cookie from pluggable.php.
package wp

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/blake2b"

	"github.com/robertbaehr/GoPress/internal/config"
)

// Auth binds salt/hash helpers to site config (keys + salts from wp-config).
type Auth struct {
	Config *config.Config
}

// NewAuth builds auth helpers. cfg must carry the WordPress secret keys/salts.
func NewAuth(cfg *config.Config) *Auth {
	return &Auth{Config: cfg}
}

// Salt returns the concatenated key+salt for a scheme (wp_salt).
func (a *Auth) Salt(scheme string) string {
	if a == nil || a.Config == nil {
		return ""
	}
	return a.Config.Salt(scheme)
}

// Hash returns HMAC-MD5 hex of data keyed by the scheme salt (wp_hash default algo).
func (a *Auth) Hash(data, scheme string) string {
	salt := a.Salt(scheme)
	mac := hmac.New(md5.New, []byte(salt))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// HashAlgo returns HMAC hex using the named algorithm (wp_hash with $algo).
// Supported: "md5" (default), "sha256".
func (a *Auth) HashAlgo(data, scheme, algo string) string {
	salt := []byte(a.Salt(scheme))
	switch strings.ToLower(algo) {
	case "", "md5":
		mac := hmac.New(md5.New, salt)
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))
	case "sha256":
		mac := hmac.New(sha256.New, salt)
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))
	default:
		mac := hmac.New(md5.New, salt)
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))
	}
}

// CreateNonce builds a nonce from the golden / wp_create_nonce formula:
//
//	substr( wp_hash( "$tick|$action|$uid|$token", "nonce" ), -12, 10 )
func (a *Auth) CreateNonce(tick int64, action string, uid int, token string) string {
	raw := a.Hash(fmt.Sprintf("%d|%s|%d|%s", tick, action, uid, token), "nonce")
	if len(raw) < 12 {
		return raw
	}
	// PHP substr($raw, -12, 10)
	start := len(raw) - 12
	return raw[start : start+10]
}

// FastHash returns wp_fast_hash: BLAKE2b- keyed generic hash, URL-safe base64.
func FastHash(message string) string {
	h, err := blake2b.New(30, []byte("wp_fast_hash_6.8+"))
	if err != nil {
		return ""
	}
	h.Write([]byte(message))
	sum := h.Sum(nil)
	return "$generic$" + base64.RawURLEncoding.EncodeToString(sum)
}

// VerifyFastHash checks message against a wp_fast_hash output.
func VerifyFastHash(message, hash string) bool {
	if !strings.HasPrefix(hash, "$generic$") {
		return false
	}
	return hmac.Equal([]byte(hash), []byte(FastHash(message)))
}

// HashPassword creates a WordPress 6.8+ bcrypt password hash with $wp prefix.
//
//	base64(HMAC-SHA384(trim(password), "wp-sha384")) → bcrypt → "$wp"+hash
func HashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) > 4096 {
		return "*", nil
	}
	mac := hmac.New(sha512.New384, []byte("wp-sha384"))
	mac.Write([]byte(password))
	pre := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	hashed, err := bcrypt.GenerateFromPassword([]byte(pre), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return "$wp" + string(hashed), nil
}

// CheckPassword verifies a plaintext password against a stored hash (wp_check_password).
// Formats covered (P2.21):
//   - $wp$2y$… WordPress 6.8+ (HMAC-SHA384 prehash + bcrypt)
//   - bare $2y$ / $2a$ / $2b$ bcrypt
//   - phpass portable $P$ / $H$
//
// Application passwords reuse this for non-$generic$ hashes via VerifyApplicationPasswordHash.
func CheckPassword(password, hash string) bool {
	password = strings.TrimSpace(password)
	if hash == "" || password == "" {
		return false
	}
	if strings.HasPrefix(hash, "$wp") {
		inner := hash[3:]
		mac := hmac.New(sha512.New384, []byte("wp-sha384"))
		mac.Write([]byte(password))
		pre := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		return bcrypt.CompareHashAndPassword([]byte(inner), []byte(pre)) == nil
	}
	if strings.HasPrefix(hash, "$2y$") || strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") {
		// Go's bcrypt accepts $2a$; map $2y$ → $2a$ for comparison.
		goHash := hash
		if strings.HasPrefix(hash, "$2y$") {
			goHash = "$2a$" + hash[4:]
		}
		return bcrypt.CompareHashAndPassword([]byte(goHash), []byte(password)) == nil
	}
	if strings.HasPrefix(hash, "$P$") || strings.HasPrefix(hash, "$H$") {
		return phpassCheck(password, hash)
	}
	return false
}

// WPCheckPassword is the wp_check_password alias for CheckPassword (P2.21).
func WPCheckPassword(password, hash string) bool {
	return CheckPassword(password, hash)
}

// GenerateAuthCookie builds an auth cookie string (wp_generate_auth_cookie).
func (a *Auth) GenerateAuthCookie(login, userPass string, expiration int64, scheme, token string) string {
	frag := passFrag(userPass)
	key := a.Hash(fmt.Sprintf("%s|%s|%d|%s", login, frag, expiration, token), scheme)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(fmt.Sprintf("%s|%d|%s", login, expiration, token)))
	sum := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s|%d|%s|%s", login, expiration, token, sum)
}

func passFrag(userPass string) string {
	if strings.HasPrefix(userPass, "$P$") || strings.HasPrefix(userPass, "$2y$") {
		if len(userPass) >= 12 {
			return userPass[8:12]
		}
	}
	if len(userPass) >= 4 {
		return userPass[len(userPass)-4:]
	}
	return userPass
}
