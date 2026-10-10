package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveAdminAjax implements a minimal /wp-admin/admin-ajax.php (P7.2).
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
	_ = c
	_ = r
	// Match wp_ajax_heartbeat / wp_ajax_nopriv_heartbeat + wp_auth_check key order.
	resp := struct {
		WPAuthCheck bool  `json:"wp-auth-check"`
		ServerTime  int64 `json:"server_time"`
	}{
		WPAuthCheck: loggedIn,
		ServerTime:  time.Now().Unix(),
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
