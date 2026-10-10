package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UserInsert holds fields for wp_insert_user (P4.62 subset).
type UserInsert struct {
	Login       string
	Pass        string // plaintext; hashed before insert
	Email       string
	URL         string
	Nicename    string
	DisplayName string
	Role        string
	Description string
}

// InsertUser creates a wp_users row and capability meta (wp_insert_user subset, P4.62).
func (c *Context) InsertUser(in UserInsert) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("wp: InsertUser: nil context")
	}
	login := strings.TrimSpace(in.Login)
	email := strings.TrimSpace(in.Email)
	if login == "" || email == "" {
		return 0, fmt.Errorf("wp: InsertUser: username and email required")
	}
	if existing, err := c.GetUserBy("login", login); err != nil {
		return 0, err
	} else if existing != nil {
		return 0, fmt.Errorf("wp: InsertUser: login exists")
	}
	if existing, err := c.GetUserBy("email", email); err != nil {
		return 0, err
	} else if existing != nil {
		return 0, fmt.Errorf("wp: InsertUser: email exists")
	}
	pass := in.Pass
	if pass == "" {
		pass = randomUserPassword()
	}
	hash, err := HashPassword(pass)
	if err != nil {
		return 0, err
	}
	nicename := strings.TrimSpace(in.Nicename)
	if nicename == "" {
		nicename = SanitizeTitle(login)
	}
	if nicename == "" {
		nicename = login
	}
	display := strings.TrimSpace(in.DisplayName)
	if display == "" {
		display = login
	}
	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "subscriber"
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Users()+
			" (user_login, user_pass, user_nicename, user_email, user_url, user_registered, user_activation_key, user_status, display_name) "+
			"VALUES (%s, %s, %s, %s, %s, %s, %s, %d, %s)",
		login, hash, nicename, email, strings.TrimSpace(in.URL), now, "", 0, display,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("wp: InsertUser: last insert id: %w", err)
	}
	if err := c.SetUserRole(id, role); err != nil {
		return id, err
	}
	if desc := strings.TrimSpace(in.Description); desc != "" && c.Meta != nil {
		_, _, _ = c.Meta.AddMetadata(MetaUser, id, "description", desc, true)
	}
	return id, nil
}

// UserUpdate holds fields for wp_update_user (P4.62 subset).
type UserUpdate struct {
	ID          int64
	Email       *string
	URL         *string
	DisplayName *string
	Nicename    *string
	Pass        *string // plaintext when set
	Role        *string
	Description *string
}

// UpdateUser updates an existing user row (wp_update_user subset, P4.62).
func (c *Context) UpdateUser(in UserUpdate) error {
	if c == nil || c.DB == nil || in.ID <= 0 {
		return fmt.Errorf("wp: UpdateUser: invalid state")
	}
	u, err := c.GetUserBy("id", in.ID)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("wp: UpdateUser: not found")
	}
	email := u.Email
	if in.Email != nil {
		email = strings.TrimSpace(*in.Email)
		if email == "" {
			return fmt.Errorf("wp: UpdateUser: empty email")
		}
		if other, err := c.GetUserBy("email", email); err != nil {
			return err
		} else if other != nil && other.ID != in.ID {
			return fmt.Errorf("wp: UpdateUser: email exists")
		}
	}
	url := u.URL
	if in.URL != nil {
		url = strings.TrimSpace(*in.URL)
	}
	display := u.DisplayName
	if in.DisplayName != nil {
		display = strings.TrimSpace(*in.DisplayName)
		if display == "" {
			display = u.Login
		}
	}
	nicename := u.Nicename
	if in.Nicename != nil {
		nicename = SanitizeTitle(strings.TrimSpace(*in.Nicename))
		if nicename == "" {
			nicename = u.Login
		}
	}
	pass := u.Pass
	if in.Pass != nil && strings.TrimSpace(*in.Pass) != "" {
		hash, err := HashPassword(strings.TrimSpace(*in.Pass))
		if err != nil {
			return err
		}
		pass = hash
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Users()+" SET user_pass = %s, user_nicename = %s, user_email = %s, user_url = %s, display_name = %s WHERE ID = %d",
		pass, nicename, email, url, display, in.ID,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Users != nil && c.Users.Cache != nil {
		c.Users.Cache.Delete(fmt.Sprintf("%d", in.ID), "users")
	}
	if in.Role != nil {
		if err := c.SetUserRole(in.ID, strings.TrimSpace(*in.Role)); err != nil {
			return err
		}
	}
	if in.Description != nil && c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(MetaUser, in.ID, "description", strings.TrimSpace(*in.Description), "")
	}
	if c.CurrentUser != nil && c.CurrentUser.ID == in.ID {
		c.CurrentUser.Email = email
		c.CurrentUser.URL = url
		c.CurrentUser.DisplayName = display
		c.CurrentUser.Nicename = nicename
		c.CurrentUser.Pass = pass
	}
	return nil
}

// SetUserRole writes wp_capabilities usermeta for a single role (P4.62).
func (c *Context) SetUserRole(userID int64, role string) error {
	if c == nil || c.Meta == nil || userID <= 0 {
		return fmt.Errorf("wp: SetUserRole: invalid state")
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "subscriber"
	}
	caps := map[any]any{role: true}
	serialized, err := SerializePHP(caps)
	if err != nil {
		return err
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaUser, userID, "wp_capabilities", serialized, "")
	return err
}

func randomUserPassword() string {
	// Deterministic-enough fallback; REST create always sends a password in tests.
	return fmt.Sprintf("GoPress%x", time.Now().UnixNano())
}

// DeleteUser removes a user and usermeta (wp_delete_user subset, P4.79).
// When reassign is non-nil and > 0, post_author rows are reassigned to that user.
// When reassign is nil, posts authored by the user are left in place (no cascade delete).
func (c *Context) DeleteUser(id int64, reassign *int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("wp: DeleteUser: invalid state")
	}
	u, err := c.GetUserBy("id", id)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("wp: DeleteUser: not found")
	}
	if reassign != nil {
		rid := *reassign
		if rid <= 0 || rid == id {
			return fmt.Errorf("wp: DeleteUser: invalid reassign")
		}
		target, err := c.GetUserBy("id", rid)
		if err != nil {
			return err
		}
		if target == nil {
			return fmt.Errorf("wp: DeleteUser: invalid reassign")
		}
		uq, uargs, err := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_author = %d WHERE post_author = %d",
			rid, id,
		)
		if err != nil {
			return err
		}
		if _, err := c.DB.Exec(uq, uargs...); err != nil {
			return err
		}
	}
	mq, margs, err := c.DB.Prepare("DELETE FROM "+c.DB.UserMeta()+" WHERE user_id = %d", id)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(mq, margs...); err != nil {
		return err
	}
	dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Users()+" WHERE ID = %d", id)
	if err != nil {
		return err
	}
	res, err := c.DB.Exec(dq, dargs...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("wp: DeleteUser: not found")
	}
	if c.Users != nil && c.Users.Cache != nil {
		c.Users.Cache.Delete(strconv.FormatInt(id, 10), "users")
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("deleted_user", id, reassign, u)
	}
	return nil
}
