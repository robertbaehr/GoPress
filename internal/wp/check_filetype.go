package wp

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Filetype is the wp_check_filetype return shape (ext + type).
type Filetype struct {
	Ext  string // matched extension, or ""
	Type string // mime type, or ""
}

// FiletypeAndExt is the wp_check_filetype_and_ext return shape (P8.24).
type FiletypeAndExt struct {
	Ext            string // matched extension, or ""
	Type           string // mime type, or ""
	ProperFilename string // corrected filename when extension was wrong; else ""
}

// defaultMimeTypes mirrors wp_get_mime_types() before the mime_types filter (P2.58).
var defaultMimeTypes = map[string]string{
	// Image formats.
	"jpg|jpeg|jpe": "image/jpeg",
	"gif":          "image/gif",
	"png":          "image/png",
	"bmp":          "image/bmp",
	"tiff|tif":     "image/tiff",
	"webp":         "image/webp",
	"avif":         "image/avif",
	"ico":          "image/x-icon",
	"heic":         "image/heic",
	"heif":         "image/heif",
	"heics":        "image/heic-sequence",
	"heifs":        "image/heif-sequence",
	// Video formats.
	"asf|asx":      "video/x-ms-asf",
	"wmv":          "video/x-ms-wmv",
	"wmx":          "video/x-ms-wmx",
	"wm":           "video/x-ms-wm",
	"avi":          "video/avi",
	"divx":         "video/divx",
	"flv":          "video/x-flv",
	"mov|qt":       "video/quicktime",
	"mpeg|mpg|mpe": "video/mpeg",
	"mp4|m4v":      "video/mp4",
	"ogv":          "video/ogg",
	"webm":         "video/webm",
	"mkv":          "video/x-matroska",
	"3gp|3gpp":     "video/3gpp",
	"3g2|3gp2":     "video/3gpp2",
	// Text formats.
	"txt|asc|c|cc|h|srt": "text/plain",
	"csv":                "text/csv",
	"tsv":                "text/tab-separated-values",
	"ics":                "text/calendar",
	"rtx":                "text/richtext",
	"css":                "text/css",
	"htm|html":           "text/html",
	"vtt":                "text/vtt",
	"dfxp":               "application/ttaf+xml",
	// Audio formats.
	"mp3|m4a|m4b": "audio/mpeg",
	"aac":         "audio/aac",
	"ra|ram":      "audio/x-realaudio",
	"wav|x-wav":   "audio/wav",
	"ogg|oga":     "audio/ogg",
	"flac":        "audio/flac",
	"mid|midi":    "audio/midi",
	"wma":         "audio/x-ms-wma",
	"wax":         "audio/x-ms-wax",
	"mka":         "audio/x-matroska",
	// Misc application formats.
	"rtf":     "application/rtf",
	"js":      "application/javascript",
	"pdf":     "application/pdf",
	"swf":     "application/x-shockwave-flash",
	"class":   "application/java",
	"tar":     "application/x-tar",
	"zip":     "application/zip",
	"gz|gzip": "application/x-gzip",
	"rar":     "application/rar",
	"7z":      "application/x-7z-compressed",
	"exe":     "application/x-msdownload",
	"psd":     "application/octet-stream",
	"xcf":     "application/octet-stream",
	// MS Office formats.
	"doc":                          "application/msword",
	"pot|pps|ppt":                  "application/vnd.ms-powerpoint",
	"wri":                          "application/vnd.ms-write",
	"xla|xls|xlt|xlw":              "application/vnd.ms-excel",
	"mdb":                          "application/vnd.ms-access",
	"mpp":                          "application/vnd.ms-project",
	"docx":                         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"docm":                         "application/vnd.ms-word.document.macroEnabled.12",
	"dotx":                         "application/vnd.openxmlformats-officedocument.wordprocessingml.template",
	"dotm":                         "application/vnd.ms-word.template.macroEnabled.12",
	"xlsx":                         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xlsm":                         "application/vnd.ms-excel.sheet.macroEnabled.12",
	"xlsb":                         "application/vnd.ms-excel.sheet.binary.macroEnabled.12",
	"xltx":                         "application/vnd.openxmlformats-officedocument.spreadsheetml.template",
	"xltm":                         "application/vnd.ms-excel.template.macroEnabled.12",
	"xlam":                         "application/vnd.ms-excel.addin.macroEnabled.12",
	"pptx":                         "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"pptm":                         "application/vnd.ms-powerpoint.presentation.macroEnabled.12",
	"ppsx":                         "application/vnd.openxmlformats-officedocument.presentationml.slideshow",
	"ppsm":                         "application/vnd.ms-powerpoint.slideshow.macroEnabled.12",
	"potx":                         "application/vnd.openxmlformats-officedocument.presentationml.template",
	"potm":                         "application/vnd.ms-powerpoint.template.macroEnabled.12",
	"ppam":                         "application/vnd.ms-powerpoint.addin.macroEnabled.12",
	"sldx":                         "application/vnd.openxmlformats-officedocument.presentationml.slide",
	"sldm":                         "application/vnd.ms-powerpoint.slide.macroEnabled.12",
	"onetoc|onetoc2|onetmp|onepkg": "application/onenote",
	"oxps":                         "application/oxps",
	"xps":                          "application/vnd.ms-xpsdocument",
	// OpenOffice formats.
	"odt": "application/vnd.oasis.opendocument.text",
	"odp": "application/vnd.oasis.opendocument.presentation",
	"ods": "application/vnd.oasis.opendocument.spreadsheet",
	"odg": "application/vnd.oasis.opendocument.graphics",
	"odc": "application/vnd.oasis.opendocument.chart",
	"odb": "application/vnd.oasis.opendocument.database",
	"odf": "application/vnd.oasis.opendocument.formula",
	// WordPerfect + iWork.
	"wp|wpd":  "application/wordperfect",
	"key":     "application/vnd.apple.keynote",
	"numbers": "application/vnd.apple.numbers",
	"pages":   "application/vnd.apple.pages",
}

