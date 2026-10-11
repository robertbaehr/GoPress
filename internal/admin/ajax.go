package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveAdminAjax implements a minimal /wp-admin/admin-ajax.php (P7.2).
func (h *Handler) serveAdminAjax(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	// P2.66: DOING_AJAX / wp_doing_ajax for the duration of admin-ajax.php.
	if c.Scope != nil {
		c.Scope.SetAjax(true)
	}
	c.AuthenticateRequest()

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate, max-age=0")

	action := r.URL.Query().Get("action")
	if action == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		action = r.FormValue("action")
	}
	if action == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
		return
	}

	loggedIn := c.CurrentUser != nil

	switch action {
	case "heartbeat":
		h.ajaxHeartbeat(w, r, c, loggedIn)
	case "wp-auth-check":
		// Dedicated auth-check probe (P2.19); mirrors heartbeat's wp-auth-check key.
		h.ajaxAuthCheck(w, r, c)
	case "rest-nonce":
		if !loggedIn {
			// Privileged action: WP dies with "0" for unknown nopriv actions.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("0"))
			return
		}
		// WP wp_ajax_rest_nonce exits raw nonce with default admin-ajax Content-Type (text/html).
		nonce := c.CreateNonce("wp_rest")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(nonce))
	case "logged-in":
		if !loggedIn {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("0"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("1"))
	case "hidden-columns":
		// P7.80: wp_ajax_hidden_columns — persist manage{page}columnshidden.
		h.ajaxHiddenColumns(w, r, c, loggedIn)
	case "update-welcome-panel":
		// P7.135: wp_ajax_update_welcome_panel.
		h.ajaxUpdateWelcomePanel(w, r, c, loggedIn)
	default:
		// Unknown / privileged-without-nopriv → "0" (WP admin-ajax.php).
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
	}
}

func (h *Handler) ajaxHeartbeat(w http.ResponseWriter, r *http.Request, c *wp.Context, loggedIn bool) {
	// P3.96: screen_id + heartbeat_* / heartbeat_nopriv_* filters/actions + wp-auth-check.
	_ = r.ParseForm()
	screenID := sanitizeKey(strings.TrimSpace(r.Form.Get("screen_id")))
	if screenID == "" {
		screenID = "front"
	}
	data := heartbeatDataFromForm(r)

	resp := map[string]any{}
	if c != nil && c.Hooks != nil {
		if loggedIn {
			if len(data) > 0 {
				if v := c.Hooks.ApplyFilters("heartbeat_received", resp, data, screenID); v != nil {
					if m, ok := v.(map[string]any); ok {
						resp = m
					}
				}
			}
			if v := c.Hooks.ApplyFilters("heartbeat_send", resp, screenID); v != nil {
				if m, ok := v.(map[string]any); ok {
					resp = m
				}
			}
			c.Hooks.DoAction("heartbeat_tick", resp, screenID)
		} else {
			if len(data) > 0 {
				if v := c.Hooks.ApplyFilters("heartbeat_nopriv_received", resp, data, screenID); v != nil {
					if m, ok := v.(map[string]any); ok {
						resp = m
					}
				}
			}
			if v := c.Hooks.ApplyFilters("heartbeat_nopriv_send", resp, screenID); v != nil {
				if m, ok := v.(map[string]any); ok {
					resp = m
				}
			}
			c.Hooks.DoAction("heartbeat_nopriv_tick", resp, screenID)
		}
	}
	// Mirror wp_auth_check on heartbeat_send / heartbeat_nopriv_send.
	if c != nil {
		resp["wp-auth-check"] = c.AuthCheck()
	} else {
		resp["wp-auth-check"] = false
	}
	resp["server_time"] = time.Now().Unix()

	body, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("0"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// heartbeatDataFromForm collects POST data[...] keys into a flat map (P3.96).
func heartbeatDataFromForm(r *http.Request) map[string]any {
	if r == nil || r.Form == nil {
		return nil
	}
	out := map[string]any{}
	for key, vals := range r.Form {
		if !strings.HasPrefix(key, "data[") || !strings.HasSuffix(key, "]") {
			continue
		}
		inner := key[len("data[") : len(key)-1]
		if inner == "" {
			continue
		}
		if len(vals) == 1 {
			out[inner] = vals[0]
		} else {
			out[inner] = vals
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (h *Handler) ajaxAuthCheck(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	resp := struct {
		WPAuthCheck bool `json:"wp-auth-check"`
	}{
		WPAuthCheck: c.AuthCheck(),
	}
	body, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("0"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// ajaxHiddenColumns mirrors wp_ajax_hidden_columns (P7.80).
func (h *Handler) ajaxHiddenColumns(w http.ResponseWriter, r *http.Request, c *wp.Context, loggedIn bool) {
	if !loggedIn || c.CurrentUser == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
		return
	}
	_ = r.ParseForm()
	if c.VerifyNonce(r.Form.Get("screenoptionnonce"), "screen-options-nonce") == 0 {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("-1"))
		return
	}
	page := strings.TrimSpace(r.Form.Get("page"))
	if page == "" || page != sanitizeKey(page) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
		return
	}
	var hidden []string
	if raw := strings.TrimSpace(r.Form.Get("hidden")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				hidden = append(hidden, part)
			}
		}
	}
	arr := map[any]any{}
	for i, col := range hidden {
		arr[i] = col
	}
	serialized, err := wp.SerializePHP(arr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("0"))
		return
	}
	if err := wp.UpdateUserOption(c, c.CurrentUser.ID, "manage"+page+"columnshidden", serialized); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("0"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("1"))
}

func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}
