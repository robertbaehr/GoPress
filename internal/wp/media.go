// Media helpers (P2.10 / P8.30) — image sizes, attachment URLs, resize via ImageEditor.
package wp

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ImageSize is a registered intermediate image size (wp_get_additional_image_sizes subset).
type ImageSize struct {
	Name   string
	Width  int
	Height int
	Crop   bool
}

var (
	additionalSizesMu sync.RWMutex
	additionalSizes   = map[string]ImageSize{}
)

// AddImageSize registers an additional image size (add_image_size).
func AddImageSize(name string, width, height int, crop bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	additionalSizesMu.Lock()
	additionalSizes[name] = ImageSize{Name: name, Width: width, Height: height, Crop: crop}
	additionalSizesMu.Unlock()
}

// RemoveImageSize unregisters an additional image size (remove_image_size).
// Returns true when the size was registered (P8.23).
func RemoveImageSize(name string) bool {
	name = strings.TrimSpace(name)
	additionalSizesMu.Lock()
	defer additionalSizesMu.Unlock()
	if _, ok := additionalSizes[name]; !ok {
		return false
	}
	delete(additionalSizes, name)
	return true
}

// HasImageSize reports whether an additional image size is registered
// (has_image_size — additional sizes only, not defaults, P8.23).
func HasImageSize(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	additionalSizesMu.RLock()
	defer additionalSizesMu.RUnlock()
	_, ok := additionalSizes[name]
	return ok
}

// GetAdditionalImageSizes returns a copy of sizes registered via AddImageSize.
func GetAdditionalImageSizes() map[string]ImageSize {
	additionalSizesMu.RLock()
	defer additionalSizesMu.RUnlock()
	out := make(map[string]ImageSize, len(additionalSizes))
	for k, v := range additionalSizes {
		out[k] = v
	}
	return out
}

// GetIntermediateImageSizes returns intermediate size names
// (get_intermediate_image_sizes + intermediate_image_sizes filter, P8.23).
func GetIntermediateImageSizes(c *Context) []string {
	names := []string{"thumbnail", "medium", "medium_large", "large"}
	for name := range GetAdditionalImageSizes() {
		names = append(names, name)
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("intermediate_image_sizes", names); v != nil {
			switch t := v.(type) {
			case []string:
				names = t
			case []any:
				out := make([]string, 0, len(t))
				for _, item := range t {
					s := strings.TrimSpace(fmt.Sprint(item))
					if s != "" && s != "<nil>" {
						out = append(out, s)
					}
				}
				names = out
			}
		}
	}
	return names
}

// GetRegisteredImageSizes returns default sizes from options plus additional sizes.
// Defaults: thumbnail, medium, medium_large, large (get_intermediate_image_sizes).
// P8.23: delegates to GetRegisteredImageSubsizes (filter-aware).
func GetRegisteredImageSizes(c *Context) map[string]ImageSize {
	return GetRegisteredImageSubsizes(c)
}

// GetRegisteredImageSubsizes returns wp_get_registered_image_subsizes (P8.23).
// Skips sizes with both width and height unset.
func GetRegisteredImageSubsizes(c *Context) map[string]ImageSize {
	additional := GetAdditionalImageSizes()
	out := make(map[string]ImageSize)
	for _, name := range GetIntermediateImageSizes(c) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var size ImageSize
		size.Name = name
		if add, ok := additional[name]; ok {
			size = add
		} else {
			size.Width = optionInt(c, name+"_size_w", 0)
			size.Height = optionInt(c, name+"_size_h", 0)
			if name == "thumbnail" {
				size.Crop = optionBool(c, "thumbnail_crop", true)
			} else {
				size.Crop = optionBool(c, name+"_crop", false)
			}
		}
		if size.Width <= 0 && size.Height <= 0 {
			continue
		}
		out[name] = size
	}
	return out
}

// GetImageSize returns one registered size by name, or false if unknown.
func GetImageSize(c *Context, name string) (ImageSize, bool) {
	sizes := GetRegisteredImageSizes(c)
	s, ok := sizes[name]
	return s, ok
}

// ImageSizeNamesChoose returns labels for the media size dropdown
// (image_size_names_choose filter defaults, P8.23).
func ImageSizeNamesChoose(c *Context) map[string]string {
	names := map[string]string{
		"thumbnail": "Thumbnail",
		"medium":    "Medium",
		"large":     "Large",
		"full":      "Full Size",
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("image_size_names_choose", names); v != nil {
			if m, ok := v.(map[string]string); ok {
				names = m
			}
		}
	}
	return names
}

