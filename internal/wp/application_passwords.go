// Application Passwords (P8.1 / P4.43) — verify and manage _application_passwords usermeta.
package wp

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const applicationPasswordsMetaKey = "_application_passwords"

// ApplicationPasswordLength is WP_Application_Passwords::PW_LENGTH.
const ApplicationPasswordLength = 24

var reAppPasswordAlnum = regexp.MustCompile(`(?i)[^a-z0-9]`)

// ApplicationPassword is one stored application password item (minus hash in REST).
type ApplicationPassword struct {
	UUID     string
	AppID    string
	Name     string
	Password string // stored hash; never exposed via REST except as new plaintext once
	Created  int64
	LastUsed *int64
	LastIP   *string
}

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
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return false
	}
	for _, item := range list {
		if VerifyApplicationPasswordHash(password, item.Password) {
			return true
		}
	}
	return false
}

// GetUserApplicationPasswords returns stored application passwords for a user.
func GetUserApplicationPasswords(c *Context, userID int64) ([]ApplicationPassword, error) {
	if c == nil || c.Meta == nil || userID <= 0 {
		return nil, fmt.Errorf("wp: GetUserApplicationPasswords: invalid state")
	}
	raw := c.Meta.GetMetadata(MetaUser, userID, applicationPasswordsMetaKey, true)
	s, _ := raw.(string)
	if s == "" {
		return nil, nil
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return nil, err
	}
	return parseApplicationPasswordList(v), nil
}

func parseApplicationPasswordList(v any) []ApplicationPassword {
	var items []any
	switch t := v.(type) {
	case map[any]any:
		// Preserve numeric order when possible.
		max := -1
		for k := range t {
			switch n := k.(type) {
			case int:
				if n > max {
					max = n
				}
			case int64:
				if int(n) > max {
					max = int(n)
				}
			}
		}
		if max >= 0 {
			for i := 0; i <= max; i++ {
				if item, ok := t[i]; ok {
					items = append(items, item)
				} else if item, ok := t[int64(i)]; ok {
					items = append(items, item)
				}
			}
		} else {
			for _, item := range t {
				items = append(items, item)
			}
		}
	case []any:
		items = t
	default:
		return nil
	}
	out := make([]ApplicationPassword, 0, len(items))
	for _, item := range items {
		if ap, ok := applicationPasswordFromAny(item); ok {
			out = append(out, ap)
		}
	}
	return out
}

func applicationPasswordFromAny(item any) (ApplicationPassword, bool) {
	m, ok := item.(map[any]any)
	if !ok {
		return ApplicationPassword{}, false
	}
	ap := ApplicationPassword{
		UUID:     anyString(m["uuid"]),
		AppID:    anyString(m["app_id"]),
		Name:     anyString(m["name"]),
		Password: anyString(m["password"]),
		Created:  anyInt64(m["created"]),
	}
	if v, ok := m["last_used"]; ok && v != nil {
		n := anyInt64(v)
		ap.LastUsed = &n
	}
	if v, ok := m["last_ip"]; ok && v != nil {
		s := anyString(v)
		if s != "" {
			ap.LastIP = &s
		}
	}
	return ap, ap.UUID != "" || ap.Password != ""
}

func anyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(t)
	}
}

func anyInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		var n int64
		_, _ = fmt.Sscan(t, &n)
		return n
	default:
		return 0
	}
}

// GetUserApplicationPassword returns one password by UUID, or nil.
func GetUserApplicationPassword(c *Context, userID int64, uuid string) (*ApplicationPassword, error) {
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].UUID == uuid {
			ap := list[i]
			return &ap, nil
		}
	}
	return nil, nil
}

