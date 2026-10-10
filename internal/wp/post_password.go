// Password-protected post helpers (P6.15) — get_the_password_form / post_password_required subset.
package wp

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

// PostPasswordRequired reports whether the post still needs a password
// (post_password_required subset).
func PostPasswordRequired(c *Context, r *http.Request, post *Post) bool {
	if post == nil || strings.TrimSpace(post.Password) == "" {
		return false
	}
	if r == nil || c == nil {
		return true
	}
	name := "wp-postpass_" + c.CookieHash()
	cookie, err := r.Cookie(name)
	if err != nil || cookie == nil || cookie.Value == "" {
		return true
	}
	hash := cookie.Value
	if !strings.HasPrefix(hash, "$P$") && !strings.HasPrefix(hash, "$H$") {
		return true
	}
	return !phpassCheck(post.Password, hash)
}

// GetThePasswordForm returns the password form markup (get_the_password_form subset).
func GetThePasswordForm(c *Context, post *Post, redirectTo string) string {
	home := ""
	if c != nil {
		home = strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	}
	action := home + "/wp-login.php?action=postpass"
	if home == "" {
		action = "/wp-login.php?action=postpass"
	}
	fieldID := "pwbox"
	if post != nil && post.ID > 0 {
		fieldID = fmt.Sprintf("pwbox-%d", post.ID)
	}
	if redirectTo == "" && post != nil {
		redirectTo = postPermalinkForPassword(c, post)
	}
	var b strings.Builder
	b.WriteString(`<form action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" class="post-password-form" method="post">` + "\n")
	b.WriteString(`<p>This content is password protected. To view it please enter your password below:</p>` + "\n")
	b.WriteString(`<p><label for="`)
	b.WriteString(html.EscapeString(fieldID))
	b.WriteString(`">Password: <input name="post_password" id="`)
	b.WriteString(html.EscapeString(fieldID))
	b.WriteString(`" type="password" spellcheck="false" size="20" /></label>`)
	if redirectTo != "" {
		b.WriteString(`<input type="hidden" name="redirect_to" value="`)
		b.WriteString(html.EscapeString(redirectTo))
		b.WriteString(`" />`)
	}
	b.WriteString(` <input type="submit" name="Submit" value="Enter" /></p>` + "\n")
	b.WriteString("</form>\n")
	return b.String()
}

func postPermalinkForPassword(c *Context, post *Post) string {
	if post == nil {
		return ""
	}
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	if post.Type == "page" {
		return PagePermalink(c, NavPage{ID: post.ID, Title: post.Title, Name: post.Name})
	}
	if post.Name != "" {
		// Pretty date permalinks are site-specific; query form is always valid.
		return fmt.Sprintf("%s/?p=%d", home, post.ID)
	}
	return fmt.Sprintf("%s/?p=%d", home, post.ID)
}