// UploadDir describes the uploads path and URL (wp_upload_dir / wp_get_upload_dir, P2.60).
type UploadDir struct {
	Path    string // basedir + subdir
	URL     string // baseurl + subdir
	Subdir  string // e.g. "/2024/06" when year/month folders on
	Basedir string
	Baseurl string
	Error   string
}

// GetUploadDir returns wp_get_upload_dir() — upload_dir without creating directories.
func GetUploadDir(c *Context) UploadDir {
	return GetUploadDirAt(c, time.Time{})
}

// GetUploadDirAt is GetUploadDir with an explicit time for year/month subdir
// (yyyy-mm-dd… or empty → now). Matches _wp_upload_dir( $time ).
func GetUploadDirAt(c *Context, when time.Time) UploadDir {
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	siteurl := strings.TrimRight(optionStringValue(c, "siteurl", home), "/")
	abspath := ""
	if c != nil && c.Config != nil {
		abspath = strings.TrimRight(c.Config.ABSPATH, "/")
	}
	uploadPath := strings.TrimSpace(optionStringValue(c, "upload_path", ""))
	basedir := abspath + "/wp-content/uploads"
	if uploadPath != "" && uploadPath != "wp-content/uploads" {
		if strings.HasPrefix(uploadPath, "/") || (abspath != "" && strings.HasPrefix(uploadPath, abspath)) {
			basedir = uploadPath
		} else if abspath != "" {
			basedir = abspath + "/" + uploadPath
		} else {
			basedir = uploadPath
		}
	}
	baseurl := home + "/wp-content/uploads"
	if custom := strings.TrimSpace(optionStringValue(c, "upload_url_path", "")); custom != "" {
		baseurl = strings.TrimRight(custom, "/")
	} else if uploadPath != "" && uploadPath != "wp-content/uploads" && basedir != abspath+"/wp-content/uploads" {
		baseurl = strings.TrimRight(siteurl, "/") + "/" + strings.TrimPrefix(uploadPath, "/")
	}

	subdir := ""
	useYearMonth := true
	if c != nil && c.Options != nil {
		useYearMonth = optionTruthy(c.Options.GetOption("uploads_use_yearmonth_folders", "1"))
	}
	if useYearMonth {
		t := when
		if t.IsZero() {
			t = time.Now()
		}
		subdir = "/" + t.Format("2006") + "/" + t.Format("01")
	}
	return UploadDir{
		Path:    basedir + subdir,
		URL:     baseurl + subdir,
		Subdir:  subdir,
		Basedir: basedir,
		Baseurl: baseurl,
	}
}

// GetAttachmentMetadata returns wp_get_attachment_metadata (P8.22).
// ok is false when missing/invalid (PHP false). Applies wp_get_attachment_metadata filter.
func GetAttachmentMetadata(c *Context, attachmentID int64) (map[string]any, bool) {
	return getAttachmentMetadata(c, attachmentID, false)
}

// AttachmentDimensions returns width/height from attachment metadata (P5.114).
func AttachmentDimensions(c *Context, attachmentID int64) (width, height int) {
	meta, ok := GetAttachmentMetadata(c, attachmentID)
	if !ok || meta == nil {
		return 0, 0
	}
	return anyToInt(meta["width"]), anyToInt(meta["height"])
}

// GetAttachmentMetadataUnfiltered is wp_get_attachment_metadata( $id, true ).
func GetAttachmentMetadataUnfiltered(c *Context, attachmentID int64) (map[string]any, bool) {
	return getAttachmentMetadata(c, attachmentID, true)
}

func getAttachmentMetadata(c *Context, attachmentID int64, unfiltered bool) (map[string]any, bool) {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return nil, false
	}
	raw := c.Meta.GetMetadata(MetaPost, attachmentID, "_wp_attachment_metadata", true)
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" || s == "<nil>" {
		return nil, false
	}
	// Already a map (some code paths store decoded meta).
	if m, ok := raw.(map[string]any); ok && len(m) > 0 {
		return normalizeAttachmentMetadata(m, unfiltered, c, attachmentID)
	}
	parsed, err := UnserializePHP(s)
	if err != nil || parsed == nil {
		return nil, false
	}
	m, ok := phpValueToStringMap(parsed)
	if !ok || len(m) == 0 {
		return nil, false
	}
	return normalizeAttachmentMetadata(m, unfiltered, c, attachmentID)
}

