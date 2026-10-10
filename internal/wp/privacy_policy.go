// Privacy policy page helpers (P8.9) — get_privacy_policy_url / get_the_privacy_policy_link.
package wp

import (
	"fmt"
	"html"
	"strings"
)

// PrivacyPolicyPageID returns wp_page_for_privacy_policy as an int64.
func PrivacyPolicyPageID(c *Context) int64 {
	if c == nil || c.Options == nil {
		return 0
	}
	switch v := c.Options.GetOption("wp_page_for_privacy_policy", 0).(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		return parseInt64Loose(v)
	default:
		return 0
	}
}

// GetPrivacyPolicyURL returns the permalink of the published privacy policy page
// (get_privacy_policy_url). Empty when unset or not published.
func GetPrivacyPolicyURL(c *Context) string {
	id := PrivacyPolicyPageID(c)
	if id <= 0 {
		return ""
	}
	post, err := GetPostByID(c, id)
	if err != nil || post == nil || post.Status != "publish" || post.Type != "page" {
		return ""
	}
	return PagePermalink(c, NavPage{ID: post.ID, Title: post.Title, Name: post.Name})
}

// GetThePrivacyPolicyLink returns markup for the privacy policy link
// (get_the_privacy_policy_link). Empty when no published policy page.
func GetThePrivacyPolicyLink(c *Context, before, after string) string {
	url := GetPrivacyPolicyURL(c)
	id := PrivacyPolicyPageID(c)
	title := ""
	if id > 0 {
		if post, err := GetPostByID(c, id); err == nil && post != nil {
			title = post.Title
		}
	}
	if url == "" || title == "" {
		return ""
	}
	link := fmt.Sprintf(
		`<a class="privacy-policy-link" href="%s" rel="privacy-policy">%s</a>`,
		html.EscapeString(url),
		html.EscapeString(title),
	)
	return before + link + after
}

// PrivacyPolicyPageLinkHTML is a convenience wrapper used by login/admin screens.
func PrivacyPolicyPageLinkHTML(c *Context) string {
	return GetThePrivacyPolicyLink(c, `<div class="privacy-policy-page-link">`, `</div>`)
}

// PrivacyPolicyAdminSummary returns a short admin note about the configured page.
func PrivacyPolicyAdminSummary(c *Context) string {
	id := PrivacyPolicyPageID(c)
	if id <= 0 {
		return "No Privacy Policy page is set."
	}
	post, err := GetPostByID(c, id)
	if err != nil || post == nil {
		return "No Privacy Policy page is set."
	}
	edit := fmt.Sprintf("/wp-admin/post.php?post=%d&action=edit", post.ID)
	status := post.Status
	title := strings.TrimSpace(post.Title)
	if title == "" {
		title = post.Name
	}
	return fmt.Sprintf(
		`Privacy Policy page: <a href="%s">%s</a> (%s).`,
		html.EscapeString(edit),
		html.EscapeString(title),
		html.EscapeString(status),
	)
}
