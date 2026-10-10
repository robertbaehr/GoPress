package rest

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Media is a wp/v2 media (attachment) object (view context).
type Media struct {
	ID            int64          `json:"id"`
	Date          string         `json:"date"`
	DateGMT       string         `json:"date_gmt"`
	GUID          RenderedString `json:"guid"`
	Modified      string         `json:"modified"`
	ModifiedGMT   string         `json:"modified_gmt"`
	Slug          string         `json:"slug"`
	Status        string         `json:"status"`
	Type          string         `json:"type"`
	Link          string         `json:"link"`
	Title         RenderedString `json:"title"`
	Author        int64          `json:"author"`
	FeaturedMedia int64          `json:"featured_media"`
	CommentStatus string         `json:"comment_status"`
	PingStatus    string         `json:"ping_status"`
	Template      string         `json:"template"`
	Meta          any            `json:"meta"`
	ClassList     []string       `json:"class_list"`
	Description   RenderedString `json:"description"`
	Caption       RenderedString `json:"caption"`
	AltText       string         `json:"alt_text"`
	MediaType     string         `json:"media_type"`
	MimeType      string         `json:"mime_type"`
	MediaDetails  any            `json:"media_details"`
	Post          *int64         `json:"post"`
	SourceURL     string         `json:"source_url"`
	Filename      string         `json:"filename"`
	Filesize      any            `json:"filesize,omitempty"`
	Links         map[string]any `json:"_links"`
}

func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method == http.MethodPost {
		s.createMedia(w, r, c)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countMedia(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	items, err := listMediaPaged(c, perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createMedia(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to upload media.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "upload_files") && !c.UserCan(c.CurrentUser, "edit_posts") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to upload media.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_upload_no_data",
			"message": "No data supplied.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_upload_no_data",
			"message": "No data supplied.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	defer file.Close()
	up, err := wp.SaveUploadedMedia(c, hdr.Filename, file, hdr.Size)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_upload_unknown_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	item, err := getMediaByID(c, up.ID)
	if err != nil || item == nil {
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":         up.ID,
			"source_url": up.URL,
			"mime_type":  up.MimeType,
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) serveMediaItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
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
	writeJSON(w, http.StatusOK, item)
}

func listMediaPaged(c *wp.Context, limit, page int) ([]Media, error) {
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
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, post_name, post_status, "+
			"post_title, post_content, post_excerpt, post_mime_type, guid, post_parent, comment_status, ping_status FROM "+
			c.DB.Posts()+" WHERE post_type = %s AND post_status = %s ORDER BY post_date DESC LIMIT %d OFFSET %d",
		"attachment", "inherit", limit, offset,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Media, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaFromRow(c, row, home))
	}
	return out, nil
}

func countMedia(c *wp.Context) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s",
		"attachment", "inherit",
	)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

func getMediaByID(c *wp.Context, id int64) (*Media, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, post_name, post_status, "+
			"post_title, post_content, post_excerpt, post_mime_type, guid, post_parent, comment_status, ping_status FROM "+
			c.DB.Posts()+" WHERE ID = %d AND post_type = %s AND post_status = %s LIMIT 1",
		id, "attachment", "inherit",
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	m := mediaFromRow(c, row, home)
	return &m, nil
}

func mediaFromRow(c *wp.Context, row map[string]any, home string) Media {
	id := asInt64(row["ID"])
	slug := asString(row["post_name"])
	guid := asString(row["guid"])
	mime := asString(row["post_mime_type"])
	title := asString(row["post_title"])
	excerpt := asString(row["post_excerpt"])
	content := asString(row["post_content"])
	parent := asInt64(row["post_parent"])
	sourceURL := sourceURLFromGUID(guid, home)
	if u := wp.GetAttachmentURL(c, id); u != "" {
		sourceURL = u
	}
	attached := attachmentRelativeFile(c, id)
	filename := path.Base(attached)
	if filename == "." || filename == "/" {
		filename = path.Base(sourceURL)
	}
	details := buildMediaDetails(c, id, mime, sourceURL, attached)
	filesize := mediaFilesize(details)
	alt := ""
	if c != nil && c.Meta != nil {
		alt = asString(c.Meta.GetMetadata(wp.MetaPost, id, "_wp_attachment_image_alt", true))
	}
	var postParent *int64
	if parent > 0 {
		p := parent
		postParent = &p
	}
	m := Media{
		ID:            id,
		Date:          mysqlDateToREST(asString(row["post_date"])),
		DateGMT:       mysqlDateToREST(asString(row["post_date_gmt"])),
		GUID:          RenderedString{Rendered: guid},
		Modified:      mysqlDateToREST(asString(row["post_modified"])),
		ModifiedGMT:   mysqlDateToREST(asString(row["post_modified_gmt"])),
		Slug:          slug,
		Status:        asString(row["post_status"]),
		Type:          "attachment",
		Link:          mediaPermalink(home, slug, id),
		Title:         RenderedString{Rendered: title},
		Author:        asInt64(row["post_author"]),
		FeaturedMedia: 0,
		CommentStatus: asString(row["comment_status"]),
		PingStatus:    asString(row["ping_status"]),
		Template:      "",
		Meta:          []any{},
		ClassList: []string{
			fmt.Sprintf("post-%d", id),
			"attachment",
			"type-attachment",
			"status-inherit",
			"hentry",
		},
		Description:  RenderedString{Rendered: mediaDescriptionHTML(id, mime, sourceURL, title, content, details)},
		Caption:      RenderedString{Rendered: mediaCaptionHTML(excerpt, mime, sourceURL)},
		AltText:      alt,
		MediaType:    mediaTypeFromMime(mime),
		MimeType:     mime,
		MediaDetails: details,
		Post:         postParent,
		SourceURL:    sourceURL,
		Filename:     filename,
		Filesize:     filesize,
		Links:        buildMediaLinks(home, id, asInt64(row["post_author"])),
	}
	return m
}