func normalizeAttachmentMetadata(data map[string]any, unfiltered bool, c *Context, attachmentID int64) (map[string]any, bool) {
	if data == nil {
		return nil, false
	}
	// Ensure sizes is a map when present but not an array (WP 6.0+ guard).
	if v, has := data["sizes"]; has {
		switch v.(type) {
		case map[string]any, map[any]any:
			// ok
		default:
			data["sizes"] = map[string]any{}
		}
	}
	if unfiltered {
		return data, true
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_metadata", data, attachmentID)
		switch t := filtered.(type) {
		case map[string]any:
			data = t
		case map[any]any:
			if m, ok := phpValueToStringMap(t); ok {
				data = m
			}
		case bool:
			if !t {
				return nil, false
			}
		case nil:
			return nil, false
		}
	}
	if data == nil {
		return nil, false
	}
	if v, has := data["sizes"]; has {
		switch v.(type) {
		case map[string]any, map[any]any:
		default:
			data["sizes"] = map[string]any{}
		}
	}
	return data, true
}

func phpValueToStringMap(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			key := fmt.Sprint(k)
			if nested, ok := val.(map[any]any); ok {
				if nm, ok := phpValueToStringMap(nested); ok {
					out[key] = nm
					continue
				}
			}
			out[key] = val
		}
		return out, true
	default:
		return nil, false
	}
}

// GetAttachedFile returns the filesystem path for an attachment (_wp_attached_file).
func GetAttachedFile(c *Context, attachmentID int64) string {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return ""
	}
	file, _ := c.Meta.GetMetadata(MetaPost, attachmentID, "_wp_attached_file", true).(string)
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	if strings.HasPrefix(file, "/") || (len(file) > 2 && file[1] == ':') {
		return file
	}
	uploads := GetUploadDir(c)
	if uploads.Basedir == "" {
		return file
	}
	return uploads.Basedir + "/" + file
}

// AttachmentPagesEnabled reports whether attachment pages are enabled
// (option wp_attachment_pages_enabled, P3.109). Default is off on modern installs.
func AttachmentPagesEnabled(c *Context) bool {
	return optionBool(c, "wp_attachment_pages_enabled", false)
}

// GetAttachmentLink returns an HTML <a> wrapping the attachment image or title
// (wp_get_attachment_link subset, P3.109). size defaults to thumbnail; permalink=false
// links to the file URL.
func GetAttachmentLink(c *Context, attachmentID int64, size string, permalink bool) string {
	if attachmentID <= 0 {
		return ""
	}
	if size == "" {
		size = "thumbnail"
	}
	url := GetAttachmentURL(c, attachmentID)
	if permalink {
		if home := strings.TrimRight(HomeURL(c, "/", ""), "/"); home != "" {
			url = fmt.Sprintf("%s/?attachment_id=%d", home, attachmentID)
		}
	}
	if url == "" {
		return ""
	}
	inner := GetAttachmentImage(c, attachmentID, size, false, nil)
	if strings.TrimSpace(inner) == "" {
		post, err := GetPost(c, QueryArgs{P: attachmentID, PostType: "attachment", PostStatus: "inherit"})
		title := ""
		if err == nil && post != nil {
			title = post.Title
		}
		if title == "" {
			title = url
		}
		inner = htmlEscapeAttr(title)
	}
	link := `<a href="` + htmlEscapeAttr(url) + `">` + inner + `</a>`
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_link", link, attachmentID, size, permalink, false, false, "")
		if s, ok := filtered.(string); ok {
			link = s
		}
	}
	return link
}

// PrependAttachment builds prepend_attachment HTML and prefixes content (P3.109).
// Images use a medium-sized linked representation; non-images fall back to a title link.
func PrependAttachment(c *Context, attachmentID int64, content string) string {
	if attachmentID <= 0 {
		return content
	}
	p := `<p class="attachment">` + GetAttachmentLink(c, attachmentID, "medium", false) + `</p>`
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("prepend_attachment", p)
		if s, ok := filtered.(string); ok {
			p = s
		}
	}
	if strings.TrimSpace(p) == "" {
		return content
	}
	if strings.TrimSpace(content) == "" {
		return p
	}
	return p + "\n" + content
}

