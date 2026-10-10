package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// mediaWriteBody is the PATCH/PUT body for /wp/v2/media/{id} (P4.47).
type mediaWriteBody struct {
	Title   json.RawMessage `json:"title"`
	AltText *string         `json:"alt_text"`
	Caption json.RawMessage `json:"caption"`
}

func (s *Server) serveMediaItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	switch r.Method {
	case http.MethodGet:
		s.getMediaItem(w, r, c, id)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateMedia(w, r, c, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) getMediaItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	item, err := getMediaByID(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if item == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, item)
}

// updateMedia handles PATCH/PUT /wp/v2/media/{id} for title and alt_text (P4.47).
func (s *Server) updateMedia(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	existing, err := getMediaByID(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
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
	if !c.UserCan(c.CurrentUser, "edit_post", id) && !c.UserCan(c.CurrentUser, "upload_files") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodeMediaWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	title := existing.Title.Rendered
	if body.Title != nil {
		title = rawOrRendered(body.Title)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := updateMediaTitle(c, id, title, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_update",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if body.AltText != nil && c.Meta != nil {
		if _, _, _, err := c.Meta.UpdateMetadata(wp.MetaPost, id, "_wp_attachment_image_alt", *body.AltText, ""); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_media_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	if body.Caption != nil {
		caption := rawOrRendered(body.Caption)
		if err := updateMediaCaption(c, id, caption, now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_media_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	item, err := getMediaByID(c, id)
	if err != nil || item == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, item)
}

func decodeMediaWriteBody(r *http.Request) (mediaWriteBody, error) {
	var body mediaWriteBody
	if r == nil || r.Body == nil {
		return body, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return body, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return body, fmt.Errorf("invalid JSON body")
	}
	return body, nil
}

func updateMediaTitle(c *wp.Context, id int64, title, modified string) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_title = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		title, modified, modified, id, "attachment",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func updateMediaCaption(c *wp.Context, id int64, caption, modified string) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_excerpt = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		caption, modified, modified, id, "attachment",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}
