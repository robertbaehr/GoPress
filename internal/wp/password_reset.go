// Password reset (P6.8) — get_password_reset_key / retrieve_password stubs.
package wp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

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
