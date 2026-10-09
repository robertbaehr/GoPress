// User loading (P2.2) — GetUserBy and password checks against wp_users.
package wp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
)

// User is a WordPress user row (wp_users fields needed for auth and caps).
type User struct {
	ID            int64
	Login         string
	Pass          string
	Nicename      string
	Email         string
	URL           string
	Registered    string
	ActivationKey string
	Status        int
	DisplayName   string

	// Caps is the raw role/cap map from usermeta (wp_capabilities); filled in P2.3.
	Caps map[string]bool
	// AllCaps merges role capabilities; filled in P2.3.
	AllCaps map[string]bool
	Roles   []string
}

// Users is the user lookup API bound to a DB (and optional object cache).
type Users struct {
	DB    *db.DB
	Cache *ObjectCache
}

// NewUsers builds a Users API. cache may be nil.
func NewUsers(database *db.DB, cache *ObjectCache) *Users {
	if cache == nil {
		cache = NewObjectCache()
	}
	return &Users{DB: database, Cache: cache}
}

// GetUserBy loads a user by "id", "login", "email", or "slug" (user_nicename).
// Mirrors get_user_by(). Returns nil, nil when not found.
func (u *Users) GetUserBy(field string, value any) (*User, error) {
	if u == nil || u.DB == nil {
		return nil, fmt.Errorf("wp: GetUserBy: nil Users/DB")
	}
	field = strings.ToLower(strings.TrimSpace(field))
	col, err := userFieldColumn(field)
	if err != nil {
		return nil, err
	}

	var query string
	var args []any
	switch col {
	case "ID":
		id := asInt64OrZero(value)
		if id <= 0 {
			return nil, nil
		}
		if cached := u.cachedUser(id); cached != nil {
			return cached, nil
		}
		query, args, err = u.DB.Prepare(
			"SELECT ID, user_login, user_pass, user_nicename, user_email, user_url, user_registered, user_activation_key, user_status, display_name FROM "+u.DB.Users()+" WHERE ID = %d LIMIT 1",
			id,
		)
		if err != nil {
			return nil, err
		}
	default:
		s := strings.TrimSpace(metaValueString(value))
		if s == "" {
			return nil, nil
		}
		query, args, err = u.DB.Prepare(
			"SELECT ID, user_login, user_pass, user_nicename, user_email, user_url, user_registered, user_activation_key, user_status, display_name FROM "+u.DB.Users()+" WHERE "+col+" = %s LIMIT 1",
			s,
		)
		if err != nil {
			return nil, err
		}
	}

	row, err := u.DB.GetRow(query, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	user := userFromRow(row)
	u.storeUser(user)
	return user, nil
}

// CheckPassword verifies plaintext against this user's stored hash.
func (u *User) CheckPassword(password string) bool {
	if u == nil {
		return false
	}
	return CheckPassword(password, u.Pass)
}

func userFieldColumn(field string) (string, error) {
	switch field {
	case "id":
		return "ID", nil
	case "login", "user_login":
		return "user_login", nil
	case "email", "user_email":
		return "user_email", nil
	case "slug", "nicename", "user_nicename":
		return "user_nicename", nil
	default:
		return "", fmt.Errorf("wp: GetUserBy: unsupported field %q", field)
	}
}

func userFromRow(row map[string]any) *User {
	return &User{
		ID:            asInt64OrZero(row["ID"]),
		Login:         metaValueString(row["user_login"]),
		Pass:          metaValueString(row["user_pass"]),
		Nicename:      metaValueString(row["user_nicename"]),
		Email:         metaValueString(row["user_email"]),
		URL:           metaValueString(row["user_url"]),
		Registered:    metaValueString(row["user_registered"]),
		ActivationKey: metaValueString(row["user_activation_key"]),
		Status:        int(asInt64OrZero(row["user_status"])),
		DisplayName:   metaValueString(row["display_name"]),
	}
}

func (u *Users) cachedUser(id int64) *User {
	if u == nil || u.Cache == nil || id <= 0 {
		return nil
	}
	v, ok := u.Cache.Get(strconv.FormatInt(id, 10), "users")
	if !ok {
		return nil
	}
	user, _ := v.(*User)
	return user
}

func (u *Users) storeUser(user *User) {
	if u == nil || u.Cache == nil || user == nil || user.ID <= 0 {
		return
	}
	u.Cache.Set(strconv.FormatInt(user.ID, 10), user, "users")
}
