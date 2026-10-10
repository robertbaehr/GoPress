package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// User is a minimal wp/v2 user object (users/me + collection).
type User struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	URL         string         `json:"url"`
	Description string         `json:"description"`
	Link        string         `json:"link"`
	Slug        string         `json:"slug"`
	Username    string         `json:"username,omitempty"`
	Email       string         `json:"email,omitempty"`
	Roles       []string       `json:"roles,omitempty"`
	Meta        []any          `json:"meta,omitempty"`
	Links       map[string]any `json:"_links,omitempty"`
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
	// WP advertises Allow even on rest_not_logged_in (P10.15 corpus).
	w.Header().Set("Allow", "GET")
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
		Meta:        []any{},
	}
	if len(u.Roles) > 0 {
		out.Roles = append([]string{}, u.Roles...)
	}
	return out
}

func buildListedUser(c *wp.Context, u *wp.User, canList bool, context string) User {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := ""
	if c != nil && c.Meta != nil {
		desc = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "description", true))
	}
	slug := u.Nicename
	if slug == "" {
		slug = u.Login
	}
	out := User{
		ID:          u.ID,
		Name:        u.DisplayName,
		URL:         u.URL,
		Description: desc,
		Link:        fmt.Sprintf("%s/author/%s/", home, slug),
		Slug:        slug,
		Meta:        []any{},
		Links: map[string]any{
			"self": []map[string]any{{
				"href":        fmt.Sprintf("%s/wp-json/wp/v2/users/%d", home, u.ID),
				"targetHints": map[string]any{"allow": []string{"GET"}},
			}},
			"collection": []map[string]any{{
				"href": fmt.Sprintf("%s/wp-json/wp/v2/users", home),
			}},
		},
	}
	if canList {
		if len(u.Roles) > 0 {
			out.Roles = append([]string{}, u.Roles...)
		}
		if context == "edit" {
			out.Username = u.Login
			out.Email = u.Email
		}
	}
	return out
}

func countUsers(c *wp.Context, publishedOnly bool) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	q := "SELECT COUNT(*) AS total FROM " + c.DB.Users() + " u"
	if publishedOnly {
		q += " WHERE EXISTS (SELECT 1 FROM " + c.DB.Posts() +
			" p WHERE p.post_author = u.ID AND p.post_status = 'publish' AND p.post_type IN ('post','page'))"
	}
	sqlStr, args, err := c.DB.Prepare(q)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

func listUsersPaged(c *wp.Context, limit, page int, publishedOnly bool) ([]*wp.User, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	q := "SELECT u.ID, u.user_login, u.user_pass, u.user_nicename, u.user_email, u.user_url, " +
		"u.user_registered, u.user_activation_key, u.user_status, u.display_name FROM " + c.DB.Users() + " u"
	if publishedOnly {
		q += " WHERE EXISTS (SELECT 1 FROM " + c.DB.Posts() +
			" p WHERE p.post_author = u.ID AND p.post_status = 'publish' AND p.post_type IN ('post','page'))"
	}
	q += " ORDER BY u.display_name ASC LIMIT %d OFFSET %d"
	sqlStr, args, err := c.DB.Prepare(q, limit, offset)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	out := make([]*wp.User, 0, len(rows))
	for _, row := range rows {
		u := &wp.User{
			ID:            asInt64(row["ID"]),
			Login:         asString(row["user_login"]),
			Pass:          asString(row["user_pass"]),
			Nicename:      asString(row["user_nicename"]),
			Email:         asString(row["user_email"]),
			URL:           asString(row["user_url"]),
			Registered:    asString(row["user_registered"]),
			ActivationKey: asString(row["user_activation_key"]),
			Status:        int(asInt64(row["user_status"])),
			DisplayName:   asString(row["display_name"]),
		}
		if c.Roles != nil {
			_ = c.Roles.InitUser(u, c.Meta)
		}
		out = append(out, u)
	}
	return out, nil
}
