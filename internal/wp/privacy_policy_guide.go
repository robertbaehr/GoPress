package wp

import (
	"fmt"
	"html"
	"strings"
	"sync"
)

// PrivacyPolicySection is one suggested policy text entry (P8.12).
type PrivacyPolicySection struct {
	PluginName string
	PolicyText string
	Removed    int64 // unix timestamp when plugin deactivated; 0 if active
	Updated    int64
}

var (
	privacyPolicyMu      sync.Mutex
	privacyPolicyContent []PrivacyPolicySection
)

// AddPrivacyPolicyContent registers suggested policy text (wp_add_privacy_policy_content, P8.12).
func AddPrivacyPolicyContent(pluginName, policyText string) {
	pluginName = strings.TrimSpace(pluginName)
	policyText = strings.TrimSpace(policyText)
	if pluginName == "" || policyText == "" {
		return
	}
	privacyPolicyMu.Lock()
	defer privacyPolicyMu.Unlock()
	for _, existing := range privacyPolicyContent {
		if existing.PluginName == pluginName && existing.PolicyText == policyText {
			return
		}
	}
	privacyPolicyContent = append(privacyPolicyContent, PrivacyPolicySection{
		PluginName: pluginName,
		PolicyText: policyText,
	})
}

// ResetPrivacyPolicyContent clears registered sections (tests).
func ResetPrivacyPolicyContent() {
	privacyPolicyMu.Lock()
	privacyPolicyContent = nil
	privacyPolicyMu.Unlock()
}

// GetSuggestedPolicyText returns registered privacy policy sections (P8.12).
func GetSuggestedPolicyText() []PrivacyPolicySection {
	privacyPolicyMu.Lock()
	defer privacyPolicyMu.Unlock()
	out := make([]PrivacyPolicySection, len(privacyPolicyContent))
	copy(out, privacyPolicyContent)
	return out
}

// AddSuggestedPrivacyPolicyContent registers the WordPress default suggested text (P8.12).
func AddSuggestedPrivacyPolicyContent(c *Context) {
	AddPrivacyPolicyContent("WordPress", GetDefaultPrivacyPolicyContent(c, false, false))
}

// PrivacyPolicyGuideHTML renders WP_Privacy_Policy_Content::privacy_policy_guide markup (P8.12).
func PrivacyPolicyGuideHTML(c *Context) string {
	AddSuggestedPrivacyPolicyContent(c)
	sections := GetSuggestedPolicyText()
	var b strings.Builder
	for i, section := range sections {
		idx := i + 1
		plugin := html.EscapeString(section.PluginName)
		b.WriteString(`<h4 class="privacy-settings-accordion-heading">`)
		b.WriteString(`<button aria-expanded="false" class="privacy-settings-accordion-trigger" aria-controls="privacy-settings-accordion-block-`)
		b.WriteString(fmt.Sprintf("%d", idx))
		b.WriteString(`" type="button">`)
		b.WriteString(`<span class="title">`)
		b.WriteString(plugin)
		b.WriteString(`</span>`)
		if section.Removed > 0 {
			b.WriteString(`<span class="badge red"> Removed</span>`)
		} else if section.Updated > 0 {
			b.WriteString(`<span class="badge blue"> Updated</span>`)
		}
		b.WriteString(`<span class="icon"></span></button></h4>`)
		b.WriteString("\n")
		b.WriteString(`<div id="privacy-settings-accordion-block-`)
		b.WriteString(fmt.Sprintf("%d", idx))
		b.WriteString(`" class="privacy-settings-accordion-panel privacy-text-box-body" hidden="hidden">` + "\n")
		b.WriteString(`<div class="privacy-text-copy-content">` + "\n")
		b.WriteString(section.PolicyText)
		b.WriteString("\n</div>\n")
		if section.Removed == 0 {
			b.WriteString(`<div class="privacy-settings-accordion-actions">`)
			b.WriteString(`<span class="success" aria-hidden="true">Copied!</span>`)
			b.WriteString(`<button type="button" class="privacy-text-copy button">`)
			b.WriteString(`<span aria-hidden="true">Copy suggested policy text to clipboard</span>`)
			b.WriteString(`<span class="screen-reader-text">Copy suggested policy text from `)
			b.WriteString(plugin)
			b.WriteString(`.</span></button></div>` + "\n")
		}
		b.WriteString("</div>\n")
	}
	return b.String()
}
