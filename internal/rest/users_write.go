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

type userWriteBody struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Slug        string   `json:"slug"`
	Roles       []string `json:"roles"`
}

func (s *Server) serveUsers(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		s.listUsers(w, r, c)
	case http.MethodPost:
		s.createUser(w, r, c)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "create_users") {
		w.Header().Set("Allow", "GET, POST")
	} else {
		w.Header().Set("Allow", "GET")
	}
	canList := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "list_users")
	context := strings.TrimSpace(r.URL.Query().Get("context"))
	if context == "" {
		context = "view"
	}
	if context == "edit" && !canList {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_forbidden_context",
			"message": "Sorry, you are not allowed to edit users.",
			"data":    map[string]any{"status": status},
		})
		return
	}

	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countUsers(c, !canList)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	users, err := listUsersPaged(c, perPage, page, !canList)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]User, 0, len(users))
	for _, u := range users {
		out = append(out, buildListedUser(c, u, canList, context))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create_user",
			"message": "Sorry, you are not allowed to create new users.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "create_users") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create_user",
			"message": "Sorry, you are not allowed to create new users.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodeUserWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	username := strings.TrimSpace(body.Username)
	email := strings.TrimSpace(body.Email)
	if username == "" || email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): username, email",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if strings.TrimSpace(body.Password) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_user_invalid_password",
			"message": "Passwords cannot be empty.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	role := "subscriber"
	if len(body.Roles) > 0 && strings.TrimSpace(body.Roles[0]) != "" {
		role = strings.TrimSpace(body.Roles[0])
	}
	id, err := c.InsertUser(wp.UserInsert{
		Login:       username,
		Pass:        body.Password,
		Email:       email,
		URL:         strings.TrimSpace(body.URL),
		Nicename:    strings.TrimSpace(body.Slug),
		DisplayName: strings.TrimSpace(body.Name),
		Role:        role,
		Description: strings.TrimSpace(body.Description),
	})
	if err != nil {
		code := "rest_user_create"
		status := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "login exists") {
			code, status, msg = "rest_user_invalid_username", http.StatusBadRequest, "Sorry, that username already exists!"
		} else if strings.Contains(msg, "email exists") {
			code, status, msg = "rest_user_invalid_email", http.StatusBadRequest, "Sorry, that email address is already used!"
		}
		writeJSON(w, status, map[string]any{
			"code":    code,
			"message": msg,
			"data":    map[string]any{"status": status},
		})
		return
	}
	u, err := c.GetUserBy("id", id)
	if err != nil || u == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_create",
			"message": "created but reload failed",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := buildListedUser(c, u, true, "edit")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Location", fmt.Sprintf("%s/wp-json/wp/v2/users/%d", home, id))
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) serveUserItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	switch r.Method {
	case http.MethodGet:
		s.getUser(w, r, c, id)
	case http.MethodPut, http.MethodPost, http.MethodPatch:
		s.updateUser(w, r, c, id)
	case http.MethodDelete:
		s.deleteUser(w, r, c, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

// deleteUser handles DELETE /wp/v2/users/{id}?force=true[&reassign=N] (P4.79).
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_user_cannot_delete",
			"message": "Sorry, you are not allowed to delete this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.CurrentUserCan("delete_user", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_user_cannot_delete",
			"message": "Sorry, you are not allowed to delete this user.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	u, err := c.GetUserBy("id", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_user_invalid_id",
			"message": "Invalid user ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	force := r.URL.Query().Get("force") == "true"
	if !force {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"code":    "rest_trash_not_supported",
			"message": "Users do not support trashing. Set 'force=true' to delete.",
			"data":    map[string]any{"status": 501},
		})
		return
	}
	var reassignPtr *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("reassign")); raw != "" {
		rid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || rid <= 0 || rid == id {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_user_invalid_reassign",
				"message": "Invalid user ID for reassignment.",
				"data":    map[string]any{"status": 400},
			})
			return
		}
		target, err := c.GetUserBy("id", rid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_user_query",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		if target == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_user_invalid_reassign",
				"message": "Invalid user ID for reassignment.",
				"data":    map[string]any{"status": 400},
			})
			return
		}
		reassignPtr = &rid
	}
	previous := buildListedUser(c, u, true, "edit")
	if err := c.DeleteUser(id, reassignPtr); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "The user cannot be deleted.",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": previous,
	})
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	u, err := c.GetUserBy("id", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_user_invalid_id",
			"message": "Invalid user ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	canList := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "list_users")
	context := strings.TrimSpace(r.URL.Query().Get("context"))
	if context == "" {
		context = "view"
	}
	if context == "edit" && !canList {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_forbidden_context",
			"message": "Sorry, you are not allowed to edit users.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if canList {
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	} else {
		w.Header().Set("Allow", "GET")
	}
	writeJSON(w, http.StatusOK, buildListedUser(c, u, canList, context))
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.CurrentUserCan("edit_user", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this user.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	u, err := c.GetUserBy("id", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_user_invalid_id",
			"message": "Invalid user ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
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
	if roles, ok := fields["roles"].([]any); ok && len(roles) > 0 {
		if rs, ok := roles[0].(string); ok && strings.TrimSpace(rs) != "" {
			role := strings.TrimSpace(rs)
			upd.Role = &role
		}
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
	u, err = c.GetUserBy("id", id)
	if err != nil || u == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_user_update",
			"message": "updated but reload failed",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	writeJSON(w, http.StatusOK, buildListedUser(c, u, true, "edit"))
}

func decodeUserWriteBody(r *http.Request) (userWriteBody, error) {
	var body userWriteBody
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return body, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return body, err
	}
	return body, nil
}

func decodeUserWriteFields(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}
