package admin

import (
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PHPVersionCheck is a subset of the wp_check_php_version response (P7.140).
type PHPVersionCheck struct {
	IsAcceptable             bool
	IsSecure                 bool
	IsLowerThanFutureMinimum bool
	RecommendedVersion       string
	Version                  string
}

// CheckPHPVersion mirrors wp_check_php_version (P7.140).
// GoPress does not run PHP, so the default is nil (no dashboard nag).
// Filter pre_wp_check_php_version may return *PHPVersionCheck for tests.
func CheckPHPVersion(c *wp.Context) *PHPVersionCheck {
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("pre_wp_check_php_version", (*PHPVersionCheck)(nil))
		if v, ok := filtered.(*PHPVersionCheck); ok {
			return v
		}
	}
	return nil
}

// UpdatePHPURL mirrors wp_get_update_php_url (P7.140).
func UpdatePHPURL(c *wp.Context) string {
	url := "https://wordpress.org/support/update-php/"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_update_php_url", url); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return url
}

// dashboardPHPNagHTML builds the dashboard_php_nag widget (P7.140).
// Empty when CheckPHPVersion is nil, acceptable, or the user cannot update_php.
func dashboardPHPNagHTML(c *wp.Context) string {
	if c == nil || !c.CurrentUserCan("update_php") {
		return ""
	}
	check := CheckPHPVersion(c)
	if check == nil || check.IsAcceptable {
		return ""
	}
	ver := strings.TrimSpace(check.Version)
	if ver == "" {
		ver = "unknown"
	}
	title := "PHP Update Recommended"
	if check.IsLowerThanFutureMinimum {
		title = "PHP Update Required"
	}
	var msg string
	if !check.IsSecure {
		if check.IsLowerThanFutureMinimum {
			msg = "Your site is running on an outdated version of PHP (" + html.EscapeString(ver) +
				"), which does not receive security updates and soon will not be supported by WordPress. Ensure that PHP is updated on your server as soon as possible. Otherwise you will not be able to upgrade WordPress."
		} else {
			msg = "Your site is running on an outdated version of PHP (" + html.EscapeString(ver) +
				"), which does not receive security updates. It should be updated."
		}
	} else if check.IsLowerThanFutureMinimum {
		msg = "Your site is running on an outdated version of PHP (" + html.EscapeString(ver) +
			"), which soon will not be supported by WordPress. Ensure that PHP is updated on your server as soon as possible. Otherwise you will not be able to upgrade WordPress."
	} else {
		msg = "Your site is running on an outdated version of PHP (" + html.EscapeString(ver) +
			"), which should be updated."
	}

	classes := "postbox"
	if !check.IsSecure {
		classes += " php-no-security-updates"
	} else if check.IsLowerThanFutureMinimum {
		classes += " php-version-lower-than-future-minimum"
	}

	learnURL := UpdatePHPURL(c)
	var b strings.Builder
	b.WriteString(`<div id="dashboard_php_nag" class="`)
	b.WriteString(html.EscapeString(classes))
	b.WriteString(`">`)
	b.WriteString(`<h2 class="hndle"><span>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</span></h2>`)
	b.WriteString(`<div class="inside">`)
	b.WriteString(`<p class="bigger-bolder-text">`)
	b.WriteString(msg)
	b.WriteString(`</p>`)
	b.WriteString(`<p>What is PHP and how does it affect my site?</p>`)
	b.WriteString(`<p>PHP is one of the programming languages used to build WordPress. Newer versions of PHP receive regular security updates and may increase your site&#8217;s performance.`)
	if rec := strings.TrimSpace(check.RecommendedVersion); rec != "" {
		b.WriteString(` The minimum recommended version of PHP is `)
		b.WriteString(html.EscapeString(rec))
		b.WriteString(`.`)
	}
	b.WriteString(`</p>`)
	b.WriteString(`<p class="button-container">`)
	b.WriteString(`<a class="button button-primary" href="`)
	b.WriteString(html.EscapeString(learnURL))
	b.WriteString(`" target="_blank">Learn more about updating PHP`)
	b.WriteString(`<span class="screen-reader-text"> (opens in a new tab)</span>`)
	b.WriteString(`<span aria-hidden="true" class="dashicons dashicons-external"></span></a>`)
	b.WriteString(`</p>`)
	b.WriteString(`</div></div>`)
	return b.String()
}
