// wp_plupload_default_settings / wp_max_upload_size (P8.29) — subset of wp-includes/media.php.
package wp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConvertHrToBytes parses PHP ini size strings like "64M" (wp_convert_hr_to_bytes).
func ConvertHrToBytes(value string) int64 {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(value, "g"):
		mult = 1024 * 1024 * 1024
		value = strings.TrimSuffix(value, "g")
	case strings.HasSuffix(value, "m"):
		mult = 1024 * 1024
		value = strings.TrimSuffix(value, "m")
	case strings.HasSuffix(value, "k"):
		mult = 1024
		value = strings.TrimSuffix(value, "k")
	}
	value = strings.TrimSpace(value)
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int64(n * float64(mult))
}

// MaxUploadSize returns the effective PHP upload limit in bytes (wp_max_upload_size).
// Filterable via upload_size_limit.
func MaxUploadSize(c *Context) int64 {
	uBytes := ConvertHrToBytes(iniGet("upload_max_filesize"))
	pBytes := ConvertHrToBytes(iniGet("post_max_size"))
	size := uBytes
	if pBytes > 0 && (size == 0 || pBytes < size) {
		size = pBytes
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("upload_size_limit", size, uBytes, pBytes); v != nil {
			switch n := v.(type) {
			case int64:
				return n
			case int:
				return int64(n)
			case float64:
				return int64(n)
			}
		}
	}
	return size
}

func iniGet(key string) string {
	// Prefer process env overrides for tests; fall back to common defaults.
	if v := strings.TrimSpace(os.Getenv("PHP_" + strings.ToUpper(key))); v != "" {
		return v
	}
	switch key {
	case "upload_max_filesize":
		return "64M"
	case "post_max_size":
		return "64M"
	default:
		return ""
	}
}

// PluploadSettings mirrors _wpPluploadSettings (defaults + browser + limitExceeded).
type PluploadSettings struct {
	Defaults      map[string]any `json:"defaults"`
	Browser       map[string]any `json:"browser"`
	LimitExceeded bool           `json:"limitExceeded"`
}

// PluploadDefaultSettings builds wp_plupload_default_settings data (P8.29).
// Does not enqueue scripts; returns the settings object used for _wpPluploadSettings.
func PluploadDefaultSettings(c *Context) PluploadSettings {
	maxSize := MaxUploadSize(c)
	exts := pluploadAllowedExtensions(c)
	defaults := map[string]any{
		"file_data_name": "async-upload",
		"url":            "/wp-admin/async-upload.php",
		"filters": map[string]any{
			"max_file_size": fmt.Sprintf("%db", maxSize),
			"mime_types": []map[string]any{
				{"extensions": strings.Join(exts, ",")},
			},
		},
	}
	// GoPress has no image editor for webp/avif/heic yet — surface upload errors like WP.
	defaults["webp_upload_error"] = true
	defaults["avif_upload_error"] = true
	defaults["heic_upload_error"] = true

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("plupload_default_settings", defaults); v != nil {
			if m, ok := v.(map[string]any); ok {
				defaults = m
			}
		}
	}

	params := map[string]any{
		"action": "upload-attachment",
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("plupload_default_params", params); v != nil {
			if m, ok := v.(map[string]any); ok {
				params = m
			}
		}
	}
	if c != nil {
		params["_wpnonce"] = c.CreateNonce("media-form")
	} else {
		params["_wpnonce"] = ""
	}
	defaults["multipart_params"] = params

	return PluploadSettings{
		Defaults: defaults,
		Browser: map[string]any{
			"mobile":    false,
			"supported": true, // _device_can_upload() stub: assume capable
		},
		LimitExceeded: false, // multisite upload space — not applicable
	}
}

func pluploadAllowedExtensions(c *Context) []string {
	mimes := GetAllowedMimeTypes(c)
	seen := map[string]bool{}
	out := make([]string, 0, len(mimes)*2)
	for key := range mimes {
		for _, ext := range strings.Split(key, "|") {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if ext == "" || seen[ext] {
				continue
			}
			seen[ext] = true
			out = append(out, ext)
		}
	}
	return out
}
