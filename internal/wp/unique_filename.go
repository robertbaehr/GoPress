// wp_unique_filename (P2.54) — subset of wp-includes/functions.php.
package wp

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var reImageSubsizeName = regexp.MustCompile(`(?i)-(?:\d+x\d+|scaled|rotated)$`)

// UniqueFilename returns a unique filename within dir (wp_unique_filename).
// Sanitizes the name, lowercases the extension when needed, forces a numeric
// suffix for names that look like image sub-sizes, and increments until free.
func UniqueFilename(dir, filename string) string {
	filename = SanitizeFileName(filename)
	if filename == "" {
		filename = "file"
	}

	ext := filepath.Ext(filename) // includes leading "."
	fname := strings.TrimSuffix(filename, ext)
	// Edge case: file named ".ext" → empty name.
	if fname == "" && ext != "" {
		fname = ""
	}

	number := ""
	if fname != "" && reImageSubsizeName.MatchString(fname) {
		number = "1"
		filename = fname + "-" + number + ext
	}

	lcExt := strings.ToLower(ext)
	var lcFilename string
	if ext != "" && lcExt != ext {
		lcFilename = strings.TrimSuffix(filename, ext) + lcExt
	}

	dir = strings.TrimRight(dir, `/\`)
	for fileExists(dir, filename) || (lcFilename != "" && fileExists(dir, lcFilename)) {
		newNumber := 1
		if number != "" {
			n, _ := strconv.Atoi(number)
			newNumber = n + 1
		}
		newNumStr := strconv.Itoa(newNumber)

		if lcFilename != "" {
			lcFilename = replaceNumberSuffix(lcFilename, number, newNumStr, lcExt)
		}
		if number+ext == "" {
			filename = filename + "-" + newNumStr
		} else {
			filename = replaceNumberSuffix(filename, number, newNumStr, ext)
		}
		number = newNumStr
	}

	if lcFilename != "" {
		filename = lcFilename
	}
	return filename
}

// replaceNumberSuffix mirrors WP's str_replace of -{$number}{$ext} / {$number}{$ext}.
func replaceNumberSuffix(name, number, newNumber, ext string) string {
	oldDash := "-" + number + ext
	oldBare := number + ext
	newDash := "-" + newNumber + ext
	if number == "" {
		// First increment: replace ".ext" with "-1.ext" (WP behavior).
		if i := strings.LastIndex(name, oldBare); i >= 0 {
			return name[:i] + newDash + name[i+len(oldBare):]
		}
		return name + "-" + newNumber + ext
	}
	if i := strings.LastIndex(name, oldDash); i >= 0 {
		return name[:i] + newDash + name[i+len(oldDash):]
	}
	if i := strings.LastIndex(name, oldBare); i >= 0 {
		return name[:i] + newDash + name[i+len(oldBare):]
	}
	return strings.TrimSuffix(name, ext) + "-" + newNumber + ext
}

func fileExists(dir, name string) bool {
	if name == "" {
		return false
	}
	path := name
	if dir != "" {
		path = filepath.Join(dir, name)
	}
	_, err := os.Stat(path)
	return err == nil
}
