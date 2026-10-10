package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveApplicationPasswords handles /wp/v2/users/{id|me}/application-passwords (P4.43).
func (s *Server) serveApplicationPasswords(w http.ResponseWriter, r *http.Request, c *wp.Context, userKey, uuid string) {
	user, status, code, msg := resolveAppPasswordUser(c, userKey)
	if user == nil {
		writeJSON(w, status, map[string]any{
			"code":    code,
			"message": msg,
			"data":    map[string]any{"status": status},
		})
		return
	}
	if uuid == "introspect" {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_no_authenticated_app_password",
			"message": "Cannot introspect application password.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if uuid != "" {
		s.serveApplicationPasswordItem(w, r, c, user, uuid)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := wp.GetUserApplicationPasswords(c, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_application_password_error",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, item := range list {
			out = append(out, prepareApplicationPassword(c, user, item, ""))
		}
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		body, err := decodeAppPasswordBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_json",
				"message": err.Error(),
				"data":    map[string]any{"status": 400},
			})
			return
		}
		name := strings.TrimSpace(body["name"])
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_missing_callback_param",
				"message": "Missing parameter(s): name",
				"data":    map[string]any{"status": 400, "params": []string{"name"}},
			})
			return
		}
		plain, item, err := wp.CreateApplicationPassword(c, user.ID, name, strings.TrimSpace(body["app_id"]))
		if err != nil {
			status := http.StatusBadRequest
			code := "application_password_empty_name"
			msg := err.Error()
			switch err.Error() {
			case "application_password_duplicate_name":
				code = "application_password_duplicate_name"
				msg = "An application password with that name already exists."
			case "application_password_empty_name":
				msg = "An application name is required to create an application password."
			default:
				status = http.StatusInternalServerError
				code = "rest_application_password_error"
			}
			writeJSON(w, status, map[string]any{
				"code":    code,
				"message": msg,
				"data":    map[string]any{"status": status},
			})
			return
		}
		resp := prepareApplicationPassword(c, user, item, plain)
		w.Header().Set("Location", appPasswordSelfLink(c, user.ID, item.UUID))
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusCreated, resp)
	case http.MethodDelete:
		n, err := wp.DeleteAllApplicationPasswords(c, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_application_password_error",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
			"count":   n,
		})
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) serveApplicationPasswordItem(w http.ResponseWriter, r *http.Request, c *wp.Context, user *wp.User, uuid string) {
	switch r.Method {
	case http.MethodGet:
		item, err := wp.GetUserApplicationPassword(c, user.ID, uuid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_application_password_error",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		if item == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_application_password_not_found",
				"message": "Application password not found.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
		writeJSON(w, http.StatusOK, prepareApplicationPassword(c, user, *item, ""))
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		body, err := decodeAppPasswordBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_json",
				"message": err.Error(),
				"data":    map[string]any{"status": 400},
			})
			return
		}
		name := strings.TrimSpace(body["name"])
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_missing_callback_param",
				"message": "Missing parameter(s): name",
				"data":    map[string]any{"status": 400, "params": []string{"name"}},
			})
			return
		}
		item, err := wp.UpdateApplicationPassword(c, user.ID, uuid, name)
		if err != nil {
			status := http.StatusNotFound
			code := "rest_application_password_not_found"
			if err.Error() != "application_password_not_found" {
				status = http.StatusBadRequest
				code = err.Error()
			}
			writeJSON(w, status, map[string]any{
				"code":    code,
				"message": err.Error(),
				"data":    map[string]any{"status": status},
			})
			return
		}
		writeJSON(w, http.StatusOK, prepareApplicationPassword(c, user, *item, ""))
	case http.MethodDelete:
		prev, err := wp.DeleteApplicationPassword(c, user.ID, uuid)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_application_password_not_found",
				"message": "Application password not found.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":  true,
			"previous": prepareApplicationPassword(c, user, *prev, ""),
		})
	default:
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func resolveAppPasswordUser(c *wp.Context, userKey string) (user *wp.User, status int, code, msg string) {
	if c == nil || c.CurrentUser == nil {
		return nil, http.StatusUnauthorized, "rest_not_logged_in", "You are not currently logged in."
	}
	userKey = strings.TrimSpace(userKey)
	if userKey == "" || userKey == "me" {
		return c.CurrentUser, 0, "", ""
	}
	id, err := strconv.ParseInt(userKey, 10, 64)
	if err != nil || id <= 0 {
		return nil, http.StatusNotFound, "rest_user_invalid_id", "Invalid user ID."
	}
	// P4.43 focuses on current-user management; other users require edit_user.
	if id != c.CurrentUser.ID {
		if !c.UserCan(c.CurrentUser, "edit_users") && !c.UserCan(c.CurrentUser, "edit_user", id) {
			return nil, http.StatusForbidden, "rest_cannot_manage_application_passwords",
				"Sorry, you are not allowed to manage application passwords for this user."
		}
		u, err := c.GetUserBy("id", id)
		if err != nil || u == nil {
			return nil, http.StatusNotFound, "rest_user_invalid_id", "Invalid user ID."
		}
		return u, 0, "", ""
	}
	return c.CurrentUser, 0, "", ""
}

func prepareApplicationPassword(c *wp.Context, user *wp.User, item wp.ApplicationPassword, newPassword string) map[string]any {
	out := map[string]any{
		"uuid":      item.UUID,
		"app_id":    item.AppID,
		"name":      item.Name,
		"created":   time.Unix(item.Created, 0).UTC().Format("2006-01-02T15:04:05"),
		"last_used": nil,
		"last_ip":   nil,
		"_links": map[string]any{
			"self": []map[string]any{{
				"href": appPasswordSelfLink(c, user.ID, item.UUID),
			}},
		},
	}
	if item.LastUsed != nil {
		out["last_used"] = time.Unix(*item.LastUsed, 0).UTC().Format("2006-01-02T15:04:05")
	}
	if item.LastIP != nil {
		out["last_ip"] = *item.LastIP
	}
	if newPassword != "" {
		out["password"] = newPassword
	}
	return out
}

func appPasswordSelfLink(c *wp.Context, userID int64, uuid string) string {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	return fmt.Sprintf("%s/wp-json/wp/v2/users/%d/application-passwords/%s", home, userID, uuid)
}

func decodeAppPasswordBody(r *http.Request) (map[string]string, error) {
	out := map[string]string{}
	if r == nil || r.Body == nil {
		return out, nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for k, v := range raw {
		if v == nil {
			continue
		}
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

// parseApplicationPasswordsPath extracts user key and optional uuid from
// /wp/v2/users/{id|me}/application-passwords[/{uuid}].
func parseApplicationPasswordsPath(path string) (userKey, uuid string, ok bool) {
	const prefix = "/wp/v2/users/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[1] != "application-passwords" {
		return "", "", false
	}
	userKey = parts[0]
	if userKey == "" {
		return "", "", false
	}
	if len(parts) >= 3 {
		uuid = parts[2]
	}
	return userKey, uuid, true
}
