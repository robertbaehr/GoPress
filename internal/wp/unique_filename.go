// wp_unique_filename (P2.54 / P2.115) — subset of wp-includes/functions.php.
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
// P2.115: also avoid collisions with existing *-NxN / *-scaled / *-rotated files
// (_wp_check_existing_file_names).
func UniqueFilename(dir, filename string) string {
	// P2.80: normalize dir separators (wp_normalize_path).
	if dir != "" {
		dir = NormalizePath(dir)
	}
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

	// Change the extension to lowercase if needed.
	if lcFilename != "" {
		filename = lcFilename
	}

	// P2.115: prevent collisions with existing dimension-like sibling names
	// (photo.jpg vs photo-150x150.jpg), mirroring _wp_check_existing_file_names.
	files := listDirNames(dir)
	if len(files) > 0 {
		count := len(files)
		i := 0
		for i <= count && checkExistingFileNames(filename, files) {
			newNumber := 1
			if number != "" {
				n, _ := strconv.Atoi(number)
				newNumber = n + 1
			}
			newNumStr := strconv.Itoa(newNumber)
			filename = replaceNumberSuffix(filename, number, newNumStr, strings.ToLower(filepath.Ext(filename)))
			number = newNumStr
			i++
		}
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

func listDirNames(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		out = append(out, name)
	}
	return out
}

// checkExistingFileNames mirrors _wp_check_existing_file_names (P2.115).
// True when files contains fname-(NxN|scaled|rotated).ext matching filename.
func checkExistingFileNames(filename string, files []string) bool {
	ext := filepath.Ext(filename)
	fname := strings.TrimSuffix(filename, ext)
	if fname == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(fname) + `-(?:\d+x\d+|scaled|rotated)` + regexp.QuoteMeta(ext) + `$`)
	for _, f := range files {
		if re.MatchString(f) {
			return true
		}
	}
	return false
}