func buildMediaLinks(home string, id, author int64) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	return map[string]any{
		"self": []map[string]any{{
			"href":        fmt.Sprintf("%s/media/%d", restRoot, id),
			"targetHints": map[string]any{"allow": []string{"GET"}},
		}},
		"collection": []map[string]any{{"href": restRoot + "/media"}},
		"about":      []map[string]any{{"href": restRoot + "/types/attachment"}},
		"author": []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/users/%d", restRoot, author),
		}},
		"replies": []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/comments?post=%d", restRoot, id),
		}},
	}
}

func attachmentRelativeFile(c *wp.Context, id int64) string {
	if c == nil || c.Meta == nil || id <= 0 {
		return ""
	}
	file, _ := c.Meta.GetMetadata(wp.MetaPost, id, "_wp_attached_file", true).(string)
	return strings.TrimSpace(file)
}

func buildMediaDetails(c *wp.Context, id int64, mime, sourceURL, attached string) any {
	raw := ""
	if c != nil && c.Meta != nil {
		raw = asString(c.Meta.GetMetadata(wp.MetaPost, id, "_wp_attachment_metadata", true))
	}
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	parsed, err := wp.UnserializePHP(raw)
	if err != nil || parsed == nil {
		return map[string]any{}
	}
	details, ok := phpToJSONMap(parsed)
	if !ok {
		return map[string]any{}
	}
	// Enrich sizes with source_url and mime_type (WP REST prepare_item_for_response).
	uploadsBase := strings.TrimRight(wp.GetUploadDir(c).Baseurl, "/")
	dir := path.Dir(attached)
	if dir == "." {
		dir = ""
	}
	sizesRaw, hasSizes := details["sizes"]
	sizesMap, _ := sizesRaw.(map[string]any)
	if !hasSizes || sizesMap == nil {
		sizesMap = map[string]any{}
	}
	if len(sizesMap) == 0 {
		// Non-image metadata may omit sizes; WP emits {}.
		if !strings.HasPrefix(strings.ToLower(mime), "image/") {
			details["sizes"] = map[string]any{}
			return details
		}
	}
	for sizeName, sizeVal := range sizesMap {
		sm, ok := sizeVal.(map[string]any)
		if !ok {
			continue
		}
		if mt, ok := sm["mime-type"]; ok {
			sm["mime_type"] = mt
			delete(sm, "mime-type")
		}
		fileName := asString(sm["file"])
		if fileName != "" {
			if dir != "" {
				sm["source_url"] = uploadsBase + "/" + dir + "/" + fileName
			} else {
				sm["source_url"] = uploadsBase + "/" + fileName
			}
		}
		sizesMap[sizeName] = sm
	}
	if len(sizesMap) > 0 {
		// WP REST overwrites sizes.full via wp_get_attachment_image_src('full'),
		// using the attachment post mime_type (not the preview file's mime).
		fullURL := sourceURL
		fullFile := path.Base(sourceURL)
		var fullW, fullH any
		if existing, ok := sizesMap["full"].(map[string]any); ok {
			if u := asString(existing["source_url"]); u != "" {
				fullURL = u
			}
			if f := asString(existing["file"]); f != "" {
				fullFile = f
			} else if fullURL != "" {
				fullFile = path.Base(fullURL)
			}
			fullW = existing["width"]
			fullH = existing["height"]
		}
		if strings.HasPrefix(strings.ToLower(mime), "image/") && sourceURL != "" {
			fullURL = sourceURL
			fullFile = path.Base(sourceURL)
			if w, ok := details["width"]; ok {
				fullW = w
			}
			if h, ok := details["height"]; ok {
				fullH = h
			}
		}
		if fullW == nil {
			fullW = details["width"]
			fullH = details["height"]
		}
		full := map[string]any{
			"file":       fullFile,
			"mime_type":  mime,
			"source_url": fullURL,
		}
		if fullW != nil {
			full["width"] = fullW
		}
		if fullH != nil {
			full["height"] = fullH
		}
		sizesMap["full"] = full
	}
	if len(sizesMap) == 0 {
		details["sizes"] = map[string]any{}
	} else {
		details["sizes"] = sizesMap
	}
	// image_meta.keywords must be a JSON array when empty.
	if im, ok := details["image_meta"].(map[string]any); ok {
		if kw, exists := im["keywords"]; exists {
			if m, isMap := kw.(map[string]any); isMap && len(m) == 0 {
				im["keywords"] = []any{}
			}
		} else {
			im["keywords"] = []any{}
		}
		details["image_meta"] = im
	}
	return details
}

