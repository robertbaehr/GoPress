package rest

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Widgets are out of scope for GoPress (P4.18). Collection routes return empty
// arrays; item and write routes report rest_widgets_disabled.

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
	if r.Method != http.MethodGet {
		s.serveWidgetsDisabled(w, r, c)
		return
	}
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, []any{})
}

func (s *Server) serveWidgetsDisabled(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	_ = c
	writeJSON(w, http.StatusForbidden, map[string]any{
		"code":    "rest_widgets_disabled",
		"message": "Widgets are out of scope for GoPress.",
		"data":    map[string]any{"status": 403},
	})
}
