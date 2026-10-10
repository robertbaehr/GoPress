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

// menuItemWriteBody is the REST create/update payload for /wp/v2/menu-items (P4.57).
type menuItemWriteBody struct {
	Title  json.RawMessage `json:"title"`
	Status string          `json:"status"`
	URL    string          `json:"url"`
	Parent any             `json:"parent"`
	Menus  any             `json:"menus"`
	Type   string          `json:"type"`
}

type menuItemWriteDecoded struct {
	body menuItemWriteBody
	raw  []byte
}

func (s *Server) createMenuItem(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create menu items as this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create menu items as this user.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	decoded, err := decodeMenuItemWrite(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	body := decoded.body
	status := body.Status
	if status == "" {
		status = "publish"
	}
	title := rawOrRendered(body.Title)
	parent := anyToInt64(body.Parent)
	menuID := anyToInt64(body.Menus)
	itemType := strings.TrimSpace(body.Type)
	if itemType == "" {
		itemType = "custom"
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	slug := wp.SanitizeTitle(title)
	if slug == "" {
		slug = "menu-item"
	}
	id, err := insertPostRow(c, postRowInsert{
		Author:  c.CurrentUser.ID,
		Date:    now,
		Content: "",
		Title:   title,
		Excerpt: "",
		Status:  status,
		Slug:    slug,
		Type:    "nav_menu_item",
		Parent:  0,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if err := saveMenuItemMeta(c, id, body.URL, parent, itemType); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if menuID > 0 {
		if err := assignNavMenuItem(c, id, menuID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": err.Error(),
				"data":    map[string]any{"status": 400},
			})
			return
		}
	}
	row, err := loadPostRow(c, "nav_menu_item", id)
	if err != nil || row == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, classicMenuItemFromWP(c, *row))
}

func (s *Server) updateMenuItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu items.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu items.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	existing, err := loadPostRow(c, "nav_menu_item", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	decoded, err := decodeMenuItemWrite(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	body := decoded.body
	title := existing.Title
	if jsonRawHasKey(decoded.raw, "title") {
		title = rawOrRendered(body.Title)
	}
	status := existing.Status
	if body.Status != "" {
		status = body.Status
	}
	url := ""
	parent := int64(0)
	itemType := "custom"
	if c.Meta != nil {
		if v, ok := c.Meta.GetMetadata(wp.MetaPost, id, "_menu_item_url", true).(string); ok {
			url = v
		}
		if v, ok := c.Meta.GetMetadata(wp.MetaPost, id, "_menu_item_menu_item_parent", true).(string); ok && v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				parent = n
			}
		}
		if v, ok := c.Meta.GetMetadata(wp.MetaPost, id, "_menu_item_type", true).(string); ok && v != "" {
			itemType = v
		}
	}
	if jsonRawHasKey(decoded.raw, "url") {
		url = body.URL
	}
	if jsonRawHasKey(decoded.raw, "parent") {
		parent = anyToInt64(body.Parent)
	}
	if body.Type != "" {
		itemType = body.Type
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	slug := existing.Name
	if title != existing.Title {
		slug = wp.SanitizeTitle(title)
		if slug == "" {
			slug = existing.Name
		}
	}
	if err := updateNavMenuItemRow(c, id, title, slug, status, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_edit",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if err := saveMenuItemMeta(c, id, url, parent, itemType); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_edit",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if jsonRawHasKey(decoded.raw, "menus") {
		menuID := anyToInt64(body.Menus)
		if menuID > 0 {
			if err := assignNavMenuItem(c, id, menuID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"code":    "rest_invalid_param",
					"message": err.Error(),
					"data":    map[string]any{"status": 400},
				})
				return
			}
		} else {
			_ = clearNavMenuItemAssignment(c, id)
		}
	}
	row, err := loadPostRow(c, "nav_menu_item", id)
	if err != nil || row == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, classicMenuItemFromWP(c, *row))
}

func (s *Server) deleteMenuItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete menu items.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete menu items.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	existing, err := loadPostRow(c, "nav_menu_item", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
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
			"message": "Menu items do not support trashing. Set 'force=true' to delete.",
			"data":    map[string]any{"status": 501},
		})
		return
	}
	prev := classicMenuItemFromWP(c, *existing)
	_ = clearNavMenuItemAssignment(c, id)
	if err := hardDeletePost(c, id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": prev,
	})
}

