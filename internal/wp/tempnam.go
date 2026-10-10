// wp_tempnam / get_temp_dir (P2.62) — subset of wp-admin/includes/file.php
// and wp-includes/functions.php.
package wp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reTempnamStripExt = regexp.MustCompile(`\.[^.]*$`)

// GetTempDir returns a writable temporary directory with a trailing separator
// (get_temp_dir). Prefers os.TempDir when writable, else falls back to ".".
func GetTempDir() string {
	dir := os.TempDir()
	if dir != "" && isWritableDir(dir) {
		return trailingslashitPath(dir)
	}
	return trailingslashitPath(".")
}

// TempNam creates an empty temporary file based on filename in dir (wp_tempnam).
// When dir is empty, GetTempDir is used. Returns the full path of the created file.
func TempNam(filename, dir string) string {
	if strings.TrimSpace(dir) == "" {
		dir = GetTempDir()
	}
	dir = trailingslashitPath(dir)

	if filename == "" || filename == "." || filename == "/" || filename == `\` {
		filename = GeneratePassword(13, false) // uniqid()-shaped stand-in
	}

	tempFilename := Basename(filename, "")
	tempFilename = reTempnamStripExt.ReplaceAllString(tempFilename, "")
	if tempFilename == "" {
		// WP recurses with dirname($filename).
		parent := filepath.Dir(filename)
		if parent == filename || parent == "." || parent == "/" {
			parent = GeneratePassword(13, false)
		}
		return TempNam(parent, dir)
	}

	tempFilename += "-" + GeneratePassword(6, false) + ".tmp"
	tempFilename = UniqueFilename(dir, tempFilename)

	// Filesystems typically limit filenames to 255 chars; WP truncates at 252.
	if over := len(tempFilename) - 252; over > 0 {
		if over >= len(filename) {
			filename = GeneratePassword(13, false)
		} else {
			filename = filename[:len(filename)-over]
		}
		return TempNam(filename, dir)
	}

	full := dir + tempFilename
	f, err := os.OpenFile(full, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if isWritableDir(dir) {
			if _, statErr := os.Stat(full); statErr == nil {
				return TempNam(filename, dir)
			}
		}
		return full
	}
	_ = f.Close()
	return full
}

func trailingslashitPath(p string) string {
	if p == "" {
		return string(os.PathSeparator)
	}
	sep := string(os.PathSeparator)
	if strings.HasSuffix(p, "/") || strings.HasSuffix(p, `\`) {
		return p
	}
	return p + sep
}

func isWritableDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false
	}
	// Probe writability with a create/remove.
	f, err := os.CreateTemp(dir, ".gopress-writetest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
