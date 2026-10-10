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

// globalStylesWriteBody is the editable subset of WP_REST_Global_Styles_Controller (P4.51).
type globalStylesWriteBody struct {
	Styles   map[string]any  `json:"styles"`
	Settings map[string]any  `json:"settings"`
	Title    json.RawMessage `json:"title"`
}

func canEditGlobalStyles(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "edit_theme_options")
}

func (s *Server) updateGlobalStylesItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if !canEditGlobalStyles(c) {
		writeGlobalStylesForbidden(w, c, "rest_cannot_edit",
			"Sorry, you are not allowed to edit this global style.")
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_global_styles", PostStatus: "publish"})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_global_styles_not_found",
			"message": "No global styles config exists with that ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	body, err := decodeGlobalStylesWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}

	existing := parseGlobalStylesConfig(row.Content)
	config := map[string]any{
		"isGlobalStylesUserThemeJSON": true,
		"version":                     3,
	}
	if body.Styles != nil {
		config["styles"] = body.Styles
	} else if existing["styles"] != nil {
		config["styles"] = existing["styles"]
	} else {
		config["styles"] = map[string]any{}
	}
	if body.Settings != nil {
		config["settings"] = body.Settings
	} else if existing["settings"] != nil {
		config["settings"] = existing["settings"]
	} else {
		config["settings"] = map[string]any{}
	}
	if v, ok := existing["version"]; ok {
		config["version"] = v
	}

	contentBytes, err := json.Marshal(config)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_global_styles_encode",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	title := row.Title
	if body.Title != nil {
		if t := decodeGlobalStylesTitle(body.Title); t != "" {
			title = t
		}
	}
	modified := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := updateGlobalStylesRow(c, id, title, string(contentBytes), modified); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_global_styles_update",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}

	settings, styles := parseGlobalStylesContent(string(contentBytes))
	w.Header().Set("Allow", "GET, POST, PUT, PATCH")
	writeJSON(w, http.StatusOK, GlobalStylesItem{
		ID:       id,
		Title:    map[string]string{"raw": title, "rendered": title},
		Settings: settings,
		Styles:   styles,
	})
}

func decodeGlobalStylesWriteBody(r *http.Request) (globalStylesWriteBody, error) {
	var body globalStylesWriteBody
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

func decodeGlobalStylesTitle(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var asObj map[string]any
	if err := json.Unmarshal(raw, &asObj); err == nil {
		if v, ok := asObj["raw"].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseGlobalStylesConfig(content string) map[string]any {
	out := map[string]any{}
	content = strings.TrimSpace(content)
	if content == "" {
		return out
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func updateGlobalStylesRow(c *wp.Context, id int64, title, content, modified string) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("nil db")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_title = %s, post_content = %s, post_modified = %s, post_modified_gmt = %s "+
			"WHERE ID = %d AND post_type = %s",
		title, content, modified, modified, id, "wp_global_styles",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}
