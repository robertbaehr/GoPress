package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// FontFamily is a wp/v2 font-families object (P4.36 read stub).
type FontFamily struct {
	ID                 int64          `json:"id"`
	ThemeJSONVersion   int            `json:"theme_json_version"`
	FontFaces          []int64        `json:"font_faces"`
	FontFamilySettings map[string]any `json:"font_family_settings"`
}

// FontFace is a wp/v2 font-faces object nested under a font family.
type FontFace struct {
	ID               int64          `json:"id"`
	ThemeJSONVersion int            `json:"theme_json_version"`
	Parent           int64          `json:"parent"`
	FontFaceSettings map[string]any `json:"font_face_settings"`
}

func canReadFonts(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	if c.UserCan(c.CurrentUser, "edit_theme_options") {
		return true
	}
	return c.UserCan(c.CurrentUser, "edit_posts")
}

func writeFontsForbidden(w http.ResponseWriter, c *wp.Context, message string) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_read",
		"message": message,
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) serveFontFamilies(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodPost:
		s.createFontFamily(w, r, c)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFonts(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access font families.")
		return
	}
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countPosts(c, "wp_font_family", nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "wp_font_family",
		PostStatus:   "publish",
		PostsPerPage: perPage,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]FontFamily, 0, len(rows))
	for _, row := range rows {
		out = append(out, fontFamilyFromWP(c, row))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	w.Header().Set("Allow", "GET, POST")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveFontFamilyItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodDelete:
		s.deleteFontFamily(w, r, c, id)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFonts(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access this font family.")
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_font_family", PostStatus: "publish"})
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
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, fontFamilyFromWP(c, *row))
}

func (s *Server) serveFontFaces(w http.ResponseWriter, r *http.Request, c *wp.Context, familyID int64) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodPost:
		s.createFontFace(w, r, c, familyID)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFonts(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access font faces.")
		return
	}
	if _, err := requireFontFamily(c, familyID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid font family ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	ids, err := listFontFaceIDs(c, familyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]FontFace, 0, len(ids))
	for _, id := range ids {
		face, err := loadFontFace(c, familyID, id)
		if err != nil || face == nil {
			continue
		}
		out = append(out, *face)
	}
	total := len(out)
	setCollectionHeaders(w, r, c, total, 1, 1)
	w.Header().Set("Allow", "GET, POST")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveFontFaceItem(w http.ResponseWriter, r *http.Request, c *wp.Context, familyID, id int64) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodDelete:
		s.deleteFontFace(w, r, c, familyID, id)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFonts(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access font faces.")
		return
	}
	if _, err := requireFontFamily(c, familyID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid font family ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	face, err := loadFontFace(c, familyID, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if face == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET, DELETE")
	writeJSON(w, http.StatusOK, face)
}

func fontFamilyFromWP(c *wp.Context, row wp.Post) FontFamily {
	faces, _ := listFontFaceIDs(c, row.ID)
	if faces == nil {
		faces = []int64{}
	}
	return FontFamily{
		ID:                 row.ID,
		ThemeJSONVersion:   3,
		FontFaces:          faces,
		FontFamilySettings: fontFamilySettings(row),
	}
}

func fontFamilySettings(row wp.Post) map[string]any {
	settings := map[string]any{
		"name":       row.Title,
		"slug":       row.Name,
		"fontFamily": "",
		"preview":    "",
	}
	content := strings.TrimSpace(row.Content)
	if content == "" {
		return settings
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return settings
	}
	if v, ok := parsed["fontFamily"].(string); ok {
		settings["fontFamily"] = v
	}
	if v, ok := parsed["preview"].(string); ok {
		settings["preview"] = v
	}
	return settings
}

func requireFontFamily(c *wp.Context, id int64) (*wp.Post, error) {
	if id <= 0 {
		return nil, errFontNotFound
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_font_family", PostStatus: "publish"})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errFontNotFound
	}
	return row, nil
}

var errFontNotFound = errors.New("font not found")

func listFontFaceIDs(c *wp.Context, familyID int64) ([]int64, error) {
	if c == nil || c.DB == nil || familyID <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s AND post_parent = %d ORDER BY ID ASC LIMIT 99",
		"wp_font_face", "publish", familyID,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["ID"])
		if id > 0 {
			out = append(out, id)
		}
	}
	return out, nil
}

func loadFontFace(c *wp.Context, familyID, id int64) (*FontFace, error) {
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_font_face", PostStatus: "publish"})
	if err != nil {
		return nil, err
	}
	if row == nil || row.Parent != familyID {
		return nil, nil
	}
	return &FontFace{
		ID:               row.ID,
		ThemeJSONVersion: 3,
		Parent:           row.Parent,
		FontFaceSettings: fontFaceSettings(*row),
	}, nil
}

func fontFaceSettings(row wp.Post) map[string]any {
	settings := map[string]any{}
	content := strings.TrimSpace(row.Content)
	if content == "" {
		return settings
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return settings
	}
	return parsed
}

// parseFontFamiliesPath matches /wp/v2/font-families[/{id}[/font-faces[/{faceID}]]].
func parseFontFamiliesPath(path string) (familyID, faceID int64, facesCollection bool, ok bool) {
	rest := strings.TrimPrefix(path, "/wp/v2/font-families")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return 0, 0, false, true
	}
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, 0, false, false
	}
	if len(parts) == 1 {
		return id, 0, false, true
	}
	if parts[1] != "font-faces" {
		return 0, 0, false, false
	}
	if len(parts) == 2 {
		return id, 0, true, true
	}
	if len(parts) != 3 {
		return 0, 0, false, false
	}
	fid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || fid <= 0 {
		return 0, 0, false, false
	}
	return id, fid, true, true
}