// GetAttachmentURL returns the public URL for an attachment (wp_get_attachment_url, P8.42).
func GetAttachmentURL(c *Context, attachmentID int64) string {
	if c == nil || attachmentID <= 0 {
		return ""
	}
	post, err := GetPost(c, QueryArgs{P: attachmentID, PostType: "attachment", PostStatus: "inherit"})
	if err != nil || post == nil {
		return ""
	}
	url := ""
	if c.Meta != nil {
		file, _ := c.Meta.GetMetadata(MetaPost, attachmentID, "_wp_attached_file", true).(string)
		file = strings.TrimSpace(file)
		if file != "" {
			uploads := GetUploadDir(c)
			if strings.HasPrefix(file, uploads.Basedir) {
				url = uploads.Baseurl + strings.TrimPrefix(file, uploads.Basedir)
			} else if strings.Contains(file, "wp-content/uploads") {
				// P8.42: relative path via basename + dirname like _wp_get_attachment_relative_path.
				rel := attachmentRelativePath(file)
				url = strings.TrimRight(uploads.Baseurl, "/") + "/" + strings.TrimPrefix(rel, "/")
			} else {
				url = uploads.Baseurl + "/" + strings.TrimPrefix(file, "/")
			}
		}
	}
	if url == "" {
		url = strings.TrimSpace(post.GUID)
	}
	// P8.42: on SSL front end (not admin / login), force https like set_url_scheme.
	if url != "" && IsSSL(c) && !scopeIsAdmin(c) && !IsLogin(c) {
		url = SetURLScheme(c, url, "https")
	}
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_url", url, attachmentID)
		if s, ok := filtered.(string); ok {
			url = s
		}
	}
	// WP returns false when empty after filters.
	return strings.TrimSpace(url)
}

// attachmentRelativePath mirrors _wp_get_attachment_relative_path (P8.42).
func attachmentRelativePath(file string) string {
	file = strings.TrimSpace(file)
	if i := strings.Index(file, "wp-content/uploads/"); i >= 0 {
		file = file[i+len("wp-content/uploads/"):]
	}
	dir := ""
	base := file
	if j := strings.LastIndex(file, "/"); j >= 0 {
		dir = file[:j]
		base = file[j+1:]
	}
	if dir == "" || dir == "." {
		return base
	}
	return strings.TrimPrefix(dir, "/") + "/" + base
}

func scopeIsAdmin(c *Context) bool {
	if c == nil || c.Scope == nil {
		return false
	}
	return c.Scope.IsAdmin
}

// GetPostThumbnailID returns the featured image attachment ID (_thumbnail_id).
func GetPostThumbnailID(c *Context, postID int64) int64 {
	if c == nil || c.Meta == nil || postID <= 0 {
		return 0
	}
	v := c.Meta.GetMetadata(MetaPost, postID, "_thumbnail_id", true)
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case []byte:
		n, err := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		n, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(t)), 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
}

// GetPostThumbnailURL returns the featured image source URL, or empty.
func GetPostThumbnailURL(c *Context, postID int64) string {
	id := GetPostThumbnailID(c, postID)
	if id <= 0 {
		return ""
	}
	return GetAttachmentURL(c, id)
}

// AttachmentImageSrc is the result of wp_get_attachment_image_src (P8.32).
type AttachmentImageSrc struct {
	URL     string
	Width   int
	Height  int
	Resized bool // intermediate size (not full)
}

