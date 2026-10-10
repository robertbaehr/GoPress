package wp

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

// AuthCheck reports whether the current user is still authenticated for the
// admin auth-check / Heartbeat response (wp_auth_check, P2.19).
//
// Returns true when a logged-in user is present and not in the cookie grace
// period. GoPress does not yet model login_grace_period; a non-nil CurrentUser
// is treated as fully authenticated.
func (c *Context) AuthCheck() bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return true
}

// AuthCheckHTML returns the wp-auth-check dialog markup for admin footers
// (wp_auth_check_html stub; P2.19 / P6.36 interim modal).
func (c *Context) AuthCheckHTML() string {
	home := ""
	if c != nil && c.Options != nil {
		if v, ok := c.Options.GetOption("home", "").(string); ok {
			home = v
		}
		if home == "" {
			if v, ok := c.Options.GetOption("siteurl", "").(string); ok {
				home = v
			}
		}
	}
	home = strings.TrimRight(home, "/")
	loginURL := home + "/wp-login.php"

	sameDomain := authCheckSameDomain(c, loginURL)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_auth_check_same_domain", sameDomain); v != nil {
			if b, ok := v.(bool); ok {
				sameDomain = b
			}
		}
	}
	wrapClass := "hidden"
	if !sameDomain {
		wrapClass = "hidden fallback"
	}

	var b strings.Builder
	b.WriteString(`<div id="wp-auth-check-wrap" class="` + html.EscapeString(wrapClass) + `">` + "\n")
	b.WriteString(`<div id="wp-auth-check-bg"></div>` + "\n")
	b.WriteString(`<div id="wp-auth-check">` + "\n")
	b.WriteString(`<button type="button" class="wp-auth-check-close button-link"><span class="screen-reader-text">Close dialog</span></button>` + "\n")
	if sameDomain {
		loginSrc := loginURL + "?interim-login=1"
		if lang := authCheckUserLocale(c); lang != "" {
			loginSrc += "&wp_lang=" + url.QueryEscape(lang)
		}
		b.WriteString(`<div id="wp-auth-check-form" class="loading" data-src="`)
		b.WriteString(html.EscapeString(loginSrc))
		b.WriteString(`"></div>` + "\n")
	}
	b.WriteString(`<div class="wp-auth-fallback">` + "\n")
	b.WriteString(`<p><b class="wp-auth-fallback-expired" tabindex="0">Session expired</b></p>` + "\n")
	b.WriteString(`<p><a href="`)
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString(`" target="_blank">Please log in again.</a>` + "\n")
	b.WriteString(`The login page will open in a new tab. After logging in you can close it and return to this page.</p>` + "\n")
	b.WriteString(`</div>` + "\n")
	b.WriteString(`</div>` + "\n")
	b.WriteString(`</div>` + "\n")
	return b.String()
}

func authCheckSameDomain(c *Context, loginURL string) bool {
	host := ""
	if c != nil && c.Req != nil {
		host = strings.TrimSpace(c.Req.Host)
	}
	if host == "" {
		// No request context: treat configured home as same-domain (admin footer default).
		return true
	}
	scheme := "http://"
	if c.Req.TLS != nil || strings.EqualFold(c.Req.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https://"
	}
	currentDomain := scheme + host
	return strings.HasPrefix(loginURL, currentDomain)
}

func authCheckUserLocale(c *Context) string {
	if c == nil {
		return ""
	}
	if c.CurrentUser != nil && c.Meta != nil {
		if s := strings.TrimSpace(fmt.Sprint(c.Meta.GetMetadata(MetaUser, c.CurrentUser.ID, "locale", true))); s != "" && s != "<nil>" {
			return s
		}
	}
	if c.Options != nil {
		if s := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("WPLANG", ""))); s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}
