package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// User is a minimal wp/v2 user object (users/me + collection).
type User struct {
	ID                int64             `json:"id"`
	Name              string            `json:"name"`
	URL               string            `json:"url"`
	Description       string            `json:"description"`
	Link              string            `json:"link"`
	Slug              string            `json:"slug"`
	Username          string            `json:"username,omitempty"`
	Email             string            `json:"email,omitempty"`
	FirstName         string            `json:"first_name,omitempty"`
	LastName          string            `json:"last_name,omitempty"`
	Nickname          string            `json:"nickname,omitempty"`
	Locale            string            `json:"locale,omitempty"`
	RegisteredDate    string            `json:"registered_date,omitempty"`
	Roles             []string          `json:"roles,omitempty"`
	Capabilities      map[string]bool   `json:"capabilities,omitempty"`
	ExtraCapabilities map[string]bool   `json:"extra_capabilities,omitempty"`
	AvatarURLs        map[string]string `json:"avatar_urls,omitempty"`
	Meta              []any             `json:"meta,omitempty"`
	Links             map[string]any    `json:"_links,omitempty"`
}

func (s *Server) serveUsersMe(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.99: WP advertises GET + write methods on /users/me.
	w.Header().Set("Allow", "GET, POST, PUT, PATCH")
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_not_logged_in",
			"message": "You are not currently logged in.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, buildCurrentUser(c))
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateCurrentUser(w, r, c)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func buildCurrentUser(c *wp.Context) User {
	u := c.CurrentUser
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	slug := u.Nicename
	if slug == "" {
		slug = u.Login
	}
	desc := ""
	firstName, lastName, nickname, locale := "", "", "", ""
	if c.Meta != nil {
		desc = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "description", true))
		firstName = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "first_name", true))
		lastName = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "last_name", true))
		nickname = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "nickname", true))
		locale = asString(c.Meta.GetMetadata(wp.MetaUser, u.ID, "locale", true))
	}
	out := User{
		ID:                u.ID,
		Name:              u.DisplayName,
		URL:               u.URL,
		Description:       desc,
		Link:              fmt.Sprintf("%s/author/%s/", home, slug),
		Slug:              slug,
		Username:          u.Login,
		Email:             u.Email,
		FirstName:         firstName,
		LastName:          lastName,
		Nickname:          nickname,
		Locale:            locale,
		RegisteredDate:    registeredDateISO(u.Registered),
		AvatarURLs:        gravatarAvatarURLs(u.Email),
		Capabilities:      copyBoolMap(u.AllCaps),
		ExtraCapabilities: copyBoolMap(u.Caps),
		Meta:              []any{},
		Links:             buildUserMeLinks(home, u.ID),
	}
	if len(u.Roles) > 0 {
		out.Roles = append([]string{}, u.Roles...)
	}
	return out
}

func buildUserMeLinks(home string, id int64) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	return map[string]any{
		"self": []map[string]any{{
			"href":        fmt.Sprintf("%s/users/%d", restRoot, id),
			"targetHints": map[string]any{"allow": []string{"GET", "POST", "PUT", "PATCH", "DELETE"}},
		}},
		"collection": []map[string]any{{
			"href": restRoot + "/users",
		}},
	}
}

func copyBoolMap(in map[string]bool) map[string]bool {
	if len(in) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func registeredDateISO(mysql string) string {
	mysql = strings.TrimSpace(mysql)
	if mysql == "" {
		return ""
	}
	// wp_users.user_registered is "YYYY-MM-DD HH:MM:SS" UTC → RFC3339-ish.
	if len(mysql) >= 19 && mysql[10] == ' ' {
		return mysql[:10] + "T" + mysql[11:19] + "+00:00"
	}
	return mysql
}

// updateCurrentUser handles POST/PUT/PATCH /wp/v2/users/me (P4.99).
func (s *Server) updateCurrentUser(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	id := c.CurrentUser.ID
	fields, err := decodeUserWriteFields(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	upd := wp.UserUpdate{ID: id}
	if v, ok := fields["email"].(string); ok {
		upd.Email = &v
	}
	if v, ok := fields["url"].(string); ok {
		upd.URL = &v
	}
	if v, ok := fields["name"].(string); ok {
		upd.DisplayName = &v
	}
	if v, ok := fields["slug"].(string); ok {
		upd.Nicename = &v
	}
	if v, ok := fields["password"].(string); ok {
		upd.Pass = &v
	}
	if v, ok := fields["description"].(string); ok {
		upd.Description = &v
	}
	if err := c.UpdateUser(upd); err != nil {
		code := "rest_user_update"
		status := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "email exists") {
			code, status, msg = "rest_user_invalid_email", http.StatusBadRequest, "Sorry, that email address is already used!"
		}
		writeJSON(w, status, map[string]any{
			"code":    code,
			"message": msg,
			"data":    map[string]any{"status": status},
		})
		return
	}
	// Refresh usermeta fields that UpdateUser may have written.
	if first, ok := fields["first_name"].(string); ok && c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, id, "first_name", first, "")
	}
	if last, ok := fields["last_name"].(string); ok && c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, id, "last_name", last, "")
	}
	if nick, ok := fields["nickname"].(string); ok && c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, id, "nickname", nick, "")
	}
	u, err := c.GetUserBy("id", id)
	if err != nil || u == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_update",
			"message": "updated but reload failed",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if c.Roles != nil {
		_ = c.Roles.InitUser(u, c.Meta)
	}
	c.CurrentUser = u
	writeJSON(w, http.StatusOK, buildCurrentUser(c))
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
