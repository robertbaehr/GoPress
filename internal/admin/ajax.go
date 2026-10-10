package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveAdminAjax implements a minimal /wp-admin/admin-ajax.php (P7.2 / P7.17).
func (h *Handler) serveAdminAjax(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
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
	default:
		// Unknown / privileged-without-nopriv → "0" (WP admin-ajax.php).
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
	}
}

func (h *Handler) ajaxHeartbeat(w http.ResponseWriter, r *http.Request, c *wp.Context, loggedIn bool) {
	_ = r.ParseForm()
	resp := map[string]any{}

	if loggedIn {
		nonce := r.FormValue("_nonce")
		if nonce == "" {
			// wp_ajax_heartbeat → wp_send_json_error() when _nonce missing.
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":false}`))
			return
		}
		if !verifyHeartbeatNonce(c, nonce) {
			resp["nonces_expired"] = true
			resp["wp-auth-check"] = true
			resp["server_time"] = time.Now().Unix()
			// Refresh transport nonces (wp_refresh_nonces subset).
			resp["heartbeat-nonce"] = c.CreateNonce("heartbeat-nonce")
			resp["rest-nonce"] = c.CreateNonce("wp_rest")
			writeJSON(w, resp)
			return
		}
	}

	// wp_auth_check filter: logged-in and not in grace period.
	resp["wp-auth-check"] = loggedIn
	resp["server_time"] = time.Now().Unix()
	writeJSON(w, resp)
}

func verifyHeartbeatNonce(c *wp.Context, nonce string) bool {
	if c == nil || nonce == "" {
		return false
	}
	// Accept current or previous tick (wp_verify_nonce 1 / 2).
	if c.CreateNonce("heartbeat-nonce") == nonce {
		return true
	}
	// Previous tick: recreate with age by temporarily checking Auth ticks.
	if c.Auth == nil {
		return false
	}
	uid := 0
	token := ""
	if c.CurrentUser != nil {
		uid = int(c.CurrentUser.ID)
		token = c.SessionToken
	}
	tick := wp.NonceTick(0, "heartbeat-nonce")
	if tick > 1 {
		prev := c.Auth.CreateNonce(tick-1, "heartbeat-nonce", uid, token)
		if prev == nonce {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("0"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