func decodeMenuItemWrite(r *http.Request) (menuItemWriteDecoded, error) {
	var out menuItemWriteDecoded
	if r == nil || r.Body == nil {
		return out, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return out, err
	}
	out.raw = data
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out.body); err != nil {
		return out, fmt.Errorf("invalid JSON body")
	}
	return out, nil
}

func jsonRawHasKey(raw []byte, key string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	case int:
		return int64(t)
	case int64:
		return t
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(t)), 10, 64)
		return n
	}
}

func saveMenuItemMeta(c *wp.Context, id int64, url string, parent int64, itemType string) error {
	if c == nil || c.Meta == nil || id <= 0 {
		return fmt.Errorf("nil meta")
	}
	keys := []struct {
		key string
		val string
	}{
		{"_menu_item_type", itemType},
		{"_menu_item_menu_item_parent", strconv.FormatInt(parent, 10)},
		{"_menu_item_object_id", "0"},
		{"_menu_item_object", ""},
		{"_menu_item_target", ""},
		{"_menu_item_classes", "a:0:{}"},
		{"_menu_item_xfn", ""},
		{"_menu_item_url", url},
	}
	for _, kv := range keys {
		if _, _, _, err := c.Meta.UpdateMetadata(wp.MetaPost, id, kv.key, kv.val, ""); err != nil {
			return err
		}
	}
	return nil
}

func assignNavMenuItem(c *wp.Context, objectID, menuTermID int64) error {
	if c == nil || c.DB == nil || objectID <= 0 || menuTermID <= 0 {
		return fmt.Errorf("invalid menu assignment")
	}
	q, args, err := c.DB.Prepare(
		"SELECT term_taxonomy_id FROM "+c.DB.TermTaxonomy()+" WHERE term_id = %d AND taxonomy = %s LIMIT 1",
		menuTermID, "nav_menu",
	)
	if err != nil {
		return err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return err
	}
	if row == nil {
		return fmt.Errorf("invalid menu id")
	}
	ttID := asInt64(row["term_taxonomy_id"])
	if ttID <= 0 {
		return fmt.Errorf("invalid menu id")
	}
	_ = clearNavMenuItemAssignment(c, objectID)
	iq, iargs, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.TermRelationships()+" (object_id, term_taxonomy_id, term_order) VALUES (%d, %d, %d)",
		objectID, ttID, 0,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(iq, iargs...); err != nil {
		return err
	}
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.TermTaxonomy()+" SET count = count + 1 WHERE term_taxonomy_id = %d",
		ttID,
	)
	if err == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}
	return nil
}

func clearNavMenuItemAssignment(c *wp.Context, objectID int64) error {
	if c == nil || c.DB == nil || objectID <= 0 {
		return nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT tr.term_taxonomy_id FROM "+c.DB.TermRelationships()+" tr "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy = %s",
		objectID, "nav_menu",
	)
	if err != nil {
		return err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return err
	}
	dq, dargs, err := c.DB.Prepare(
		"DELETE tr FROM "+c.DB.TermRelationships()+" tr "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy = %s",
		objectID, "nav_menu",
	)
	if err == nil {
		_, _ = c.DB.Exec(dq, dargs...)
	}
	for _, row := range rows {
		ttID := asInt64(row["term_taxonomy_id"])
		if ttID <= 0 {
			continue
		}
		uq, uargs, err := c.DB.Prepare(
			"UPDATE "+c.DB.TermTaxonomy()+" SET count = GREATEST(count - 1, 0) WHERE term_taxonomy_id = %d",
			ttID,
		)
		if err == nil {
			_, _ = c.DB.Exec(uq, uargs...)
		}
	}
	return nil
}

func updateNavMenuItemRow(c *wp.Context, id int64, title, slug, status, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_title = %s, post_name = %s, post_status = %s, "+
			"post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		title, slug, status, modified, modified, id, "nav_menu_item",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func hardDeletePost(c *wp.Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid delete")
	}
	if c.Meta != nil {
		for _, key := range []string{
			"_menu_item_type", "_menu_item_menu_item_parent", "_menu_item_object_id",
			"_menu_item_object", "_menu_item_target", "_menu_item_classes",
			"_menu_item_xfn", "_menu_item_url",
		} {
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, id, key, "")
		}
	}
	q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}
