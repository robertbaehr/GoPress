// WP_Privacy_Policy_Content text-change notice (P8.14).
package wp

import (
	"encoding/json"
	"time"
)

const (
	suggestedPolicyChangedOption = "_wp_suggested_policy_text_has_changed"
	suggestedPolicyMetaKey       = "_wp_suggested_privacy_policy_content"
)

// SuggestedPolicyTextHasChanged reports whether suggested policy text differs
// from the last reviewed cache (option _wp_suggested_policy_text_has_changed).
func SuggestedPolicyTextHasChanged(c *Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	v := c.Options.GetOption(suggestedPolicyChangedOption, "")
	s, _ := v.(string)
	return s == "changed"
}

// SetSuggestedPolicyTextChanged caches the changed/not-changed state.
func SetSuggestedPolicyTextChanged(c *Context, changed bool) {
	if c == nil || c.Options == nil {
		return
	}
	state := "not-changed"
	if changed {
		state = "changed"
	}
	_, _ = c.Options.UpdateOption(suggestedPolicyChangedOption, state, "no")
}

// PrivacyPolicyUpdatedNoticeHTML returns the admin warning when suggested
// privacy policy text has changed (policy_text_changed_notice, P8.14).
func PrivacyPolicyUpdatedNoticeHTML(c *Context) string {
	if !SuggestedPolicyTextHasChanged(c) {
		return ""
	}
	msg := `The suggested privacy policy text has changed. Please <a href="/wp-admin/privacy-policy-guide.php?tab=policyguide">review the guide</a> and update your privacy policy.`
	return `<div class="notice notice-warning is-dismissible policy-text-updated"><p>` + msg + `</p></div>`
}

// TextChangeCheck compares registered suggested policy text against the
// privacy page meta cache and updates `_wp_suggested_policy_text_has_changed`
// (WP_Privacy_Policy_Content::text_change_check, P8.14).
func TextChangeCheck(c *Context) bool {
	pageID := PrivacyPolicyPageID(c)
	if pageID <= 0 {
		return false
	}
	AddSuggestedPrivacyPolicyContent(c)
	newSections := GetSuggestedPolicyText()
	old := loadSuggestedPolicyCache(c, pageID)
	if len(old) == 0 {
		// No prior review — seed cache without flagging a change.
		saveSuggestedPolicyCache(c, pageID, newSections)
		SetSuggestedPolicyTextChanged(c, false)
		return false
	}
	changed := !suggestedPolicyEqual(old, newSections)
	if changed {
		// Mark newly differing sections as Updated for the guide badges.
		markUpdatedSections(newSections, old)
	}
	SetSuggestedPolicyTextChanged(c, changed)
	return changed
}

// PolicyPageUpdated clears updated/removed markers when the privacy policy
// page itself is updated (WP_Privacy_Policy_Content::_policy_page_updated).
func PolicyPageUpdated(c *Context, postID int64) {
	pageID := PrivacyPolicyPageID(c)
	if pageID <= 0 || postID != pageID {
		return
	}
	AddSuggestedPrivacyPolicyContent(c)
	sections := GetSuggestedPolicyText()
	// Persist current suggestions as the reviewed cache; clear change flag.
	cleared := make([]PrivacyPolicySection, 0, len(sections))
	for _, s := range sections {
		if s.Removed > 0 {
			continue
		}
		s.Updated = 0
		cleared = append(cleared, s)
	}
	saveSuggestedPolicyCache(c, pageID, cleared)
	SetSuggestedPolicyTextChanged(c, false)
}

// SeedSuggestedPolicyCache stores the current suggested texts for a policy page.
func SeedSuggestedPolicyCache(c *Context) {
	pageID := PrivacyPolicyPageID(c)
	if pageID <= 0 {
		return
	}
	AddSuggestedPrivacyPolicyContent(c)
	saveSuggestedPolicyCache(c, pageID, GetSuggestedPolicyText())
	SetSuggestedPolicyTextChanged(c, false)
}

type suggestedPolicyCacheEntry struct {
	PluginName string `json:"plugin_name"`
	PolicyText string `json:"policy_text"`
	Removed    int64  `json:"removed,omitempty"`
	Updated    int64  `json:"updated,omitempty"`
}

func loadSuggestedPolicyCache(c *Context, pageID int64) []PrivacyPolicySection {
	if c == nil || c.Meta == nil || pageID <= 0 {
		return nil
	}
	raw := c.Meta.GetMetadata(MetaPost, pageID, suggestedPolicyMetaKey, true)
	s, ok := raw.(string)
	if !ok || s == "" {
		return nil
	}
	var entries []suggestedPolicyCacheEntry
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil
	}
	out := make([]PrivacyPolicySection, 0, len(entries))
	for _, e := range entries {
		out = append(out, PrivacyPolicySection{
			PluginName: e.PluginName,
			PolicyText: e.PolicyText,
			Removed:    e.Removed,
			Updated:    e.Updated,
		})
	}
	return out
}

func saveSuggestedPolicyCache(c *Context, pageID int64, sections []PrivacyPolicySection) {
	if c == nil || c.Meta == nil || pageID <= 0 {
		return
	}
	entries := make([]suggestedPolicyCacheEntry, 0, len(sections))
	for _, s := range sections {
		entries = append(entries, suggestedPolicyCacheEntry{
			PluginName: s.PluginName,
			PolicyText: s.PolicyText,
			Removed:    s.Removed,
			Updated:    s.Updated,
		})
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return
	}
	_, _, _, _ = c.Meta.UpdateMetadata(MetaPost, pageID, suggestedPolicyMetaKey, string(data), "")
}

func suggestedPolicyEqual(a, b []PrivacyPolicySection) bool {
	type pair struct{ n, t string }
	norm := func(in []PrivacyPolicySection) map[pair]struct{} {
		m := make(map[pair]struct{}, len(in))
		for _, s := range in {
			if s.Removed > 0 {
				continue
			}
			m[pair{s.PluginName, s.PolicyText}] = struct{}{}
		}
		return m
	}
	am, bm := norm(a), norm(b)
	if len(am) != len(bm) {
		return false
	}
	for k := range am {
		if _, ok := bm[k]; !ok {
			return false
		}
	}
	return true
}

func markUpdatedSections(newSections, old []PrivacyPolicySection) {
	privacyPolicyMu.Lock()
	defer privacyPolicyMu.Unlock()
	oldText := map[string]string{}
	for _, s := range old {
		if s.Removed == 0 {
			oldText[s.PluginName] = s.PolicyText
		}
	}
	now := time.Now().Unix()
	for i := range privacyPolicyContent {
		prev, ok := oldText[privacyPolicyContent[i].PluginName]
		if !ok || prev != privacyPolicyContent[i].PolicyText {
			privacyPolicyContent[i].Updated = now
		}
	}
	_ = newSections
}