// defaultMimeTypeOrder is PHP wp_get_mime_types() insertion order (P2.110).
var defaultMimeTypeOrder = []string{
	"jpg|jpeg|jpe", "gif", "png", "bmp", "tiff|tif", "webp", "avif", "ico",
	"heic", "heif", "heics", "heifs",
	"asf|asx", "wmv", "wmx", "wm", "avi", "divx", "flv", "mov|qt", "mpeg|mpg|mpe",
	"mp4|m4v", "ogv", "webm", "mkv", "3gp|3gpp", "3g2|3gp2",
	"txt|asc|c|cc|h|srt", "csv", "tsv", "ics", "rtx", "css", "htm|html", "vtt", "dfxp",
	"mp3|m4a|m4b", "aac", "ra|ram", "wav|x-wav", "ogg|oga", "flac", "mid|midi", "wma", "wax", "mka",
	"rtf", "js", "pdf", "swf", "class", "tar", "zip", "gz|gzip", "rar", "7z", "exe", "psd", "xcf",
	"doc", "pot|pps|ppt", "wri", "xla|xls|xlt|xlw", "mdb", "mpp",
	"docx", "docm", "dotx", "dotm", "xlsx", "xlsm", "xlsb", "xltx", "xltm", "xlam",
	"pptx", "pptm", "ppsx", "ppsm", "potx", "potm", "ppam", "sldx", "sldm",
	"onetoc|onetoc2|onetmp|onepkg", "oxps", "xps",
	"odt", "odp", "ods", "odg", "odc", "odb", "odf",
	"wp|wpd", "key", "numbers", "pages",
}

// GetMimeTypes returns the mime map (wp_get_mime_types, P2.58).
// Applies "mime_types" filter when c != nil.
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

// GetAllowedMimeTypes returns upload-allowed mimes (get_allowed_mime_types, P2.59).
// Unsets swf/exe; unsets htm|html/js unless the (optional) user can unfiltered_html.
// When userOpt is omitted, uses c.CurrentUser (PHP $user = null → current_user_can).
// Applies "upload_mimes" filter with the user argument when c != nil.
func GetAllowedMimeTypes(c *Context, userOpt ...*User) map[string]string {
	t := GetMimeTypes(c)
	delete(t, "swf")
	delete(t, "exe")

	var user *User
	if len(userOpt) > 0 {
		user = userOpt[0]
	} else if c != nil {
		user = c.CurrentUser
	}

	unfiltered := false
	if c != nil && user != nil {
		unfiltered = c.UserCan(user, "unfiltered_html")
	}
	if !unfiltered {
		delete(t, "htm|html")
		delete(t, "js")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("upload_mimes", t, user); v != nil {
			if m, ok := v.(map[string]string); ok {
				return m
			}
		}
	}
	return t
}

