// Application Passwords (P8.1) — verify Basic auth against _application_passwords usermeta.
package wp

import (
	"fmt"
	"regexp"
	"strings"
)

const applicationPasswordsMetaKey = "_application_passwords"

var reAppPasswordAlnum = regexp.MustCompile(`(?i)[^a-z0-9]`)

// AuthenticateApplicationPassword attempts REST/XML-RPC Basic auth using application passwords.
// When CurrentUser is already set, or Basic credentials are absent, this is a no-op.
func (c *Context) AuthenticateApplicationPassword() {
	if c == nil || c.Req == nil || c.CurrentUser != nil || c.Users == nil || c.Meta == nil {
		return
	}
	user, pass, ok := c.Req.BasicAuth()
	if !ok || user == "" || pass == "" {
		return
	}
	u, err := c.authenticateApplicationPassword(user, pass)
	if err != nil || u == nil {
		return
	}
	c.CurrentUser = u
}

func (c *Context) authenticateApplicationPassword(username, password string) (*User, error) {
	u, err := c.GetUserBy("login", username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		u, err = c.GetUserBy("email", username)
		if err != nil {
			return nil, err
		}
	}
	if u == nil {
		return nil, fmt.Errorf("wp: application password: unknown user")
	}
	if !CheckApplicationPassword(c, u.ID, password) {
		return nil, fmt.Errorf("wp: application password: invalid")
	}
	return u, nil
}

// CheckApplicationPassword reports whether plaintext matches a stored application password hash.
func CheckApplicationPassword(c *Context, userID int64, password string) bool {
	if c == nil || c.Meta == nil || userID <= 0 || password == "" {
		return false
	}
	password = reAppPasswordAlnum.ReplaceAllString(password, "")
	if password == "" {
		return false
	}
	raw := c.Meta.GetMetadata(MetaUser, userID, applicationPasswordsMetaKey, true)
	s, _ := raw.(string)
	if s == "" {
		return false
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return false
	}
	list, ok := v.(map[any]any)
	if !ok {
		// Some serializers may yield []any for packed arrays — accept both.
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if matchAppPasswordItem(item, password) {
					return true
				}
			}
		}
		return false
	}
	for _, item := range list {
		if matchAppPasswordItem(item, password) {
			return true
		}
	}
	return false
}

func matchAppPasswordItem(item any, password string) bool {
	m, ok := item.(map[any]any)
	if !ok {
		return false
	}
	hash, _ := m["password"].(string)
	if hash == "" {
		return false
	}
	return VerifyApplicationPasswordHash(password, hash)
}

// VerifyApplicationPasswordHash checks plaintext against a stored application password hash.
func VerifyApplicationPasswordHash(password, hash string) bool {
	if strings.HasPrefix(hash, "$generic$") {
		return VerifyFastHash(password, hash)
	}
	return CheckPassword(password, hash)
}

// SetApplicationPasswordsForTest stores a single application password hash for tests.
func SetApplicationPasswordsForTest(c *Context, userID int64, uuid, name, hash string) error {
	if c == nil || c.Meta == nil || userID <= 0 || hash == "" {
		return fmt.Errorf("wp: SetApplicationPasswordsForTest: invalid state")
	}
	if uuid == "" {
		uuid = "00000000-0000-4000-8000-000000000099"
	}
	if name == "" {
		name = "GoPress Test"
	}
	payload := map[any]any{
		0: map[any]any{
			"uuid":      uuid,
			"app_id":    "gopress-test",
			"name":      name,
			"password":  hash,
			"created":   int64(1704103200),
			"last_used": nil,
			"last_ip":   nil,
		},
	}
	serialized, err := phpSerialize(payload)
	if err != nil {
		return err
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaUser, userID, applicationPasswordsMetaKey, serialized, "")
	return err
}
