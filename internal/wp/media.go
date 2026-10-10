// Media helpers (P2.10) — image sizes, attachment URLs, resize stub.
//
// Actual image resizing is deferred (no libvips/imaging dependency); sizes are
// registered from options so callers can discover dimensions.
package wp

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
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

// RemoveImageSize unregisters an additional image size.
func RemoveImageSize(name string) {
	additionalSizesMu.Lock()
	delete(additionalSizes, name)
	additionalSizesMu.Unlock()
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

// GetRegisteredImageSizes returns default sizes from options plus additional sizes.
// Defaults: thumbnail, medium, medium_large, large (get_intermediate_image_sizes).
func GetRegisteredImageSizes(c *Context) map[string]ImageSize {
	out := map[string]ImageSize{
		"thumbnail": {
			Name:   "thumbnail",
			Width:  optionInt(c, "thumbnail_size_w", 150),
			Height: optionInt(c, "thumbnail_size_h", 150),
			Crop:   optionBool(c, "thumbnail_crop", true),
		},
		"medium": {
			Name:   "medium",
			Width:  optionInt(c, "medium_size_w", 300),
			Height: optionInt(c, "medium_size_h", 300),
		},
		"medium_large": {
			Name:   "medium_large",
			Width:  optionInt(c, "medium_large_size_w", 768),
			Height: optionInt(c, "medium_large_size_h", 0),
		},
		"large": {
			Name:   "large",
			Width:  optionInt(c, "large_size_w", 1024),
			Height: optionInt(c, "large_size_h", 1024),
		},
	}
	for k, v := range GetAdditionalImageSizes() {
		out[k] = v
	}
	return out
}

// GetImageSize returns one registered size by name, or false if unknown.
func GetImageSize(c *Context, name string) (ImageSize, bool) {
	sizes := GetRegisteredImageSizes(c)
	s, ok := sizes[name]
	return s, ok
}

// UploadDir describes the uploads base path and URL (wp_upload_dir subset).
type UploadDir struct {
	Basedir string
	Baseurl string
	Error   string
}

// GetUploadDir returns the uploads basedir/baseurl for the site.
func GetUploadDir(c *Context) UploadDir {
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	abspath := ""
	if c != nil && c.Config != nil {
		abspath = strings.TrimRight(c.Config.ABSPATH, "/")
	}
	basedir := abspath + "/wp-content/uploads"
	if custom := strings.TrimSpace(optionStringValue(c, "upload_path", "")); custom != "" {
		if strings.HasPrefix(custom, "/") {
			basedir = custom
		} else if abspath != "" {
			basedir = abspath + "/" + custom
		} else {
			basedir = custom
		}
	}
	baseurl := home + "/wp-content/uploads"
	if custom := strings.TrimSpace(optionStringValue(c, "upload_url_path", "")); custom != "" {
		baseurl = strings.TrimRight(custom, "/")
	}
	return UploadDir{Basedir: basedir, Baseurl: baseurl}
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

// ResizeImage is a stub for intermediate-size generation. GoPress registers sizes
// but does not resize files yet (no imaging/libvips dependency).
func ResizeImage(srcPath string, size ImageSize) (string, error) {
	_, _ = srcPath, size
	return "", fmt.Errorf("wp: ResizeImage: not implemented (sizes registered only)")
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