// ImageDownsize returns src/width/height/is_intermediate for an attachment size
// (image_downsize subset, P8.32 / P8.44). ok false when no image URL is available.
// P8.44: thumb→thumbnail alias; ImageGetIntermediateSize; non-image sizes.full preview.
func ImageDownsize(c *Context, attachmentID int64, size string) (AttachmentImageSrc, bool) {
	var out AttachmentImageSrc
	if c == nil || attachmentID <= 0 {
		return out, false
	}
	size = strings.TrimSpace(size)
	if size == "" {
		size = "medium"
	}
	// P8.44: WP accepts 'thumb' as an alias for 'thumbnail'.
	if size == "thumb" {
		size = "thumbnail"
	}
	// image_downsize filter short-circuit.
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("image_downsize", false, attachmentID, size)
		switch t := filtered.(type) {
		case AttachmentImageSrc:
			if t.URL != "" {
				return t, true
			}
		case []any:
			if len(t) >= 3 {
				src := AttachmentImageSrc{URL: strings.TrimSpace(fmt.Sprint(t[0]))}
				src.Width, _ = strconv.Atoi(fmt.Sprint(t[1]))
				src.Height, _ = strconv.Atoi(fmt.Sprint(t[2]))
				if len(t) >= 4 {
					src.Resized = optionTruthy(t[3])
				}
				if src.URL != "" {
					return src, true
				}
			}
		case bool:
			// false → continue
		}
	}
	fullURL := GetAttachmentURL(c, attachmentID)
	if fullURL == "" {
		return out, false
	}
	meta, ok := GetAttachmentMetadata(c, attachmentID)
	width := 0
	height := 0
	if ok {
		width = anyToInt(meta["width"])
		height = anyToInt(meta["height"])
	}
	isIntermediate := false
	imgURL := fullURL
	imgURLBasename := path.Base(fullURL)

	// P8.44: non-image attachments use sizes.full preview (e.g. PDF *-pdf.jpg).
	// ImageGetIntermediateSize intentionally skips "full", so read sizes bag directly.
	// Use mime-only check (not AttachmentIsImage) to avoid GetAttachmentImageSrc recursion.
	if !attachmentMimeIsImage(c, attachmentID) {
		fullSz := attachmentSizeEntry(meta, "full")
		if fullSz == nil {
			return out, false
		}
		file := strings.TrimSpace(fmt.Sprint(fullSz["file"]))
		if file == "" {
			return out, false
		}
		imgURL = strings.Replace(fullURL, imgURLBasename, path.Base(file), 1)
		imgURLBasename = path.Base(file)
		width = anyToInt(fullSz["width"])
		height = anyToInt(fullSz["height"])
	}

	// Try registered intermediate size (image_get_intermediate_size).
	if size != "full" {
		if intermediate := ImageGetIntermediateSize(c, attachmentID, size); intermediate != nil {
			file := strings.TrimSpace(fmt.Sprint(intermediate["file"]))
			if file != "" {
				imgURL = strings.Replace(imgURL, imgURLBasename, path.Base(file), 1)
				width = anyToInt(intermediate["width"])
				height = anyToInt(intermediate["height"])
				isIntermediate = true
			}
		} else if size == "thumbnail" && ok {
			// P8.44: legacy meta['thumb'] fallback when no sizes.thumbnail.
			if thumb := strings.TrimSpace(fmt.Sprint(meta["thumb"])); thumb != "" && thumb != "<nil>" {
				imgURL = strings.Replace(imgURL, imgURLBasename, path.Base(thumb), 1)
				isIntermediate = true
				// Dimensions unknown without file probe; leave width/height if set.
			}
		}
	}

	if width == 0 && height == 0 && ok {
		width = anyToInt(meta["width"])
		height = anyToInt(meta["height"])
	}
	out = AttachmentImageSrc{URL: imgURL, Width: width, Height: height, Resized: isIntermediate}
	return out, out.URL != ""
}

// attachmentMimeIsImage reports image/* mime from the posts table (P8.44).
// Empty/unknown mime is treated as image so existing image attachments keep working.
func attachmentMimeIsImage(c *Context, attachmentID int64) bool {
	if c == nil || c.DB == nil || attachmentID <= 0 {
		return true
	}
	q, args, err := c.DB.Prepare(
		"SELECT post_mime_type FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s LIMIT 1",
		attachmentID, "attachment",
	)
	if err != nil {
		return true
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return true
	}
	mime := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["post_mime_type"])))
	if mime == "" || mime == "<nil>" {
		return true
	}
	return strings.HasPrefix(mime, "image/")
}

// attachmentSizeEntry reads meta["sizes"][name] as map[string]any (P8.44).
func attachmentSizeEntry(meta map[string]any, name string) map[string]any {
	if meta == nil || name == "" {
		return nil
	}
	sizesRaw, ok := meta["sizes"]
	if !ok || sizesRaw == nil {
		return nil
	}
	var sizes map[string]any
	switch t := sizesRaw.(type) {
	case map[string]any:
		sizes = t
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			sizes = m
		}
	}
	if sizes == nil {
		return nil
	}
	entry, ok := sizes[name]
	if !ok || entry == nil {
		return nil
	}
	switch t := entry.(type) {
	case map[string]any:
		return t
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			return m
		}
	}
	return nil
}

