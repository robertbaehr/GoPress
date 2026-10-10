package blocks

// renderLegacyWidget renders core/legacy-widget (P5.77).
//
// Classic PHP widgets are out of scope for GoPress (widgets REST disabled;
// no WP_Widget_Factory). Match render_block_core_legacy_widget when the
// widget cannot be resolved: return empty string. A missing idBase with no
// saved markup yields nothing; saved HTML is not reused because widget
// instance output is dynamic and unavailable.
func (e *Env) renderLegacyWidget(block Block) string {
	_ = e.ensure()
	_ = block
	// Widgets are gone — empty/placeholder (no server-side widget HTML).
	return ""
}
