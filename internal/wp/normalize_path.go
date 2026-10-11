// wp_normalize_path / wp_is_stream (P2.41 / P2.80 / P2.116) — subset of wp-includes/functions.php.
package wp

import (
	"strings"
	"sync"
	"unicode"
)

var (
	normalizePathMu    sync.Mutex
	normalizePathCache = map[string]string{}

	// P2.116: mutable stream_get_wrappers()-like registry (case-sensitive).
	streamWrappersMu sync.RWMutex
	streamWrappers   = defaultStreamWrappers()
)

// Common PHP stream wrappers recognized by wp_is_stream (stream_get_wrappers subset).
// Keys match PHP in_array(..., true) — case-sensitive (P2.80).
func defaultStreamWrappers() map[string]bool {
	return map[string]bool{
		"http": true, "https": true, "ftp": true, "ftps": true,
		"file": true, "php": true, "zlib": true, "data": true,
		"glob": true, "phar": true, "zip": true, "ssh2": true,
		"rar": true, "ogg": true, "expect": true,
	}
}

// RegisterStreamWrapper adds a scheme to the wp_is_stream registry (P2.116).
// Empty or duplicate schemes are ignored. Matching remains case-sensitive.
func RegisterStreamWrapper(scheme string) {
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		return
	}
	streamWrappersMu.Lock()
	streamWrappers[scheme] = true
	streamWrappersMu.Unlock()
	// Paths with the new scheme must not stay memoized as non-streams.
	normalizePathMu.Lock()
	normalizePathCache = map[string]string{}
	normalizePathMu.Unlock()
}

// UnregisterStreamWrapper removes a scheme (tests / reset, P2.116).
func UnregisterStreamWrapper(scheme string) {
	streamWrappersMu.Lock()
	delete(streamWrappers, scheme)
	streamWrappersMu.Unlock()
	normalizePathMu.Lock()
	normalizePathCache = map[string]string{}
	normalizePathMu.Unlock()
}

// ResetStreamWrappersForTest restores the default wrapper set (tests only).
func ResetStreamWrappersForTest() {
	streamWrappersMu.Lock()
	streamWrappers = defaultStreamWrappers()
	streamWrappersMu.Unlock()
	ResetNormalizePathCacheForTest()
}

// IsStream reports whether path uses a registered stream wrapper (wp_is_stream).
// Scheme matching is case-sensitive, matching PHP in_array(..., true).
// P2.116: consults the mutable streamWrappers registry (stream_get_wrappers stand-in).
func IsStream(path string) bool {
	i := strings.Index(path, "://")
	if i <= 0 {
		return false
	}
	scheme := path[:i]
	streamWrappersMu.RLock()
	ok := streamWrappers[scheme]
	streamWrappersMu.RUnlock()
	return ok
}

// NormalizePath standardizes directory separators and collapses duplicate slashes
// (wp_normalize_path). Stream wrappers are preserved; Windows drive letters are uppercased.
func NormalizePath(path string) string {
	normalizePathMu.Lock()
	if cached, ok := normalizePathCache[path]; ok {
		normalizePathMu.Unlock()
		return cached
	}
	normalizePathMu.Unlock()

	original := path
	wrapper := ""
	rest := path
	if IsStream(path) {
		parts := strings.SplitN(path, "://", 2)
		wrapper = parts[0] + "://"
		if len(parts) > 1 {
			rest = parts[1]
		} else {
			rest = ""
		}
	}

	rest = strings.ReplaceAll(rest, `\`, `/`)
	rest = collapseSlashes(rest)

	// Windows paths should uppercase the drive letter (c:/ → C:/).
	if len(rest) >= 2 && rest[1] == ':' {
		r := []rune(rest)
		r[0] = unicode.ToUpper(r[0])
		rest = string(r)
	}

	out := wrapper + rest
	normalizePathMu.Lock()
	normalizePathCache[original] = out
	normalizePathMu.Unlock()
	return out
}

// collapseSlashes mirrors PHP preg_replace('|(?<=.)/+|', '/', $path):
// keep the first character, then collapse every subsequent run of '/' to one.
// That preserves a leading '//' UNC prefix while turning '///' into '//'.
func collapseSlashes(path string) string {
	if path == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(path))
	b.WriteByte(path[0])
	for i := 1; i < len(path); {
		if path[i] == '/' {
			b.WriteByte('/')
			for i < len(path) && path[i] == '/' {
				i++
			}
			continue
		}
		b.WriteByte(path[i])
		i++
	}
	return b.String()
}

// ResetNormalizePathCacheForTest clears the normalize-path memo (tests only).
func ResetNormalizePathCacheForTest() {
	normalizePathMu.Lock()
	normalizePathCache = map[string]string{}
	normalizePathMu.Unlock()
}
