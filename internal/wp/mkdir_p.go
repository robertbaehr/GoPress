// wp_mkdir_p (P2.61) — subset of wp-includes/functions.php.
package wp

import (
	"os"
	"path/filepath"
	"strings"
)

// MkdirP recursively creates directory target, inheriting parent mode bits
// when possible (wp_mkdir_p). Returns true when target exists as a directory
// after the call. Rejects path traversal containing "..".
func MkdirP(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	// P2.80: normalize separators before mkdir (wp_normalize_path).
	target = NormalizePath(target)

	// Stream wrappers beyond local paths are unsupported.
	if IsStream(target) {
		return false
	}

	target = strings.TrimRight(target, "/")
	if target == "" {
		target = "/"
	}

	// Do not allow path traversals (checked before Clean, like WP str_contains).
	if strings.Contains(target, "../") || strings.Contains(target, ".."+string(os.PathSeparator)) ||
		strings.HasSuffix(target, "/..") || strings.HasSuffix(target, string(os.PathSeparator)+"..") ||
		target == ".." || strings.HasPrefix(target, "../") {
		return false
	}

	target = filepath.Clean(filepath.FromSlash(target))

	if fi, err := os.Stat(target); err == nil {
		return fi.IsDir()
	}

	// Inherit permission bits from nearest existing parent.
	dirPerms := os.FileMode(0o777)
	parent := filepath.Dir(target)
	for {
		if fi, err := os.Stat(parent); err == nil && fi.IsDir() {
			dirPerms = fi.Mode().Perm()
			break
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}

	if err := os.MkdirAll(target, dirPerms); err != nil {
		return false
	}
	fi, err := os.Stat(target)
	return err == nil && fi.IsDir()
}
