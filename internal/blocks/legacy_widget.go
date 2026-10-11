package blocks

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

// renderLegacyWidget renders core/legacy-widget (P5.77 / P5.131).
//
// Classic PHP widgets are out of scope for GoPress (widgets REST disabled;
// no WP_Widget_Factory). Match render_block_core_legacy_widget:
//   - attributes.id → would call wp_render_widget (unavailable → "")
//   - missing idBase → ""
//   - instance.encoded + instance.hash must pass wp_hash / hash_equals (else "")
//   - resolved widget object missing → ""
// Saved InnerHTML is never reused (widget output is dynamic).
func (e *Env) renderLegacyWidget(block Block) string {
	e = e.ensure()

	// Multi-widget id path (wp_find_widgets_sidebar / wp_render_widget).
	if id := legacyWidgetAttrString(block, "id"); id != "" {
		return ""
	}

	idBase := legacyWidgetAttrString(block, "idBase")
	if idBase == "" {
		return ""
	}

	// Optional encoded instance: reject tampered hashes (P5.131).
	if ok, present := legacyWidgetInstanceValid(e, block); present && !ok {
		return ""
	}

	// No widget factory — cannot call the_widget().
	_ = idBase
	return ""
}

// legacyWidgetAttrString returns a non-empty string attribute, or "".
func legacyWidgetAttrString(block Block, key string) string {
	if block.Attrs == nil {
		return ""
	}
	raw, ok := block.Attrs[key]
	if !ok || raw == nil {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// legacyWidgetInstanceValid validates instance.encoded against instance.hash
// when both are present (P5.131). present is false when either field is absent
// (PHP uses an empty $instance and continues). ok is false on decode/hash
// mismatch. When HashNonce is unset, a present encoded+hash pair fails closed.
func legacyWidgetInstanceValid(e *Env, block Block) (ok, present bool) {
	inst := legacyWidgetInstanceMap(block)
	if inst == nil {
		return true, false
	}
	encRaw, hasEnc := inst["encoded"]
	hashRaw, hasHash := inst["hash"]
	if !hasEnc || !hasHash || encRaw == nil || hashRaw == nil {
		return true, false
	}
	encoded, okEnc := encRaw.(string)
	hash, okHash := hashRaw.(string)
	if !okEnc || !okHash {
		return false, true
	}
	present = true
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// PHP base64_decode is permissive; still require a hash match on bytes.
		raw = []byte(encoded)
	}
	if e == nil || e.HashNonce == nil {
		return false, true
	}
	want := e.HashNonce(string(raw))
	return hashEqualsStr(want, hash), true
}

func hashEqualsStr(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func legacyWidgetInstanceMap(block Block) map[string]any {
	if block.Attrs == nil {
		return nil
	}
	raw, ok := block.Attrs["instance"]
	if !ok || raw == nil {
		return nil
	}
	switch t := raw.(type) {
	case map[string]any:
		return t
	default:
		return nil
	}
}
