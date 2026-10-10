package wp

import (
	"regexp"
	"strings"
)

// Filetype is the wp_check_filetype return shape (ext + type).
type Filetype struct {
	Ext  string // matched extension, or ""
	Type string // mime type, or ""
}

// defaultMimeTypes mirrors wp_get_mime_types() before the mime_types filter (subset + common keys).
var defaultMimeTypes = map[string]string{
	"jpg|jpeg|jpe":       "image/jpeg",
	"gif":                "image/gif",
	"png":                "image/png",
	"bmp":                "image/bmp",
	"tiff|tif":           "image/tiff",
	"webp":               "image/webp",
	"avif":               "image/avif",
	"ico":                "image/x-icon",
	"heic":               "image/heic",
	"heif":               "image/heif",
	"heics":              "image/heic-sequence",
	"heifs":              "image/heif-sequence",
	"asf|asx":            "video/x-ms-asf",
	"wmv":                "video/x-ms-wmv",
	"wmx":                "video/x-ms-wmx",
	"wm":                 "video/x-ms-wm",
	"avi":                "video/avi",
	"divx":               "video/divx",
	"flv":                "video/x-flv",
	"mov|qt":             "video/quicktime",
	"mpeg|mpg|mpe":       "video/mpeg",
	"mp4|m4v":            "video/mp4",
	"ogv":                "video/ogg",
	"webm":               "video/webm",
	"mkv":                "video/x-matroska",
	"3gp|3gpp":           "video/3gpp",
	"3g2|3gp2":           "video/3gpp2",
	"txt|asc|c|cc|h|srt": "text/plain",
	"csv":                "text/csv",
	"tsv":                "text/tab-separated-values",
	"ics":                "text/calendar",
	"rtx":                "text/richtext",
	"css":                "text/css",
	"htm|html":           "text/html",
	"vtt":                "text/vtt",
	"mp3|m4a|m4b":        "audio/mpeg",
	"aac":                "audio/aac",
	"ra|ram":             "audio/x-realaudio",
	"wav|x-wav":          "audio/wav",
	"ogg|oga":            "audio/ogg",
	"flac":               "audio/flac",
	"mid|midi":           "audio/midi",
	"wma":                "audio/x-ms-wma",
	"wax":                "audio/x-ms-wax",
	"mka":                "audio/x-matroska",
	"rtf":                "application/rtf",
	"js":                 "application/javascript",
	"pdf":                "application/pdf",
	"swf":                "application/x-shockwave-flash",
	"class":              "application/java",
	"tar":                "application/x-tar",
	"zip":                "application/zip",
	"gz|gzip":            "application/x-gzip",
	"rar":                "application/rar",
	"7z":                 "application/x-7z-compressed",
	"exe":                "application/x-msdownload",
	"psd":                "application/octet-stream",
	"xcf":                "application/octet-stream",
	"doc":                "application/msword",
	"pot|pps|ppt":        "application/vnd.ms-powerpoint",
	"wri":                "application/vnd.ms-write",
	"xla|xls|xlt|xlw":    "application/vnd.ms-excel",
	"docx":               "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xlsx":               "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"pptx":               "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"odt":                "application/vnd.oasis.opendocument.text",
	"odp":                "application/vnd.oasis.opendocument.presentation",
	"ods":                "application/vnd.oasis.opendocument.spreadsheet",
	"wp|wpd":             "application/wordperfect",
	"key":                "application/vnd.apple.keynote",
	"numbers":            "application/vnd.apple.numbers",
	"pages":              "application/vnd.apple.pages",
}

// GetMimeTypes returns the mime map (wp_get_mime_types). Applies "mime_types" filter when c != nil.
func GetMimeTypes(c *Context) map[string]string {
	out := copyStringMap(defaultMimeTypes)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("mime_types", out); v != nil {
			if m, ok := v.(map[string]string); ok {
				return m
			}
		}
	}
	return out
}

// GetAllowedMimeTypes returns upload-allowed mimes (get_allowed_mime_types).
// Unsets swf/exe; unsets htm|html/js unless the user can unfiltered_html.
// Applies "upload_mimes" filter when c != nil.
func GetAllowedMimeTypes(c *Context) map[string]string {
	t := GetMimeTypes(c)
	delete(t, "swf")
	delete(t, "exe")
	unfiltered := false
	if c != nil && c.CurrentUser != nil {
		unfiltered = c.UserCan(c.CurrentUser, "unfiltered_html")
	}
	if !unfiltered {
		delete(t, "htm|html")
		delete(t, "js")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("upload_mimes", t, nil); v != nil {
			if m, ok := v.(map[string]string); ok {
				return m
			}
		}
	}
	return t
}

// CheckFiletype maps a filename to extension + mime (wp_check_filetype, P2.57).
// When mimes is nil, uses GetAllowedMimeTypes(c) (or defaults when c is nil).
func CheckFiletype(c *Context, filename string, mimes map[string]string) Filetype {
	if mimes == nil {
		if c != nil {
			mimes = GetAllowedMimeTypes(c)
		} else {
			mimes = GetAllowedMimeTypes(nil)
		}
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return Filetype{}
	}
	for extPreg, mime := range mimes {
		re, err := regexp.Compile(`(?i)\.(` + extPreg + `)$`)
		if err != nil {
			continue
		}
		if m := re.FindStringSubmatch(filename); len(m) >= 2 {
			return Filetype{Ext: strings.ToLower(m[1]), Type: mime}
		}
	}
	return Filetype{}
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