// GetAttachmentImageSrc is wp_get_attachment_image_src (P8.32 / P8.48).
// size is a registered size name (thumbnail, medium, …). For [width,height] arrays
// use GetAttachmentImageSrcAny.
func GetAttachmentImageSrc(c *Context, attachmentID int64, size string, icon bool) (AttachmentImageSrc, bool) {
	return GetAttachmentImageSrcAny(c, attachmentID, size, icon)
}

// GetAttachmentImageSrcAny is wp_get_attachment_image_src with string|int[] size (P8.48).
// size may be a registered name or []int{width,height} / []any{width,height}.
// Filter returning false / non-array clears the result (WP returns false).
func GetAttachmentImageSrcAny(c *Context, attachmentID int64, size any, icon bool) (AttachmentImageSrc, bool) {
	sizeName, reqW, reqH := normalizeAttachmentImageSize(size)
	var src AttachmentImageSrc
	var ok bool
	if sizeName != "" {
		src, ok = ImageDownsize(c, attachmentID, sizeName)
	} else if reqW > 0 || reqH > 0 {
		src, ok = imageDownsizeByDimensions(c, attachmentID, reqW, reqH)
	} else {
		src, ok = ImageDownsize(c, attachmentID, "thumbnail")
	}
	if !ok {
		// P8.48: icon=true soft fallback when downsize fails (mime icon stub deferred;
		// keep false unless a filter supplies a source).
		_ = icon
		src = AttachmentImageSrc{}
	}
	if c != nil && c.Hooks != nil {
		var filterSize any = sizeName
		if sizeName == "" && (reqW > 0 || reqH > 0) {
			filterSize = []int{reqW, reqH}
		} else if sizeName == "" {
			filterSize = size
		}
		var arr any
		if ok && src.URL != "" {
			arr = []any{src.URL, src.Width, src.Height, src.Resized}
		} else {
			arr = false
		}
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_image_src", arr, attachmentID, filterSize, icon)
		switch t := filtered.(type) {
		case bool:
			if !t {
				return AttachmentImageSrc{}, false
			}
		case nil:
			return AttachmentImageSrc{}, false
		case []any:
			if len(t) < 1 {
				return AttachmentImageSrc{}, false
			}
			src.URL = strings.TrimSpace(fmt.Sprint(t[0]))
			if len(t) >= 2 {
				src.Width, _ = strconv.Atoi(fmt.Sprint(t[1]))
			}
			if len(t) >= 3 {
				src.Height, _ = strconv.Atoi(fmt.Sprint(t[2]))
			}
			if len(t) >= 4 {
				src.Resized = optionTruthy(t[3])
			}
			ok = src.URL != ""
		case AttachmentImageSrc:
			src = t
			ok = src.URL != ""
		}
	}
	if !ok || src.URL == "" {
		return AttachmentImageSrc{}, false
	}
	return src, true
}

// normalizeAttachmentImageSize parses WP size arg: string name or [w,h] (P8.48).
func normalizeAttachmentImageSize(size any) (name string, w, h int) {
	switch t := size.(type) {
	case string:
		return strings.TrimSpace(t), 0, 0
	case []int:
		if len(t) >= 2 {
			return "", t[0], t[1]
		}
		if len(t) == 1 {
			return "", t[0], 0
		}
	case []int64:
		if len(t) >= 2 {
			return "", int(t[0]), int(t[1])
		}
	case []any:
		if len(t) >= 2 {
			return "", anyToInt(t[0]), anyToInt(t[1])
		}
	case []float64:
		if len(t) >= 2 {
			return "", int(t[0]), int(t[1])
		}
	}
	return "", 0, 0
}

