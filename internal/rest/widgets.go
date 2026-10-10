package rest

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Widgets and sidebars are out of scope for GoPress (P4.18 / P4.54).
// Collection GET routes return []; every other method and all item routes
// respond with rest_widgets_disabled so classic widget APIs stay cleanly off.

func (s *Server) serveWidgets(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveWidgetItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	s.serveWidgetsDisabled(w, r, c)
}

func (s *Server) serveWidgetTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveWidgetTypeItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	s.serveWidgetsDisabled(w, r, c)
}

func (s *Server) serveSidebars(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveWidgetsGoneCollection(w, r, c)
}

func (s *Server) serveSidebarItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	_ = id
	s.serveWidgetsDisabled(w, r, c)
}

func (s *Server) serveWidgetsGoneCollection(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = c
	switch r.Method {
	case http.MethodGet, http.MethodHead:
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
		s.serveWidgetsDisabled(w, r, c)
	}
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
