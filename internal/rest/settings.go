package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func (s *Server) serveSettings(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "manage_options") {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_forbidden",
			"message": "Sorry, you are not allowed to do that.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	writeJSON(w, http.StatusOK, buildSettings(c))
}

func buildSettings(c *wp.Context) map[string]any {
	postsPerPage := optionInt(c, "posts_per_page", 10)
	return map[string]any{
		"blogname":        optionString(c, "blogname", ""),
		"blogdescription": optionString(c, "blogdescription", ""),
		"siteurl":         strings.TrimRight(optionString(c, "siteurl", ""), "/"),
		"home":            strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/"),
		"timezone":        optionString(c, "timezone_string", ""),
		"date_format":     optionString(c, "date_format", "F j, Y"),
		"time_format":     optionString(c, "time_format", "g:i a"),
		"posts_per_page":  postsPerPage,
		// WP REST aliases commonly consumed by clients.
		"title":       optionString(c, "blogname", ""),
		"description": optionString(c, "blogdescription", ""),
		"url":         strings.TrimRight(optionString(c, "siteurl", ""), "/"),
	}
}

func optionInt(c *wp.Context, name string, fallback int) int {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return fallback
		}
		return n
	case []byte:
		n, err := strconv.Atoi(strings.TrimSpace(string(t)))
		if err != nil {
			return fallback
		}
		return n
	default:
		n, err := strconv.Atoi(strings.TrimSpace(asString(v)))
		if err != nil {
			return fallback
		}
		return n
	}
}
