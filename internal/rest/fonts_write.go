package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func canWriteFonts(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "edit_theme_options")
}

func writeFontsCannotCreate(w http.ResponseWriter, c *wp.Context) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_create",
		"message": "Sorry, you are not allowed to create font families as this user.",
		"data":    map[string]any{"status": status},
	})
}

func writeFontsCannotDelete(w http.ResponseWriter, c *wp.Context) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_delete",
		"message": "Sorry, you are not allowed to delete this font family.",
		"data":    map[string]any{"status": status},
	})
}

// createFontFamily handles POST /wp/v2/font-families (P4.75).
func (s *Server) createFontFamily(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if !canWriteFonts(c) {
		writeFontsCannotCreate(w, c)
		return
	}
	settings, err := decodeFontFamilySettings(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	name := strings.TrimSpace(asStringAny(settings["name"]))
	slug := strings.TrimSpace(asStringAny(settings["slug"]))
	fontFamily := strings.TrimSpace(asStringAny(settings["fontFamily"]))
	if name == "" || slug == "" || fontFamily == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "font_family_settings requires name, slug, and fontFamily.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	slug = wp.SanitizeTitle(slug)
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid font family slug.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if exists, err := fontFamilySlugExists(c, slug); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	} else if exists {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_duplicate_font_family",
			"message": `A font family with slug "` + slug + `" already exists.`,
			"data":    map[string]any{"status": 400},
		})
		return
	}

	contentObj := map[string]any{
		"fontFamily": fontFamily,
		"preview":    strings.TrimSpace(asStringAny(settings["preview"])),
	}
	raw, _ := json.Marshal(contentObj)
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	author := int64(0)
	if c.CurrentUser != nil {
		author = c.CurrentUser.ID
	}
	id, err := insertPostRow(c, postRowInsert{
		Author:  author,
		Date:    now,
		Content: string(raw),
		Title:   name,
		Excerpt: "",
		Status:  "publish",
		Slug:    slug,
		Type:    "wp_font_family",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_font_family", PostStatus: "publish"})
	if err != nil || row == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, fontFamilyFromWP(c, *row))
}

// deleteFontFamily handles DELETE /wp/v2/font-families/{id}?force=true (P4.75).
// Font families do not support trashing.
func (s *Server) deleteFontFamily(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if !canWriteFonts(c) {
		writeFontsCannotDelete(w, c)
		return
	}
	row, err := requireFontFamily(c, id)
	if err != nil || row == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	force := r.URL.Query().Get("force") == "true"
	if !force {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"code":    "rest_trash_not_supported",
			"message": `Font faces do not support trashing. Set "force=true" to delete.`,
			"data":    map[string]any{"status": 501},
		})
		return
	}
	previous := fontFamilyFromWP(c, *row)
	// Cascade-delete child font faces (WP deletes children with parent).
	if faces, err := listFontFaceIDs(c, id); err == nil {
		for _, fid := range faces {
			dq, dargs, err := c.DB.Prepare(
				"DELETE FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s",
				fid, "wp_font_face",
			)
			if err == nil {
				_, _ = c.DB.Exec(dq, dargs...)
			}
		}
	}
	dq, dargs, err := c.DB.Prepare(
		"DELETE FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s",
		id, "wp_font_family",
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if _, err := c.DB.Exec(dq, dargs...); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": previous,
	})
}

func decodeFontFamilySettings(r *http.Request) (map[string]any, error) {
	if r == nil || r.Body == nil {
		return nil, errFontSettingsMissing
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errFontSettingsMissing
	}
	var body struct {
		Settings json.RawMessage `json:"font_family_settings"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, errFontSettingsJSON
	}
	if len(body.Settings) == 0 {
		return nil, errFontSettingsMissing
	}
	// WP accepts stringified JSON (multipart) or a raw object.
	var asString string
	if err := json.Unmarshal(body.Settings, &asString); err == nil {
		var settings map[string]any
		if err := json.Unmarshal([]byte(asString), &settings); err != nil {
			return nil, errFontSettingsJSON
		}
		return settings, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(body.Settings, &settings); err != nil {
		return nil, errFontSettingsJSON
	}
	return settings, nil
}

func fontFamilySlugExists(c *wp.Context, slug string) (bool, error) {
	if c == nil || c.DB == nil || slug == "" {
		return false, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_name = %s LIMIT 1",
		"wp_font_family", slug,
	)
	if err != nil {
		return false, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return false, err
	}
	return row != nil && asInt64(row["ID"]) > 0, nil
}

func asStringAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return strings.Trim(string(b), `"`)
	}
}

var (
	errFontSettingsMissing = errString("font_family_settings is required.")
	errFontSettingsJSON    = errString("font_family_settings parameter must be a valid JSON string.")
)

type errString string

func (e errString) Error() string { return string(e) }
