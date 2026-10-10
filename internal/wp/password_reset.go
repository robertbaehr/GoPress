// Password reset (P6.8 / P6.13) — get_password_reset_key / check_password_reset_key /
// reset_password / retrieve_password stubs.
package wp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var passwordResetKeySanitize = regexp.MustCompile(`(?i)[^a-z0-9]`)

const passwordResetAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GeneratePassword returns a random password (wp_generate_password subset).
// When special is false, only alphanumerics are used.
func GeneratePassword(length int, special bool) string {
	if length <= 0 {
		length = 12
	}
	alphabet := passwordResetAlphabet
	if special {
		alphabet += "!@#$%^&*()"
	}
	var b strings.Builder
	b.Grow(length)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			b.WriteByte(alphabet[i%len(alphabet)])
			continue
		}
		b.WriteByte(alphabet[n.Int64()])
	}
	return b.String()
}

// GetPasswordResetKey generates a reset key, stores time:fast_hash in
// user_activation_key, and returns the plaintext key (get_password_reset_key).
func GetPasswordResetKey(c *Context, user *User) (string, error) {
	if c == nil || user == nil || user.ID <= 0 {
		return "", fmt.Errorf("invalidcombo")
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("retrieve_password", user.Login)
	}
	key := GeneratePassword(20, false)
	if c.Hooks != nil {
		c.Hooks.DoAction("retrieve_password_key", user.Login, key)
	}
	hashed := fmt.Sprintf("%d:%s", time.Now().Unix(), FastHash(key))
	if err := c.SetUserActivationKey(user.ID, hashed); err != nil {
		return "", err
	}
	user.ActivationKey = hashed
	return key, nil
}

// SetUserActivationKey updates wp_users.user_activation_key and refreshes cache.
func (c *Context) SetUserActivationKey(userID int64, key string) error {
	if c == nil || c.DB == nil || userID <= 0 {
		return fmt.Errorf("wp: SetUserActivationKey: invalid state")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Users()+" SET user_activation_key = %s WHERE ID = %d",
		key, userID,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Users != nil && c.Users.Cache != nil {
		c.Users.Cache.Delete(fmt.Sprintf("%d", userID), "users")
	}
	return nil
}

// RetrievePassword looks up a user by login/email, generates a reset key, and
// records a wp_mail message (retrieve_password).
func RetrievePassword(c *Context, userLogin string) error {
	userLogin = strings.TrimSpace(userLogin)
	if userLogin == "" {
		return fmt.Errorf("empty_username")
	}
	var user *User
	var err error
	if strings.Contains(userLogin, "@") {
		user, err = c.GetUserBy("email", userLogin)
		if err != nil {
			return err
		}
		if user == nil {
			user, err = c.GetUserBy("login", userLogin)
		}
	} else {
		user, err = c.GetUserBy("login", userLogin)
	}
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalidcombo")
	}

	key, err := GetPasswordResetKey(c, user)
	if err != nil {
		return err
	}

	siteName := "GoPress"
	if c.Options != nil {
		if v, ok := c.Options.GetOption("blogname", siteName).(string); ok && v != "" {
			siteName = v
		}
	}
	home := ""
	if c.Options != nil {
		home, _ = c.Options.GetOption("home", "").(string)
		if home == "" {
			home, _ = c.Options.GetOption("siteurl", "").(string)
		}
	}
	home = strings.TrimRight(home, "/")
	resetURL := home + "/wp-login.php?action=rp&key=" + key + "&login=" + user.Login

	var msg strings.Builder
	msg.WriteString("Someone has requested a password reset for the following account:\r\n\r\n")
	msg.WriteString("Site Name: ")
	msg.WriteString(siteName)
	msg.WriteString("\r\n\r\n")
	msg.WriteString("Username: ")
	msg.WriteString(user.Login)
	msg.WriteString("\r\n\r\n")
	msg.WriteString("If this was a mistake, ignore this email and nothing will happen.\r\n\r\n")
	msg.WriteString("To reset your password, visit the following address:\r\n\r\n")
	msg.WriteString(resetURL)
	msg.WriteString("\r\n")

	subject := fmt.Sprintf("[%s] Password Reset", siteName)
	ok, err := Mail(c, []string{user.Email}, subject, msg.String(), nil)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("retrieve_password_email_failure")
	}
	return nil
}

// CheckPasswordResetKey validates a reset key for login (check_password_reset_key, P6.13).
// Returns the user on success, or an error with code "invalid_key" / "expired_key".
func CheckPasswordResetKey(c *Context, key, login string) (*User, error) {
	key = passwordResetKeySanitize.ReplaceAllString(key, "")
	login = strings.TrimSpace(login)
	if key == "" || login == "" {
		return nil, fmt.Errorf("invalid_key")
	}
	user, err := c.GetUserBy("login", login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid_key")
	}
	expirationDuration := DayInSeconds
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("password_reset_expiration", expirationDuration); v != nil {
			switch t := v.(type) {
			case int:
				expirationDuration = t
			case int64:
				expirationDuration = int(t)
			}
		}
	}
	stored := user.ActivationKey
	passKey := stored
	var expirationTime int64
	hasExpiration := false
	if strings.Contains(stored, ":") {
		parts := strings.SplitN(stored, ":", 2)
		if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			expirationTime = ts + int64(expirationDuration)
			hasExpiration = true
		}
		passKey = parts[1]
	}
	if passKey == "" {
		return nil, fmt.Errorf("invalid_key")
	}
	hashOK := VerifyFastHash(key, passKey)
	if hashOK && hasExpiration && time.Now().Unix() < expirationTime {
		return user, nil
	}
	if hashOK && hasExpiration {
		return nil, fmt.Errorf("expired_key")
	}
	// Old-style plaintext key or hash without expiration → treat as expired.
	if stored == key || (hashOK && !hasExpiration) {
		return nil, fmt.Errorf("expired_key")
	}
	return nil, fmt.Errorf("invalid_key")
}

// SetUserPassword updates user_pass and clears user_activation_key (wp_set_password).
func (c *Context) SetUserPassword(userID int64, password string) error {
	if c == nil || c.DB == nil || userID <= 0 {
		return fmt.Errorf("wp: SetUserPassword: invalid state")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Users()+" SET user_pass = %s, user_activation_key = %s WHERE ID = %d",
		hash, "", userID,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Users != nil && c.Users.Cache != nil {
		c.Users.Cache.Delete(fmt.Sprintf("%d", userID), "users")
	}
	if c.CurrentUser != nil && c.CurrentUser.ID == userID {
		c.CurrentUser.Pass = hash
		c.CurrentUser.ActivationKey = ""
	}
	return nil
}

// ResetPassword sets a new password after a successful key check (reset_password, P6.13).
func ResetPassword(c *Context, user *User, newPass string) error {
	if c == nil || user == nil || user.ID <= 0 {
		return fmt.Errorf("invalid_user")
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("password_reset", user, newPass)
	}
	if err := c.SetUserPassword(user.ID, newPass); err != nil {
		return err
	}
	if c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(MetaUser, user.ID, "default_password_nag", false, "")
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("after_password_reset", user, newPass)
	}
	return nil
}
