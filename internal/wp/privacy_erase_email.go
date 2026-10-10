package wp

import (
	"fmt"
	"strings"
)

// PersonalDataErasureEmailData is the filter bag for erasure fulfillment mail (P8.16).
type PersonalDataErasureEmailData struct {
	Email            string
	PrivacyPolicyURL string
	SiteName         string
	SiteURL          string
}

// SendPersonalDataErasureEmail sends the erasure-fulfilled notification (P8.16).
// Mirrors _wp_privacy_send_erasure_fulfillment_notification without a user_request post.
func SendPersonalDataErasureEmail(c *Context, email string) error {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("wp: invalid erasure email recipient")
	}

	siteName := optionStringPrivacy(c, "blogname", "GoPress")
	siteURL := strings.TrimRight(optionStringPrivacy(c, "home", optionStringPrivacy(c, "siteurl", "")), "/")
	policyURL := ""
	if c != nil {
		policyURL = GetPrivacyPolicyURL(c)
	}

	data := PersonalDataErasureEmailData{
		Email:            email,
		PrivacyPolicyURL: policyURL,
		SiteName:         siteName,
		SiteURL:          siteURL,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("user_erasure_fulfillment_email_to", email, data); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				email = strings.TrimSpace(s)
				data.Email = email
			}
		}
	}

	subject := fmt.Sprintf("[%s] Erasure Request Fulfilled", siteName)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("user_erasure_fulfillment_email_subject", subject, siteName, data); v != nil {
			if s, ok := v.(string); ok && s != "" {
				subject = s
			}
		}
	}

	var emailText string
	if policyURL != "" {
		emailText = `Howdy,

Your request to erase your personal data on ###SITENAME### has been completed.

If you have any follow-up questions or concerns, please contact the site administrator.

For more information, you can also read our privacy policy: ###PRIVACY_POLICY_URL###

Regards,
All at ###SITENAME###
###SITEURL###`
	} else {
		emailText = `Howdy,

Your request to erase your personal data on ###SITENAME### has been completed.

If you have any follow-up questions or concerns, please contact the site administrator.

Regards,
All at ###SITENAME###
###SITEURL###`
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("user_erasure_fulfillment_email_content", emailText, data); v != nil {
			if s, ok := v.(string); ok && s != "" {
				emailText = s
			}
		}
	}
	content := emailText
	content = strings.ReplaceAll(content, "###SITENAME###", siteName)
	content = strings.ReplaceAll(content, "###PRIVACY_POLICY_URL###", policyURL)
	content = strings.ReplaceAll(content, "###SITEURL###", siteURL)
	content = strings.ReplaceAll(content, "###EMAIL###", email)

	ok, err := Mail(c, []string{email}, subject, content, nil)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("wp: erasure email was not sent")
	}
	return nil
}
