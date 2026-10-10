package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// User is a minimal wp/v2 user object (users/me).
type User struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
	Slug        string   `json:"slug"`
	Username    string   `json:"username,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

func (s *Server) serveUsersMe(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_not_logged_in",
			"message": "You are not currently logged in.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	writeJSON(w, http.StatusOK, buildCurrentUser(c))
}

func buildCurrentUser(c *wp.Context) User {
	u := c.CurrentUser
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := ""
	if c.Meta != nil {
		desc = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "description", true))
	}
	out := User{
		ID:          u.ID,
		Name:        u.DisplayName,
		URL:         u.URL,
		Description: desc,
		Link:        fmt.Sprintf("%s/author/%s/", home, u.Nicename),
		Slug:        u.Nicename,
		Username:    u.Login,
		Email:       u.Email,
	}
	if len(u.Roles) > 0 {
		out.Roles = append([]string{}, u.Roles...)
	}
	return out
}
