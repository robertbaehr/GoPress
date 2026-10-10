package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FeedCopyright returns channel copyright text for RSS2/Atom (P3.54).
// WordPress core emits no copyright/rights by default; when the
// "rss_copyright" option is set:
//   - "auto" → "Copyright {year} {blogname}" (year from when, or now UTC)
//   - any other non-empty value → used as-is
func FeedCopyright(c *Context, when time.Time) string {
	if c == nil || c.Options == nil {
		return ""
	}
	raw := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("rss_copyright", "")))
	if raw == "" || raw == "<nil>" {
		return ""
	}
	if !strings.EqualFold(raw, "auto") {
		return raw
	}
	blogname := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blogname", "")))
	if blogname == "<nil>" {
		blogname = ""
	}
	if blogname == "" {
		blogname = "WordPress"
	}
	if when.IsZero() {
		when = time.Now().UTC()
	}
	return "Copyright " + strconv.Itoa(when.Year()) + " " + blogname
}