// imageDownsizeByDimensions picks the closest intermediate or full for [w,h] (P8.48).
func imageDownsizeByDimensions(c *Context, attachmentID int64, reqW, reqH int) (AttachmentImageSrc, bool) {
	full, ok := ImageDownsize(c, attachmentID, "full")
	if !ok {
		return full, false
	}
	meta, metaOK := GetAttachmentMetadata(c, attachmentID)
	best := full
	bestScore := -1
	if metaOK {
		sizesRaw, _ := meta["sizes"]
		var sizes map[string]any
		switch t := sizesRaw.(type) {
		case map[string]any:
			sizes = t
		case map[any]any:
			if m, ok := phpValueToStringMap(t); ok {
				sizes = m
			}
		}
		for name := range sizes {
			if name == "full" {
				continue
			}
			cand, cok := ImageDownsize(c, attachmentID, name)
			if !cok || cand.URL == "" {
				continue
			}
			// Prefer sizes that fit within requested box; score by area proximity.
			if reqW > 0 && cand.Width > reqW {
				continue
			}
			if reqH > 0 && cand.Height > reqH {
				continue
			}
			score := cand.Width*cand.Height
			if score > bestScore {
				best = cand
				bestScore = score
			}
		}
	}
	if bestScore < 0 {
		// No fitting intermediate: return full with constrained display dims.
		w, h := full.Width, full.Height
		if reqW > 0 && w > reqW {
			h = h * reqW / w
			w = reqW
		}
		if reqH > 0 && h > reqH {
			w = w * reqH / h
			h = reqH
		}
		full.Width, full.Height = w, h
		return full, true
	}
	return best, true
}

// GetAttachmentImageURL is wp_get_attachment_image_url (P8.32).
func GetAttachmentImageURL(c *Context, attachmentID int64, size string, icon bool) string {
	src, ok := GetAttachmentImageSrc(c, attachmentID, size, icon)
	if !ok {
		return ""
	}
	return src.URL
}

// GetAttachmentImage returns an HTML <img> for an attachment (wp_get_attachment_image, P8.32 / P8.43).
// attr may include class, alt, loading, decoding, fetchpriority, width, height, src.
// P8.43: fires wp_get_attachment_image_attributes; validates decoding; honors fetchpriority.
func GetAttachmentImage(c *Context, attachmentID int64, size string, icon bool, attr map[string]string) string {
	if size == "" {
		size = "thumbnail"
	}
	src, ok := GetAttachmentImageSrc(c, attachmentID, size, icon)
	if !ok {
		return ""
	}
	sizeClass := sanitizeImageSizeClass(size)
	alt := ""
	if c != nil && c.Meta != nil {
		alt = strings.TrimSpace(fmt.Sprint(c.Meta.GetMetadata(MetaPost, attachmentID, "_wp_attachment_image_alt", true)))
		if alt == "<nil>" {
			alt = ""
		}
	}
	attrs := map[string]string{
		"src":      src.URL,
		"class":    "attachment-" + sizeClass + " size-" + sizeClass,
		"alt":      alt,
		"loading":  "lazy",
		"decoding": "async",
	}
	if src.Width > 0 {
		attrs["width"] = strconv.Itoa(src.Width)
	}
	if src.Height > 0 {
		attrs["height"] = strconv.Itoa(src.Height)
	}
	for k, v := range attr {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		attrs[k] = v
	}
	// P8.43: width/height from src when caller omitted a numeric override.
	if w := strings.TrimSpace(attrs["width"]); w == "" || !isNumericString(w) {
		if src.Width > 0 {
			attrs["width"] = strconv.Itoa(src.Width)
		} else {
			delete(attrs, "width")
		}
	}
	if h := strings.TrimSpace(attrs["height"]); h == "" || !isNumericString(h) {
		if src.Height > 0 {
			attrs["height"] = strconv.Itoa(src.Height)
		} else {
			delete(attrs, "height")
		}
	}
	// P8.43: omit decoding unless async|sync|auto (WP media.php).
	switch strings.TrimSpace(attrs["decoding"]) {
	case "async", "sync", "auto":
	default:
		delete(attrs, "decoding")
	}
	// P8.43: omit loading / fetchpriority when false or empty.
	if v := strings.TrimSpace(attrs["loading"]); v == "" || v == "false" {
		delete(attrs, "loading")
	}
	if v := strings.TrimSpace(attrs["fetchpriority"]); v == "" || v == "false" {
		delete(attrs, "fetchpriority")
	} else if v != "high" && v != "low" && v != "auto" {
		delete(attrs, "fetchpriority")
	}

	// P8.45: generate srcset/sizes when not already present (wp_get_attachment_image).
	if strings.TrimSpace(attrs["srcset"]) == "" {
		if meta, okMeta := GetAttachmentMetadata(c, attachmentID); okMeta {
			srcset := CalculateImageSrcset(c, src.Width, src.Height, src.URL, meta, attachmentID)
			sizes := CalculateImageSizes(c, src.Width, src.Height, src.URL, meta, attachmentID)
			if srcset != "" && (sizes != "" || strings.TrimSpace(attrs["sizes"]) != "") {
				attrs["srcset"] = srcset
				if strings.TrimSpace(attrs["sizes"]) == "" {
					attrs["sizes"] = sizes
				}
			}
		}
	}
	// P8.45: prepend auto to sizes for lazy-loaded images (wp_img_tag_add_auto_sizes).
	addAutoSizes := true
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_img_tag_add_auto_sizes", true); v != nil {
			if b, ok := v.(bool); ok {
				addAutoSizes = b
			}
		}
	}
	// P8.46: prependAutoSizes helper; only when loading=lazy (trimmed).
	if addAutoSizes && strings.EqualFold(strings.TrimSpace(attrs["loading"]), "lazy") {
		if sz := strings.TrimSpace(attrs["sizes"]); sz != "" {
			attrs["sizes"] = prependAutoSizes(sz)
		}
	}

	var attachment *Post
	if c != nil {
		attachment, _ = GetPostByID(c, attachmentID)
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_image_attributes", attrs, attachment, size)
		if m, ok := filtered.(map[string]string); ok && m != nil {
			attrs = m
		}
	}

	// Preserve prior GoPress attr order (src/class/width/height/alt/decoding/loading)
	// so front custom-logo / featured-image HTML stays stable for the diff corpus.
	order := []string{"src", "class", "width", "height", "alt", "decoding", "loading", "fetchpriority", "srcset", "sizes"}
	seen := map[string]bool{}
	var b strings.Builder
	b.WriteString(`<img`)
	for _, name := range order {
		v, ok := attrs[name]
		if !ok || strings.TrimSpace(v) == "" && name != "alt" {
			continue
		}
		seen[name] = true
		b.WriteByte(' ')
		b.WriteString(htmlEscapeAttr(name))
		b.WriteString(`="`)
		b.WriteString(htmlEscapeAttr(v))
		b.WriteByte('"')
	}
	var extra []string
	for name := range attrs {
		if seen[name] || name == "" {
			continue
		}
		extra = append(extra, name)
	}
	sort.Strings(extra)
	for _, name := range extra {
		b.WriteByte(' ')
		b.WriteString(htmlEscapeAttr(name))
		b.WriteString(`="`)
		b.WriteString(htmlEscapeAttr(attrs[name]))
		b.WriteByte('"')
	}
	b.WriteString(` />`)
	html := b.String()
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_image", html, attachmentID, size, icon, attr)
		if s, ok := filtered.(string); ok {
			html = s
		}
	}
	return html
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r < '0' || r > '9' {
			if i == 0 && (r == '+' || r == '-') {
				continue
			}
			return false
		}
	}
	return true
}

