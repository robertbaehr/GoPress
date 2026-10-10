package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// settingsOptionMap maps REST setting keys to wp_options names (P4.40).
// Keys match WP show_in_rest names plus the raw option aliases we already expose.
var settingsOptionMap = map[string]string{
	"title":           "blogname",
	"description":     "blogdescription",
	"blogname":        "blogname",
	"blogdescription": "blogdescription",
	"timezone":        "timezone_string",
	"date_format":     "date_format",
	"time_format":     "time_format",
	"posts_per_page":  "posts_per_page",
}

func (s *Server) serveSettings(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r, c)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateSettings(w, r, c)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if !canManageOptions(c) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_forbidden",
			"message": "Sorry, you are not allowed to do that.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH")
	writeJSON(w, http.StatusOK, buildSettings(c))
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if !canManageOptions(c) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_forbidden",
			"message": "Sorry, you are not allowed to do that.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	params, err := decodeSettingsBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	for key, val := range params {
		opt, ok := settingsOptionMap[key]
		if !ok {
			continue
		}
		serialized := settingsValueString(val)
		if _, err := c.Options.UpdateOption(opt, serialized, ""); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_settings_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH")
	writeJSON(w, http.StatusOK, buildSettings(c))
}

func canManageOptions(c *wp.Context) bool {
	return c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "manage_options")
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

func decodeSettingsBody(r *http.Request) (map[string]any, error) {
	out := map[string]any{}
	if r == nil || r.Body == nil {
		return out, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON body")
	}
	return out, nil
}

func settingsValueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return "0"
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
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