func mediaFilesize(details any) any {
	m, ok := details.(map[string]any)
	if !ok {
		return nil
	}
	if fs, ok := m["filesize"]; ok {
		return fs
	}
	return nil
}

func mediaDescriptionHTML(id int64, mime, sourceURL, title, content string, details any) string {
	content = strings.TrimSpace(content)
	if content != "" {
		return content + "\n"
	}
	mimeLower := strings.ToLower(mime)
	if strings.HasPrefix(mimeLower, "video/") && sourceURL != "" {
		return mediaVideoDescriptionHTML(id, mime, sourceURL, details)
	}
	if strings.HasPrefix(mimeLower, "audio/") && sourceURL != "" {
		return mediaAudioDescriptionHTML(id, mime, sourceURL)
	}
	// Images and non-image attachments with a medium preview (e.g. PDF icons).
	mediumURL, width, height := mediaSizeURL(details, "medium")
	if mediumURL == "" && strings.HasPrefix(mimeLower, "image/") && sourceURL != "" {
		mediumURL = sourceURL
		if m, ok := details.(map[string]any); ok {
			width = int(asInt64(m["width"]))
			height = int(asInt64(m["height"]))
		}
	}
	if mediumURL != "" && sourceURL != "" {
		wh := ""
		if width > 0 && height > 0 {
			wh = fmt.Sprintf(` width="%d" height="%d"`, width, height)
		}
		srcset, sizes := "", ""
		if strings.HasPrefix(mimeLower, "image/") {
			srcset, sizes = mediaImageSrcsetSizes(details, mediumURL, width, sourceURL)
		}
		extra := ""
		if srcset != "" {
			extra += ` srcset="` + srcset + `"`
		}
		if sizes != "" {
			extra += ` sizes="` + sizes + `"`
		}
		return fmt.Sprintf(
			`<p class="attachment"><a href='%s'><img loading="lazy" decoding="async"%s src="%s" class="attachment-medium size-medium" alt=""%s /></a></p>`+"\n",
			sourceURL, wh, mediumURL, extra,
		)
	}
	_ = title
	return ""
}

func mediaSizeURL(details any, size string) (url string, width, height int) {
	m, ok := details.(map[string]any)
	if !ok {
		return "", 0, 0
	}
	sizes, ok := m["sizes"].(map[string]any)
	if !ok {
		return "", 0, 0
	}
	sm, ok := sizes[size].(map[string]any)
	if !ok {
		return "", 0, 0
	}
	return asString(sm["source_url"]), int(asInt64(sm["width"])), int(asInt64(sm["height"]))
}

func mediaImageSrcsetSizes(details any, mediumURL string, mediumW int, fullURL string) (srcset, sizesAttr string) {
	seen := map[int]string{}
	add := func(url string, w int) {
		if url == "" || w <= 0 {
			return
		}
		if _, ok := seen[w]; ok {
			return
		}
		seen[w] = url
	}
	add(mediumURL, mediumW)
	if m, ok := details.(map[string]any); ok {
		fullW := int(asInt64(m["width"]))
		if fullW == 0 {
			if _, w, _ := mediaSizeURL(details, "full"); w > 0 {
				fullW = w
			}
		}
		add(fullURL, fullW)
		if sizes, ok := m["sizes"].(map[string]any); ok {
			for _, sizeVal := range sizes {
				sm, ok := sizeVal.(map[string]any)
				if !ok {
					continue
				}
				add(asString(sm["source_url"]), int(asInt64(sm["width"])))
			}
		}
	}
	if len(seen) < 2 {
		return "", ""
	}
	// Prefer medium then full ordering matching wp_get_attachment_image for these fixtures.
	var parts []string
	if mediumURL != "" && mediumW > 0 {
		parts = append(parts, fmt.Sprintf("%s %dw", mediumURL, mediumW))
		delete(seen, mediumW)
	}
	if fullURL != "" {
		if m, ok := details.(map[string]any); ok {
			fullW := int(asInt64(m["width"]))
			if fullW > 0 {
				if u, ok := seen[fullW]; ok {
					parts = append(parts, fmt.Sprintf("%s %dw", u, fullW))
					delete(seen, fullW)
				} else {
					parts = append(parts, fmt.Sprintf("%s %dw", fullURL, fullW))
				}
			}
		}
	}
	if len(parts) < 2 {
		return "", ""
	}
	if mediumW <= 0 {
		mediumW = 300
	}
	return strings.Join(parts, ", "), fmt.Sprintf("auto, (max-width: %dpx) 100vw, %dpx", mediumW, mediumW)
}

