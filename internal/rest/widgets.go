package rest

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Widgets and sidebars are out of scope for GoPress (P4.18 / P4.54 / P4.102 / P4.104).
// Collection GET for users with edit_theme_options returns []; anonymous and
// unauthorized callers get rest_cannot_manage_widgets (WP permission parity).
// Widget/sidebar/widget-type item GETs return invalid_id / invalid type (404);
// writes remain rest_widgets_disabled once authorized.

func (s *Server) serveWidgets(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveWidgetItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	s.serveWidgetsItemOrWrite(w, r, c)
}

func (s *Server) serveWidgetTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveWidgetTypeItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	// P4.104: no classic widget types are registered (widgets out of scope).
	// Authorized GET/HEAD → rest_widget_type_invalid (404); writes stay disabled.
	if !canManageWidgets(c) {
		s.serveWidgetsCannotManage(w, r, c)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		w.Header().Set("Allow", "GET")
		w.Header().Set("X-GoPress-Widgets", "disabled")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_widget_type_invalid",
			"message": "Invalid widget type.",
			"data":    map[string]any{"status": 404},
		})
	case http.MethodOptions:
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusOK)
	default:
		s.serveWidgetsDisabled(w, r, c)
	}
}

func (s *Server) serveSidebars(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.103: sidebars collection — same permission gate as widgets; empty when authorized.
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveSidebarItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	// P4.103: no classic sidebars are registered (widgets out of scope).
	// Authorized GET/HEAD → rest_sidebar_invalid_id (404); writes stay disabled.
	if !canManageWidgets(c) {
		s.serveWidgetsCannotManage(w, r, c)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		w.Header().Set("Allow", "GET")
		w.Header().Set("X-GoPress-Widgets", "disabled")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_sidebar_invalid_id",
			"message": "No sidebar exists with that id.",
			"data":    map[string]any{"status": 404},
		})
	case http.MethodOptions:
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusOK)
	default:
		s.serveWidgetsDisabled(w, r, c)
	}
}

func (s *Server) serveWidgetsGoneCollection(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !canManageWidgets(c) {
			s.serveWidgetsCannotManage(w, r, c)
			return
		}
		w.Header().Set("Allow", "GET")
		w.Header().Set("X-GoPress-Widgets", "disabled")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeJSON(w, http.StatusOK, []any{})
	case http.MethodOptions:
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusOK)
	default:
		s.serveWidgetsItemOrWrite(w, r, c)
	}
}

func (s *Server) serveWidgetsItemOrWrite(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if !canManageWidgets(c) {
		s.serveWidgetsCannotManage(w, r, c)
		return
	}
	s.serveWidgetsDisabled(w, r, c)
}

func canManageWidgets(c *wp.Context) bool {
	return c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_theme_options")
}

func (s *Server) serveWidgetsCannotManage(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	w.Header().Set("X-GoPress-Widgets", "disabled")
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_manage_widgets",
		"message": "Sorry, you are not allowed to manage widgets on this site.",
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) serveWidgetsDisabled(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	_ = c
	w.Header().Set("X-GoPress-Widgets", "disabled")
	writeJSON(w, http.StatusForbidden, map[string]any{
		"code":    "rest_widgets_disabled",
		"message": "Widgets are out of scope for GoPress.",
		"data":    map[string]any{"status": 403},
	})
}
