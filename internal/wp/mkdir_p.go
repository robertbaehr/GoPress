// wp_mkdir_p (P2.61 / P2.114) — subset of wp-includes/functions.php.
package wp

import (
	"os"
	"path/filepath"
	"strings"
)

// MkdirP recursively creates directory target, inheriting parent mode bits
// when possible (wp_mkdir_p). Returns true when target exists as a directory
// after the call. Rejects path traversal containing "..".
//
// P2.114: file:// stream wrappers are stripped to a local path; other streams
// remain unsupported. Parent mode uses mode&07777 (setgid/sticky included) and
// chmod is applied after MkdirAll so umask cannot leave a weaker mode.
func MkdirP(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	// P2.80: normalize separators before mkdir (wp_normalize_path).
	target = NormalizePath(target)

	// P2.114: strip file:// wrapper (PHP keeps wrapper for stream mkdir; GoPress
	// only creates local directories). Other stream wrappers stay unsupported.
	if IsStream(target) {
		if !strings.HasPrefix(target, "file://") {
			return false
		}
		target = strings.TrimPrefix(target, "file://")
		if target == "" {
			return false
		}
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

	// Inherit permission bits from nearest existing parent (mode & 07777).
	dirPerms := os.FileMode(0o777)
	targetParent := filepath.Dir(target)
	parent := targetParent
	for {
		if fi, err := os.Stat(parent); err == nil && fi.IsDir() {
			dirPerms = fi.Mode() & os.FileMode(0o7777)
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

	// P2.114: re-apply dir_perms after umask (WP chmod loop on folder_parts).
	rel := strings.TrimPrefix(filepath.ToSlash(target), filepath.ToSlash(targetParent)+"/")
	if rel == "" || rel == filepath.ToSlash(target) {
		_ = os.Chmod(target, dirPerms)
	} else {
		parts := strings.Split(rel, "/")
		cur := targetParent
		for i := 1; i <= len(parts); i++ {
			cur = filepath.Join(targetParent, filepath.Join(parts[:i]...))
			_ = os.Chmod(cur, dirPerms)
		}
	}

	fi, err := os.Stat(target)
	return err == nil && fi.IsDir()
}
