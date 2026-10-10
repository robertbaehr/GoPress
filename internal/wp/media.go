// Media helpers (P2.10 / P8.30) — image sizes, attachment URLs, resize via ImageEditor.
package wp

import (
	"fmt"
	"path"
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

// GetAttachmentURL returns the public URL for an attachment (wp_get_attachment_url).
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
				rel := file
				if i := strings.Index(file, "wp-content/uploads/"); i >= 0 {
					rel = file[i+len("wp-content/uploads/"):]
				}
				url = uploads.Baseurl + "/" + strings.TrimPrefix(rel, "/")
			} else {
				url = uploads.Baseurl + "/" + strings.TrimPrefix(file, "/")
			}
		}
	}
	if url == "" {
		url = strings.TrimSpace(post.GUID)
	}
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_get_attachment_url", url, attachmentID)
		if s, ok := filtered.(string); ok {
			url = s
		}
	}
	return url
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
