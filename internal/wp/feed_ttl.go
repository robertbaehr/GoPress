package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// FeedTTLMinutes returns the RSS channel <ttl> value in minutes (P3.57).
// WordPress core does not emit <ttl> by default; when the "rss_ttl" option is a
// positive integer, GoPress emits it. Non-numeric / empty / zero → "".
func FeedTTLMinutes(c *Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	raw := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("rss_ttl", "")))
	if raw == "" || raw == "<nil>" {
		return ""
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}
