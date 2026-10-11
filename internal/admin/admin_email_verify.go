package admin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// pendingNewAdminEmail returns the pending new_admin_email when it differs
// from the active admin_email (P7.142).
func pendingNewAdminEmail(c *wp.Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	current := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", "")))
	pending := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("new_admin_email", "")))
	if pending == "" || pending == "<nil>" || pending == current {
		return ""
	}
	return pending
}

// updateOptionNewAdminEmail mirrors update_option_new_admin_email (P7.142).
// When value equals admin_email or is not a valid email, no-op.
// Otherwise stores adminhash + new_admin_email and records a confirmation mail.
func updateOptionNewAdminEmail(c *wp.Context, value string) {
	if c == nil || c.Options == nil {
		return
	}
	value = strings.TrimSpace(value)
	current := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", "")))
	if value == "" || value == current || !wp.IsEmailValid(value) {
		return
	}
	hash := md5.Sum([]byte(value + fmt.Sprint(time.Now().UnixNano()) + fmt.Sprint(wp.Rand())))
	hashHex := hex.EncodeToString(hash[:])
	payload := map[string]any{
		"hash":     hashHex,
		"newemail": value,
	}
	serialized, err := wp.SerializePHP(payload)
	if err != nil {
		return
	}
	_, _ = c.Options.UpdateOption("adminhash", serialized, "no")
	_, _ = c.Options.UpdateOption("new_admin_email", value, "yes")

	username := "admin"
	if c.CurrentUser != nil && strings.TrimSpace(c.CurrentUser.Login) != "" {
		username = c.CurrentUser.Login
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	confirmURL := home + "/wp-admin/options.php?adminhash=" + url.QueryEscape(hashHex)
	body := "Howdy,\n\n" +
		"A site administrator (" + username + ") recently requested to have the\n" +
		"administration email address changed on this site:\n" +
		home + "\n\n" +
		"To confirm this change, please click on the following link:\n" +
		confirmURL + "\n\n" +
		"You can safely ignore and delete this email if you do not want to\n" +
		"take this action.\n\n" +
		"This email has been sent to " + value + "\n\n" +
		"Regards,\nAll at " + blogname + "\n" + home
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("new_admin_email_content", body, payload)
		if s, ok := filtered.(string); ok && strings.TrimSpace(s) != "" {
			body = s
		}
	}
	subject := "[" + blogname + "] New Admin Email Address"
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("new_admin_email_subject", subject)
		if s, ok := filtered.(string); ok && strings.TrimSpace(s) != "" {
			subject = s
		}
	}
	_, _ = wp.Mail(c, []string{value}, subject, body, nil)
}

// handleAdminEmailVerifyGET handles options.php?adminhash= / dismiss=new_admin_email (P7.142).
// Returns true when the request was handled (redirected).
func handleAdminEmailVerifyGET(w http.ResponseWriter, r *http.Request, c *wp.Context) bool {
	if r == nil || r.Method != http.MethodGet || c == nil || c.Options == nil {
		return false
	}
	q := r.URL.Query()
	if hash := strings.TrimSpace(q.Get("adminhash")); hash != "" {
		if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "manage_options") {
			http.Error(w, "Sorry, you are not allowed to manage options for this site.\n", http.StatusForbidden)
			return true
		}
		raw := c.Options.GetOption("adminhash", nil)
		details, _ := maybeUnserializeAdminHash(raw)
		redirect := "/wp-admin/options-general.php?updated=false"
		if details != nil {
			storedHash := strings.TrimSpace(fmt.Sprint(details["hash"]))
			newEmail := strings.TrimSpace(fmt.Sprint(details["newemail"]))
			if storedHash != "" && newEmail != "" && wp.HashEquals(storedHash, hash) {
				_, _ = c.Options.UpdateOption("admin_email", newEmail, "yes")
				_, _ = c.Options.DeleteOption("adminhash")
				_, _ = c.Options.DeleteOption("new_admin_email")
				redirect = "/wp-admin/options-general.php?updated=true&admin_email_updated=true"
			}
		}
		http.Redirect(w, r, redirect, http.StatusFound)
		return true
	}
	if q.Get("dismiss") == "new_admin_email" {
		if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "manage_options") {
			http.Error(w, "Sorry, you are not allowed to manage options for this site.\n", http.StatusForbidden)
			return true
		}
		nonce := q.Get("_wpnonce")
		if c.VerifyNonce(nonce, "dismiss-1-new_admin_email") == 0 {
			http.Error(w, "The link you followed has expired.\n", http.StatusForbidden)
			return true
		}
		_, _ = c.Options.DeleteOption("adminhash")
		_, _ = c.Options.DeleteOption("new_admin_email")
		http.Redirect(w, r, "/wp-admin/options-general.php?updated=true", http.StatusFound)
		return true
	}
	return false
}

func maybeUnserializeAdminHash(raw any) (map[string]any, bool) {
	switch t := raw.(type) {
	case map[string]any:
		return t, true
	case map[any]any:
		return stringKeyMap(t), true
	case string:
		if strings.TrimSpace(t) == "" {
			return nil, false
		}
		parsed, err := wp.UnserializePHP(t)
		if err != nil {
			return nil, false
		}
		switch m := parsed.(type) {
		case map[string]any:
			return m, true
		case map[any]any:
			return stringKeyMap(m), true
		}
	}
	return nil, false
}

func stringKeyMap(in map[any]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[fmt.Sprint(k)] = v
	}
	return out
}

// adminEmailPendingNoticeHTML builds pending-change notices for options-general (P7.142).
func adminEmailPendingNoticeHTML(c *wp.Context) (topNotice, inlineNotice string) {
	pending := pendingNewAdminEmail(c)
	if pending == "" {
		return "", ""
	}
	cancelURL := "/wp-admin/options.php?dismiss=new_admin_email"
	if c != nil {
		nonce := c.CreateNonce("dismiss-1-new_admin_email")
		cancelURL = "/wp-admin/options.php?dismiss=new_admin_email&_wpnonce=" + url.QueryEscape(nonce)
	}
	cancel := ` <a href="` + html.EscapeString(cancelURL) + `">Cancel request</a>`
	code := `<code>` + html.EscapeString(pending) + `</code>`
	topMsg := "The administration email address has not been updated yet. Please check the inbox at " +
		code + " for a confirmation email." + cancel
	topNotice = wp.GetAdminNotice(c, topMsg, wp.AdminNoticeArgs{Type: "info"})
	inlineMsg := "There is a pending change of the administration email to " + code + "." + cancel
	inlineNotice = wp.GetAdminNotice(c, inlineMsg, wp.AdminNoticeArgs{
		Type:              "info",
		AdditionalClasses: []string{"inline"},
	})
	return topNotice, inlineNotice
}