func sanitizeImageSizeClass(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return "thumbnail"
	}
	var b strings.Builder
	for _, r := range size {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func anyToInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
		return n
	}
}

// AttachmentPermalink returns the front URL for an attachment page.
func AttachmentPermalink(c *Context, attachmentID int64, slug string) string {
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	if slug == "" {
		return fmt.Sprintf("%s/?attachment_id=%d", home, attachmentID)
	}
	return fmt.Sprintf("%s/%s/", home, slug)
}

func optionInt(c *Context, name string, fallback int) int {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return fallback
		}
		return n
	case []byte:
		n, err := strconv.Atoi(strings.TrimSpace(string(t)))
		if err != nil {
			return fallback
		}
		return n
	default:
		n, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(t)))
		if err != nil {
			return fallback
		}
		return n
	}
}

func optionBool(c *Context, name string, fallback bool) bool {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off", "":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func optionStringValue(c *Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		if t == "" {
			return fallback
		}
		return t
	case []byte:
		if len(t) == 0 {
			return fallback
		}
		return string(t)
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}

// IntermediateSizeURL builds a best-effort URL for a named size by inserting
// -{w}x{h} before the file extension (WordPress naming convention). Does not
// verify the file exists on disk.
func IntermediateSizeURL(fullURL string, size ImageSize) string {
	if fullURL == "" || (size.Width <= 0 && size.Height <= 0) {
		return fullURL
	}
	ext := path.Ext(fullURL)
	if ext == "" {
		return fullURL
	}
	base := strings.TrimSuffix(fullURL, ext)
	return fmt.Sprintf("%s-%dx%d%s", base, size.Width, size.Height, ext)
}