// CheckFiletype maps a filename to extension + mime (wp_check_filetype, P2.57 / P2.110).
// When mimes is nil, uses GetAllowedMimeTypes(c) (or defaults when c is nil).
// P2.110: basename + strip query/fragment before match; stable mime-key iteration.
func CheckFiletype(c *Context, filename string, mimes map[string]string) Filetype {
	if mimes == nil {
		if c != nil {
			mimes = GetAllowedMimeTypes(c)
		} else {
			mimes = GetAllowedMimeTypes(nil)
		}
	}
	filename = normalizeCheckFilename(filename)
	if filename == "" {
		return Filetype{}
	}
	for _, extPreg := range mimeExtKeys(mimes) {
		mime := mimes[extPreg]
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

// normalizeCheckFilename prepares a name for wp_check_filetype (P2.110).
// Strips URL query/fragment and reduces to the basename so path prefixes and
// download URLs still match extension patterns anchored at $.
func normalizeCheckFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	if i := strings.IndexAny(filename, "?#"); i >= 0 {
		filename = filename[:i]
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	return filepath.Base(filename)
}

// mimeExtKeys returns mime map keys in a stable order (P2.110).
// Prefer PHP defaultMimeTypes insertion order when the key set matches; otherwise sort.
func mimeExtKeys(mimes map[string]string) []string {
	if len(mimes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(mimes))
	seen := make(map[string]bool, len(mimes))
	for _, k := range defaultMimeTypeOrder {
		if _, ok := mimes[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var extra []string
	for k := range mimes {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

// CheckFiletypeAndExt validates filename + optional on-disk image mime
// (wp_check_filetype_and_ext subset, P8.24). Applies upload_mimes via
// GetAllowedMimeTypes when mimes is nil, and the wp_check_filetype_and_ext filter.
func CheckFiletypeAndExt(c *Context, file, filename string, mimes map[string]string) FiletypeAndExt {
	if mimes == nil {
		if c != nil {
			mimes = GetAllowedMimeTypes(c)
		} else {
			mimes = GetAllowedMimeTypes(nil)
		}
	}
	ft := CheckFiletype(c, filename, mimes)
	ext, typ := ft.Ext, ft.Type
	proper := ""
	realMime := ""

	if file != "" {
		if st, err := os.Stat(file); err == nil && !st.IsDir() {
			if typ != "" && strings.HasPrefix(typ, "image/") {
				realMime = GetImageMime(file)
				if realMime != "" && (realMime != typ || isHEICExt(ext)) {
					if mappedExt := imageMimeToExt(c, realMime); mappedExt != "" {
						base := strings.TrimSuffix(filename, filepath.Ext(filename))
						newName := base + "." + mappedExt
						if newName != filename {
							proper = newName
						}
						ft2 := CheckFiletype(c, newName, mimes)
						ext, typ = ft2.Ext, ft2.Type
					} else {
						realMime = ""
					}
				}
			}
		}
	}

	// Final allow-list check (values of get_allowed_mime_types).
	if typ != "" {
		allowed := false
		for _, mime := range mimes {
			if mime == typ {
				allowed = true
				break
			}
		}
		if !allowed {
			ext, typ = "", ""
			proper = ""
		}
	}

	out := FiletypeAndExt{Ext: ext, Type: typ, ProperFilename: proper}
	if c != nil && c.Hooks != nil {
		payload := map[string]any{
			"ext":             out.Ext,
			"type":            out.Type,
			"proper_filename": out.ProperFilename,
		}
		if v := c.Hooks.ApplyFilters("wp_check_filetype_and_ext", payload, file, filename, mimes, realMime); v != nil {
			if m, ok := v.(map[string]any); ok {
				out.Ext = anyString(m["ext"])
				out.Type = anyString(m["type"])
				out.ProperFilename = anyString(m["proper_filename"])
			}
		}
	}
	return out
}

func isHEICExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "heif", "heics", "heifs":
		return true
	default:
		return false
	}
}

func imageMimeToExt(c *Context, mime string) string {
	mimesToExts := map[string]string{
		"image/jpeg":          "jpg",
		"image/png":           "png",
		"image/gif":           "gif",
		"image/bmp":           "bmp",
		"image/tiff":          "tif",
		"image/webp":          "webp",
		"image/avif":          "avif",
		"image/heic":          "heic",
		"image/heif":          "heic",
		"image/heic-sequence": "heic",
		"image/heif-sequence": "heic",
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("getimagesize_mimes_to_exts", mimesToExts); v != nil {
			if m, ok := v.(map[string]string); ok {
				mimesToExts = m
			}
		}
	}
	return mimesToExts[mime]
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
