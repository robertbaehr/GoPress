// wp_is_json_request / wp_is_json_media_type (P2.64) — subset of wp-includes/load.php.
package wp

import (
	"net/http"
	"regexp"
	"sync"
)

var (
	reJSONMediaType = regexp.MustCompile(`(?i)(^|\s|,)application/([\w!#\$&-\^\.\+]+\+)?json(\+oembed)?($|\s|;|,)`)
	jsonMediaMu     sync.Mutex
	jsonMediaCache  = map[string]bool{}
)

// IsJSONMediaType reports whether mediaType looks like application/json
// (including +json / application/json+oembed variants). Mirrors
// wp_is_json_media_type.
func IsJSONMediaType(mediaType string) bool {
	if mediaType == "" {
		return false
	}
	jsonMediaMu.Lock()
	defer jsonMediaMu.Unlock()
	if v, ok := jsonMediaCache[mediaType]; ok {
		return v
	}
	v := reJSONMediaType.MatchString(mediaType)
	jsonMediaCache[mediaType] = v
	return v
}

// IsJSONRequest reports whether the request Accept or Content-Type header
// indicates JSON (wp_is_json_request).
func IsJSONRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if IsJSONMediaType(r.Header.Get("Accept")) {
		return true
	}
	if IsJSONMediaType(r.Header.Get("Content-Type")) {
		return true
	}
	return false
}
