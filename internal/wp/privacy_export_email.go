package wp

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const privacyExportExpirationSecs = 3 * 24 * 60 * 60 // 3 days

// PersonalDataExportEmailData is the filter bag for export completion mail (P8.15).
type PersonalDataExportEmailData struct {
	Email          string
	ExpirationSecs int
	ExpirationDate string
	ExportFileURL  string
	SiteName       string
	SiteURL        string
}

// SendPersonalDataExportEmail sends the completed-export notification (P8.15).
// Mirrors wp_privacy_send_personal_data_export_email without a user_request post.
func SendPersonalDataExportEmail(c *Context, email, exportFileURL string) error {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("wp: invalid export email recipient")
	}
	exportFileURL = strings.TrimSpace(exportFileURL)
	if exportFileURL == "" {
		return fmt.Errorf("wp: missing export file URL")
	}

	siteName := optionStringPrivacy(c, "blogname", "GoPress")
	siteURL := strings.TrimRight(optionStringPrivacy(c, "home", optionStringPrivacy(c, "siteurl", "")), "/")
	expiration := privacyExportExpirationSecs
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_privacy_export_expiration", expiration); v != nil {
			switch t := v.(type) {
			case int:
				if t > 0 {
					expiration = t
				}
			case int64:
				if t > 0 {
					expiration = int(t)
				}
			}
		}
	}
	dateFmt := optionStringPrivacy(c, "date_format", "F j, Y")
	expirationDate := time.Now().UTC().Add(time.Duration(expiration) * time.Second).Format(goDateFromWP(dateFmt))

	data := PersonalDataExportEmailData{
		Email:          email,
		ExpirationSecs: expiration,
		ExpirationDate: expirationDate,
		ExportFileURL:  exportFileURL,
		SiteName:       siteName,
		SiteURL:        siteURL,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_privacy_personal_data_email_to", email, data); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				email = strings.TrimSpace(s)
				data.Email = email
			}
		}
	}

	subject := fmt.Sprintf("[%s] Personal Data Export", siteName)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_privacy_personal_data_email_subject", subject, siteName, data); v != nil {
			if s, ok := v.(string); ok && s != "" {
				subject = s
			}
		}
	}

	emailText := `Howdy,

Your request for an export of personal data has been completed. You may
download your personal data by clicking on the link below. For privacy
and security, we will automatically delete the file on ###EXPIRATION###,
so please download it before then.

###LINK###

Regards,
All at ###SITENAME###
###SITEURL###`
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_privacy_personal_data_email_content", emailText, 0, data); v != nil {
			if s, ok := v.(string); ok && s != "" {
				emailText = s
			}
		}
	}
	content := emailText
	content = strings.ReplaceAll(content, "###EXPIRATION###", expirationDate)
	content = strings.ReplaceAll(content, "###LINK###", exportFileURL)
	content = strings.ReplaceAll(content, "###EMAIL###", email)
	content = strings.ReplaceAll(content, "###SITENAME###", siteName)
	content = strings.ReplaceAll(content, "###SITEURL###", siteURL)

	ok, err := Mail(c, []string{email}, subject, content, nil)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("wp: export email was not sent")
	}
	return nil
}

// SendPersonalDataExportEmailForResult emails the HTML export download link (P8.15).
func SendPersonalDataExportEmailForResult(c *Context, email string, res *PersonalDataExportResult) error {
	if res == nil {
		return fmt.Errorf("wp: nil export result")
	}
	link := res.HTMLURL
	if link == "" {
		link = PersonalDataExportDownloadURL(filepath.Base(res.HTMLPath))
	}
	if link == "" {
		link = res.JSONURL
	}
	return SendPersonalDataExportEmail(c, email, link)
}

// goDateFromWP maps a tiny subset of WP date_format tokens to Go layouts.
func goDateFromWP(fmtWP string) string {
	fmtWP = strings.TrimSpace(fmtWP)
	if fmtWP == "" {
		return "January 2, 2006"
	}
	// Common WP defaults.
	repl := strings.NewReplacer(
		"F", "January",
		"j", "2",
		"Y", "2006",
		"m", "01",
		"d", "02",
		"y", "06",
	)
	return repl.Replace(fmtWP)
}
