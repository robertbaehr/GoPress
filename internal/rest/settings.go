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

// settingsOptionMap maps REST setting keys to wp_options names (P4.40 / P4.63 / P4.123).
// Keys match WP show_in_rest names plus the raw option aliases we already expose.
var settingsOptionMap = map[string]string{
	"title":                   "blogname",
	"description":             "blogdescription",
	"blogname":                "blogname",
	"blogdescription":         "blogdescription",
	"url":                     "siteurl",
	"email":                   "admin_email",
	"timezone":                "timezone_string",
	"date_format":             "date_format",
	"time_format":             "time_format",
	"start_of_week":           "start_of_week",
	"language":                "WPLANG",
	"use_smilies":             "use_smilies",
	"default_category":        "default_category",
	"default_post_format":     "default_post_format",
	"posts_per_page":          "posts_per_page",
	"show_on_front":           "show_on_front",
	"page_on_front":           "page_on_front",
	"page_for_posts":          "page_for_posts",
	"page_for_privacy_policy": "wp_page_for_privacy_policy",
	"default_ping_status":     "default_ping_status",
	"default_comment_status":  "default_comment_status",
	"site_logo":               "site_logo",
	"site_icon":               "site_icon",
}

// settingsReadonly keys cannot be updated via REST (P4.123; matches WP url readonly).
var settingsReadonly = map[string]bool{
	"url":     true,
	"siteurl": true,
	"home":    true,
}

func (s *Server) serveSettings(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodOptions:
		// P4.123: OPTIONS returns public item schema (like WP_REST_Settings_Controller).
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, HEAD, OPTIONS")
		writeJSON(w, http.StatusOK, settingsItemSchema())
		return
	case http.MethodGet, http.MethodHead:
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
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, HEAD, OPTIONS")
	// P4.123: HEAD returns early with empty body.
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
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
		if settingsReadonly[key] {
			continue
		}
		opt, ok := settingsOptionMap[key]
		if !ok {
			continue
		}
		// P4.123: rest_pre_update_setting may short-circuit (return true → skip).
		if c != nil && c.Hooks != nil {
			pre := c.Hooks.ApplyFilters("rest_pre_update_setting", false, key, val, opt)
			if b, ok := pre.(bool); ok && b {
				continue
			}
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
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, HEAD, OPTIONS")
	writeJSON(w, http.StatusOK, buildSettings(c))
}

func canManageOptions(c *wp.Context) bool {
	return c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "manage_options")
}

// buildSettings returns the WP REST settings object shape (P4.63 / P4.123).
func buildSettings(c *wp.Context) map[string]any {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	lang := optionString(c, "WPLANG", "")
	if lang == "" {
		lang = "en_US"
	}
	siteLogo := optionInt64(c, "site_logo", 0)
	var siteLogoVal any
	if siteLogo > 0 {
		siteLogoVal = siteLogo
	} else {
		siteLogoVal = nil
	}
	raw := map[string]any{
		// WP show_in_rest names.
		"title":                   optionString(c, "blogname", ""),
		"description":             optionString(c, "blogdescription", ""),
		"url":                     siteURL,
		"email":                   optionString(c, "admin_email", ""),
		"timezone":                optionString(c, "timezone_string", ""),
		"date_format":             optionString(c, "date_format", "F j, Y"),
		"time_format":             optionString(c, "time_format", "g:i a"),
		"start_of_week":           optionInt(c, "start_of_week", 1),
		"language":                lang,
		"use_smilies":             optionBool(c, "use_smilies", true),
		"default_category":        optionInt(c, "default_category", 1),
		"default_post_format":     optionString(c, "default_post_format", "0"),
		"posts_per_page":          optionInt(c, "posts_per_page", 10),
		"show_on_front":           optionString(c, "show_on_front", "posts"),
		"page_on_front":           optionInt(c, "page_on_front", 0),
		"page_for_posts":          optionInt(c, "page_for_posts", 0),
		"page_for_privacy_policy": optionInt(c, "wp_page_for_privacy_policy", 0),
		"default_ping_status":     optionString(c, "default_ping_status", "open"),
		"default_comment_status":  optionString(c, "default_comment_status", "open"),
		"site_logo":               siteLogoVal,
		"site_icon":               optionInt(c, "site_icon", 0),
		// GoPress aliases retained for older clients.
		"blogname":        optionString(c, "blogname", ""),
		"blogdescription": optionString(c, "blogdescription", ""),
		"siteurl":         siteURL,
		"home":            home,
	}
	// P4.123: rest_pre_get_setting may override individual values.
	if c != nil && c.Hooks != nil {
		for name := range settingsOptionMap {
			if _, ok := raw[name]; !ok {
				continue
			}
			filtered := c.Hooks.ApplyFilters("rest_pre_get_setting", nil, name, map[string]any{
				"option_name": settingsOptionMap[name],
			})
			if filtered != nil {
				raw[name] = filtered
			}
		}
	}
	return raw
}

// settingsItemSchema is the public REST schema for /wp/v2/settings (P4.123).
func settingsItemSchema() map[string]any {
	ctx := []string{"view", "edit"}
	prop := func(desc, typ string, extra ...map[string]any) map[string]any {
		m := map[string]any{
			"description": desc,
			"type":        typ,
			"context":     ctx,
		}
		for _, e := range extra {
			for k, v := range e {
				m[k] = v
			}
		}
		return m
	}
	return map[string]any{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title":   "settings",
		"type":    "object",
		"properties": map[string]any{
			"title":                   prop("Site title.", "string"),
			"description":             prop("Site tagline.", "string"),
			"url":                     prop("Site URL.", "string", map[string]any{"format": "uri", "readonly": true}),
			"email":                   prop("This address is used for admin purposes.", "string", map[string]any{"format": "email"}),
			"timezone":                prop("A city in the same timezone as you.", "string"),
			"date_format":             prop("A date format for all date strings.", "string"),
			"time_format":             prop("A time format for all time strings.", "string"),
			"start_of_week":           prop("A day number of the week that the week should start on.", "integer"),
			"language":                prop("WordPress locale code.", "string"),
			"use_smilies":             prop("Convert emoticons like :-) and :-P to graphics on display.", "boolean"),
			"default_category":        prop("Default post category.", "integer"),
			"default_post_format":     prop("Default post format.", "string"),
			"posts_per_page":          prop("Blog pages show at most.", "integer"),
			"show_on_front":           prop("What to show on the front page", "string"),
			"page_on_front":           prop("The ID of the page that should be displayed on the front page", "integer"),
			"page_for_posts":          prop("The ID of the page that should display the latest posts", "integer"),
			"page_for_privacy_policy": prop("The ID of the page that should display the privacy policy", "integer"),
			"default_ping_status":     prop("Allow link notifications from other blogs (pingbacks and trackbacks) on new articles.", "string"),
			"default_comment_status":  prop("Allow people to submit comments on new posts.", "string"),
			"site_logo":               prop("Site logo media attachment ID.", "integer"),
			"site_icon":               prop("Site icon.", "integer"),
		},
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

func optionBool(c *wp.Context, name string, fallback bool) bool {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	case []byte:
		s := strings.TrimSpace(strings.ToLower(string(t)))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	default:
		return fallback
	}
}