// CreateApplicationPassword creates a new application password.
// Returns the plaintext password (chunked for display) and the stored item.
func CreateApplicationPassword(c *Context, userID int64, name, appID string) (plain string, item ApplicationPassword, err error) {
	if c == nil || c.Meta == nil || userID <= 0 {
		return "", ApplicationPassword{}, fmt.Errorf("wp: CreateApplicationPassword: invalid state")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ApplicationPassword{}, fmt.Errorf("application_password_empty_name")
	}
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return "", ApplicationPassword{}, err
	}
	for _, existing := range list {
		if strings.EqualFold(existing.Name, name) {
			return "", ApplicationPassword{}, fmt.Errorf("application_password_duplicate_name")
		}
	}
	plain = GeneratePassword(ApplicationPasswordLength, false)
	hash := FastHash(plain)
	item = ApplicationPassword{
		UUID:     GenerateUUID4(),
		AppID:    strings.TrimSpace(appID),
		Name:     name,
		Password: hash,
		Created:  time.Now().Unix(),
	}
	list = append(list, item)
	if err := setUserApplicationPasswords(c, userID, list); err != nil {
		return "", ApplicationPassword{}, err
	}
	return ChunkPassword(plain), item, nil
}

// UpdateApplicationPassword updates the name of an application password.
func UpdateApplicationPassword(c *Context, userID int64, uuid, name string) (*ApplicationPassword, error) {
	if c == nil || c.Meta == nil || userID <= 0 || uuid == "" {
		return nil, fmt.Errorf("wp: UpdateApplicationPassword: invalid state")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("application_password_empty_name")
	}
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return nil, err
	}
	found := -1
	for i := range list {
		if list[i].UUID == uuid {
			found = i
			break
		}
	}
	if found < 0 {
		return nil, fmt.Errorf("application_password_not_found")
	}
	list[found].Name = name
	if err := setUserApplicationPasswords(c, userID, list); err != nil {
		return nil, err
	}
	ap := list[found]
	return &ap, nil
}

// DeleteApplicationPassword removes one application password by UUID.
func DeleteApplicationPassword(c *Context, userID int64, uuid string) (*ApplicationPassword, error) {
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return nil, err
	}
	var prev *ApplicationPassword
	out := make([]ApplicationPassword, 0, len(list))
	for i := range list {
		if list[i].UUID == uuid {
			ap := list[i]
			prev = &ap
			continue
		}
		out = append(out, list[i])
	}
	if prev == nil {
		return nil, fmt.Errorf("application_password_not_found")
	}
	if err := setUserApplicationPasswords(c, userID, out); err != nil {
		return nil, err
	}
	return prev, nil
}

// DeleteAllApplicationPasswords removes all application passwords for a user.
func DeleteAllApplicationPasswords(c *Context, userID int64) (int, error) {
	list, err := GetUserApplicationPasswords(c, userID)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, nil
	}
	if err := setUserApplicationPasswords(c, userID, nil); err != nil {
		return 0, err
	}
	return len(list), nil
}

func setUserApplicationPasswords(c *Context, userID int64, list []ApplicationPassword) error {
	payload := map[any]any{}
	for i, ap := range list {
		item := map[any]any{
			"uuid":      ap.UUID,
			"app_id":    ap.AppID,
			"name":      ap.Name,
			"password":  ap.Password,
			"created":   ap.Created,
			"last_used": nil,
			"last_ip":   nil,
		}
		if ap.LastUsed != nil {
			item["last_used"] = *ap.LastUsed
		}
		if ap.LastIP != nil {
			item["last_ip"] = *ap.LastIP
		}
		payload[i] = item
	}
	serialized, err := phpSerialize(payload)
	if err != nil {
		return err
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaUser, userID, applicationPasswordsMetaKey, serialized, "")
	return err
}

// ChunkPassword splits a password into 4-char groups (WP_Application_Passwords::chunk_password).
func ChunkPassword(raw string) string {
	raw = reAppPasswordAlnum.ReplaceAllString(raw, "")
	var b strings.Builder
	for i, r := range raw {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// GenerateUUID4 returns a random RFC 4122 version-4 UUID string.
func GenerateUUID4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back to time-based bytes.
		n := time.Now().UnixNano()
		for i := 0; i < 16; i++ {
			b[i] = byte(n >> (8 * (i % 8)))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
	list := []ApplicationPassword{{
		UUID:     uuid,
		AppID:    "gopress-test",
		Name:     name,
		Password: hash,
		Created:  1704103200,
	}}
	return setUserApplicationPasswords(c, userID, list)
}