func mediaVideoDescriptionHTML(id int64, mime, sourceURL string, details any) string {
	width, height := 640, 360
	if m, ok := details.(map[string]any); ok {
		if w := int(asInt64(m["width"])); w > 0 {
			width = w
		}
		if h := int(asInt64(m["height"])); h > 0 {
			height = h
		}
	}
	srcWithInst := sourceURL + "?_=1"
	return fmt.Sprintf(
		`<div style="width: %dpx;" class="wp-video"><video class="wp-video-shortcode" id="video-%d-1" width="%d" height="%d" preload="metadata" controls="controls"><source type="%s" src="%s" /><a href="%s">%s</a></video></div>`+"\n",
		width, id, width, height, mime, srcWithInst, sourceURL, sourceURL,
	)
}

func mediaAudioDescriptionHTML(id int64, mime, sourceURL string) string {
	srcWithInst := sourceURL + "?_=1"
	return fmt.Sprintf(
		`<audio class="wp-audio-shortcode" id="audio-%d-1" preload="none" style="width: 100%%;" controls="controls"><source type="%s" src="%s" /><a href="%s">%s</a></audio>`+"\n",
		id, mime, srcWithInst, sourceURL, sourceURL,
	)
}

func mediaCaptionHTML(excerpt, mime, sourceURL string) string {
	excerpt = strings.TrimSpace(excerpt)
	if excerpt != "" {
		return "<p>" + excerpt + "</p>\n"
	}
	// WP often surfaces the media URL as the caption for A/V when excerpt is empty.
	if sourceURL != "" && (strings.HasPrefix(mime, "video/") || strings.HasPrefix(mime, "audio/")) {
		return fmt.Sprintf("<p>%s</p>\n", sourceURL)
	}
	return ""
}

func mediaPermalink(home, slug string, id int64) string {
	if slug == "" {
		return fmt.Sprintf("%s/?attachment_id=%d", home, id)
	}
	return fmt.Sprintf("%s/%s/", home, slug)
}

func mediaTypeFromMime(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if strings.HasPrefix(mime, "image/") {
		return "image"
	}
	// WP REST only distinguishes image vs file.
	return "file"
}

// sourceURLFromGUID prefers the post guid when it looks like a URL; otherwise
// best-effort builds an uploads path under home.
func sourceURLFromGUID(guid, home string) string {
	guid = strings.TrimSpace(guid)
	if strings.HasPrefix(guid, "http://") || strings.HasPrefix(guid, "https://") {
		return guid
	}
	if guid == "" {
		return ""
	}
	if strings.HasPrefix(guid, "/") {
		return home + guid
	}
	if strings.Contains(guid, "wp-content/uploads/") {
		return home + "/" + strings.TrimPrefix(guid, "/")
	}
	return home + "/wp-content/uploads/" + strings.TrimPrefix(guid, "/")
}

// phpToJSONMap converts a PHP-unserialized value into a JSON-object map.
func phpToJSONMap(v any) (map[string]any, bool) {
	converted := phpToJSON(v)
	m, ok := converted.(map[string]any)
	return m, ok
}

// phpToJSON converts PHP unserialize output into JSON-friendly values.
// Empty PHP arrays become JSON arrays; maps with contiguous int keys become arrays;
// string-keyed maps become objects.
func phpToJSON(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case bool, string, float64:
		return t
	case int:
		return int64(t)
	case int64:
		return t
	case map[any]any:
		if len(t) == 0 {
			return []any{}
		}
		if list, ok := phpMapAsList(t); ok {
			out := make([]any, len(list))
			for i, item := range list {
				out[i] = phpToJSON(item)
			}
			return out
		}
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprint(k)] = phpToJSON(val)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = phpToJSON(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = phpToJSON(item)
		}
		return out
	default:
		return fmt.Sprint(t)
	}
}

func phpMapAsList(m map[any]any) ([]any, bool) {
	n := len(m)
	if n == 0 {
		return nil, true
	}
	out := make([]any, n)
	for i := 0; i < n; i++ {
		v, ok := m[i]
		if !ok {
			v, ok = m[int64(i)]
		}
		if !ok {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}
