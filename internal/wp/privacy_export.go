package wp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PersonalDataExport is a stub personal-data export report (P8.4).
type PersonalDataExport struct {
	Email     string                    `json:"email"`
	SiteName  string                    `json:"site_name"`
	SiteURL   string                    `json:"site_url"`
	Generated string                    `json:"generated"`
	Groups    []PersonalDataExportGroup `json:"groups"`
}

// PersonalDataExportGroup is one section of the export.
type PersonalDataExportGroup struct {
	GroupID          string              `json:"group_id"`
	GroupLabel       string              `json:"group_label"`
	GroupDescription string              `json:"group_description"`
	Items            []map[string]string `json:"items"`
}

// PersonalDataExportResult describes files written for an export request.
type PersonalDataExportResult struct {
	JSONPath string
	HTMLPath string
	JSONURL  string
	HTMLURL  string
}

// PrivacyExportsDir returns uploads/wp-personal-data-exports.
func PrivacyExportsDir(c *Context) string {
	uploads := GetUploadDir(c)
	if uploads.Basedir == "" {
		return ""
	}
	return filepath.Join(uploads.Basedir, "wp-personal-data-exports")
}

// GeneratePersonalDataExportFile writes stub JSON+HTML export files for email (P8.4).
// Full request workflow / ZIP packaging lands later; this primes the export folder shape.
func GeneratePersonalDataExportFile(c *Context, email string) (*PersonalDataExportResult, error) {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("wp: invalid export email")
	}
	dir := PrivacyExportsDir(c)
	if dir == "" {
		return nil, fmt.Errorf("wp: uploads directory unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	indexPath := filepath.Join(dir, "index.php")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		_ = os.WriteFile(indexPath, []byte("<?php\n// Silence is golden.\n"), 0o644)
	}

	obscura, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	base := "wp-personal-data-file-" + obscura
	jsonName := base + ".json"
	htmlName := base + ".html"
	jsonPath := filepath.Join(dir, jsonName)
	htmlPath := filepath.Join(dir, htmlName)

	siteName := optionStringPrivacy(c, "blogname", "GoPress")
	siteURL := strings.TrimRight(optionStringPrivacy(c, "home", optionStringPrivacy(c, "siteurl", "")), "/")
	generated := time.Now().UTC().Format("2006-01-02 15:04:05")

	report := PersonalDataExport{
		Email:     email,
		SiteName:  siteName,
		SiteURL:   siteURL,
		Generated: generated,
		Groups: []PersonalDataExportGroup{
			{
				GroupID:          "about",
				GroupLabel:       "About",
				GroupDescription: "Overview of export report.",
				Items: []map[string]string{
					{"name": "Report generated for", "value": email},
					{"name": "For site", "value": siteName},
					{"name": "At URL", "value": siteURL},
					{"name": "On", "value": generated},
				},
			},
		},
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(jsonPath, raw, 0o644); err != nil {
		return nil, err
	}

	var html strings.Builder
	html.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>")
	html.WriteString(htmlEscapePrivacy("Personal Data Export for " + email))
	html.WriteString("</title></head><body>\n<h1>")
	html.WriteString(htmlEscapePrivacy("Personal Data Export for " + email))
	html.WriteString("</h1>\n")
	html.WriteString("<p>GoPress personal data export stub — full exporter groups land later.</p>\n")
	html.WriteString("<h2>About</h2>\n<ul>\n")
	for _, item := range report.Groups[0].Items {
		html.WriteString("<li><strong>")
		html.WriteString(htmlEscapePrivacy(item["name"]))
		html.WriteString(":</strong> ")
		html.WriteString(htmlEscapePrivacy(item["value"]))
		html.WriteString("</li>\n")
	}
	html.WriteString("</ul>\n</body></html>\n")
	if err := os.WriteFile(htmlPath, []byte(html.String()), 0o644); err != nil {
		return nil, err
	}

	uploads := GetUploadDir(c)
	baseURL := strings.TrimRight(uploads.Baseurl, "/") + "/wp-personal-data-exports"
	return &PersonalDataExportResult{
		JSONPath: jsonPath,
		HTMLPath: htmlPath,
		JSONURL:  baseURL + "/" + jsonName,
		HTMLURL:  baseURL + "/" + htmlName,
	}, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func optionStringPrivacy(c *Context, name, def string) string {
	if c == nil || c.Options == nil {
		return def
	}
	v := c.Options.GetOption(name, def)
	s, ok := v.(string)
	if !ok {
		s = fmt.Sprint(v)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" {
		return def
	}
	return s
}

func htmlEscapePrivacy(s string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`"`, `&quot;`,
		`'`, `&#39;`,
	)
	return replacer.Replace(s)
}
