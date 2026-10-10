package admin

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// siteHealthDashboardWidgetHTML builds the Site Health Status dashboard widget
// (P7.57 / wp_dashboard_site_health).
func siteHealthDashboardWidgetHTML(c *wp.Context) string {
	counts := siteHealthIssueCounts(c)
	hasResults := counts != nil
	if counts == nil {
		counts = map[string]int{"good": 0, "recommended": 0, "critical": 0}
	}
	issuesTotal := counts["recommended"] + counts["critical"]

	var b strings.Builder
	b.WriteString(`<div id="dashboard_site_health" class="postbox">`)
	b.WriteString(`<h2 class="hndle"><span>Site Health Status</span></h2>`)
	b.WriteString(`<div class="inside">`)
	b.WriteString(`<div class="health-check-widget">`)
	b.WriteString(`<div class="health-check-widget-title-section site-health-progress-wrapper`)
	if !hasResults {
		b.WriteString(` loading`)
	}
	b.WriteString(`">`)
	b.WriteString(`<div class="site-health-progress">`)
	b.WriteString(`<svg aria-hidden="true" focusable="false" width="100%" height="100%" viewBox="0 0 200 200" version="1.1" xmlns="http://www.w3.org/2000/svg">`)
	b.WriteString(`<circle r="90" cx="100" cy="100" fill="transparent" stroke-dasharray="565.48" stroke-dashoffset="0"></circle>`)
	b.WriteString(`<circle id="bar" r="90" cx="100" cy="100" fill="transparent" stroke-dasharray="565.48" stroke-dashoffset="0"></circle>`)
	b.WriteString(`</svg></div>`)
	b.WriteString(`<div class="site-health-progress-label">`)
	if !hasResults {
		b.WriteString(`No information yet&hellip;`)
	} else if issuesTotal <= 0 {
		b.WriteString(`Good`)
	} else if counts["critical"] > 0 {
		b.WriteString(`Should be improved`)
	} else {
		b.WriteString(`Could be improved`)
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="site-health-details"><p>`)
	if !hasResults {
		b.WriteString(`Site health checks will automatically run periodically to gather information about your site. You can also <a href="/wp-admin/site-health.php">visit the Site Health screen</a> to gather information about your site now.`)
	} else if issuesTotal <= 0 {
		b.WriteString(`Great job! Your site currently passes all site health checks.`)
	} else if counts["critical"] == 1 {
		b.WriteString(`Your site has a critical issue that should be addressed as soon as possible to improve its performance and security.`)
	} else if counts["critical"] > 1 {
		b.WriteString(`Your site has critical issues that should be addressed as soon as possible to improve its performance and security.`)
	} else if counts["recommended"] == 1 {
		b.WriteString(`Your site&#8217;s health is looking good, but there is still one thing you can do to improve its performance and security.`)
	} else {
		b.WriteString(`Your site&#8217;s health is looking good, but there are still some things you can do to improve its performance and security.`)
	}
	b.WriteString(`</p>`)
	if hasResults && issuesTotal > 0 {
		itemWord := "items"
		if issuesTotal == 1 {
			itemWord = "item"
		}
		b.WriteString(`<p>Take a look at the <strong>`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%d", issuesTotal)))
		b.WriteString(` `)
		b.WriteString(itemWord)
		b.WriteString(`</strong> on the <a href="/wp-admin/site-health.php">Site Health screen</a>.</p>`)
	}
	b.WriteString(`</div></div></div></div>`)
	return b.String()
}

// siteHealthIssueCounts reads the health-check-site-status-result transient.
// Returns nil when no results have been stored yet.
func siteHealthIssueCounts(c *wp.Context) map[string]int {
	if c == nil {
		return nil
	}
	raw, ok := c.GetTransient("health-check-site-status-result")
	if !ok || raw == nil {
		return nil
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil || len(decoded) == 0 {
		return nil
	}
	out := map[string]int{
		"good":        jsonInt(decoded["good"]),
		"recommended": jsonInt(decoded["recommended"]),
		"critical":    jsonInt(decoded["critical"]),
	}
	return out
}

func jsonInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		var n int
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}
