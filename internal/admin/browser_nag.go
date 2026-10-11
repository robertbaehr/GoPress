package admin

import (
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// BrowserVersionCheck is a subset of the wp_check_browser_version response (P7.141).
type BrowserVersionCheck struct {
	Name      string
	Upgrade   bool
	Insecure  bool
	UpdateURL string
	ImgSrc    string
	ImgSrcSSL string
	IsIE      bool
}

// CheckBrowserVersion mirrors wp_check_browser_version (P7.141).
// GoPress does not call api.wordpress.org browse-happy, so the default is nil
// (no dashboard nag). Filter pre_wp_check_browser_version may return
// *BrowserVersionCheck for tests.
func CheckBrowserVersion(c *wp.Context) *BrowserVersionCheck {
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("pre_wp_check_browser_version", (*BrowserVersionCheck)(nil))
		if v, ok := filtered.(*BrowserVersionCheck); ok {
			return v
		}
	}
	return nil
}

// dashboardBrowserNagHTML builds the dashboard_browser_nag widget (P7.141).
// Empty when CheckBrowserVersion is nil or does not request an upgrade/insecure notice.
func dashboardBrowserNagHTML(c *wp.Context) string {
	check := CheckBrowserVersion(c)
	if check == nil || (!check.Upgrade && !check.Insecure && !check.IsIE) {
		return ""
	}
	name := strings.TrimSpace(check.Name)
	if name == "" {
		name = "your browser"
	}
	updateURL := strings.TrimSpace(check.UpdateURL)
	if updateURL == "" {
		updateURL = "https://browsehappy.com/"
	}

	var msg string
	if check.IsIE {
		msg = "Internet Explorer does not give you the best WordPress experience. Switch to Microsoft Edge, or another more modern browser to get the most from your site."
	} else if check.Insecure {
		msg = "It looks like you&#8217;re using an insecure version of <a href=\"" +
			html.EscapeString(updateURL) + "\">" + html.EscapeString(name) +
			"</a>. Using an outdated browser makes your computer unsafe. For the best WordPress experience, please update your browser."
	} else {
		msg = "It looks like you&#8217;re using an old version of <a href=\"" +
			html.EscapeString(updateURL) + "\">" + html.EscapeString(name) +
			"</a>. For the best WordPress experience, please update your browser."
	}

	notice := ""
	browserNagClass := ""
	imgSrc := strings.TrimSpace(check.ImgSrc)
	if imgSrc != "" {
		notice += `<div class="alignright browser-icon"><img src="` + html.EscapeString(imgSrc) + `" alt="" /></div>`
		browserNagClass = " has-browser-icon"
	}
	notice += `<p class="browser-update-nag` + browserNagClass + `">` + msg + `</p>`

	browseHappy := "https://browsehappy.com/"
	var msgBrowseHappy string
	if check.IsIE {
		msgBrowseHappy = `Learn how to <a href="` + html.EscapeString(browseHappy) + `" class="update-browser-link">browse happy</a>`
	} else {
		msgBrowseHappy = `<a href="` + html.EscapeString(updateURL) + `" class="update-browser-link">Update ` +
			html.EscapeString(name) + `</a> or learn how to <a href="` + html.EscapeString(browseHappy) +
			`" class="browse-happy-link">browse happy</a>`
	}
	notice += `<p>` + msgBrowseHappy + `</p>`
	notice += `<p class="hide-if-no-js"><a href="" class="dismiss" aria-label="Dismiss the browser warning panel">Dismiss</a></p>`
	notice += `<div class="clear"></div>`

	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("browse-happy-notice", notice, check)
		if s, ok := filtered.(string); ok {
			notice = s
		}
	}

	classes := "postbox"
	if check.Insecure {
		classes += " browser-insecure"
	}
	title := "Your browser is out of date!"
	if check.Insecure {
		title = "You are using an insecure browser!"
	}

	var b strings.Builder
	b.WriteString(`<div id="dashboard_browser_nag" class="`)
	b.WriteString(html.EscapeString(classes))
	b.WriteString(`">`)
	b.WriteString(`<h2 class="hndle"><span>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</span></h2>`)
	b.WriteString(`<div class="inside">`)
	b.WriteString(notice)
	b.WriteString(`</div></div>`)
	return b.String()
}
