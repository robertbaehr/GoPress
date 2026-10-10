// get_bloginfo('language') subset (P3.53) — used by bloginfo_rss('language').
package wp

import (
	"fmt"
	"strings"
)

// BlogInfoLanguage returns the site language tag for feeds/HTML (get_bloginfo('language')).
// Uses Locale.Code when set, else WPLANG, else en_US; underscores become hyphens.
func BlogInfoLanguage(c *Context) string {
	loc := ""
	if c != nil && c.Locale != nil {
		loc = strings.TrimSpace(c.Locale.Code)
	}
	if loc == "" && c != nil && c.Options != nil {
		v := c.Options.GetOption("WPLANG", "")
		switch t := v.(type) {
		case string:
			loc = strings.TrimSpace(t)
		case nil:
			loc = ""
		default:
			s := strings.TrimSpace(fmt.Sprint(t))
			if s != "" && s != "<nil>" {
				loc = s
			}
		}
	}
	if loc == "" {
		loc = "en_US"
	}
	return strings.ReplaceAll(loc, "_", "-")
}
