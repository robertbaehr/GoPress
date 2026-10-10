package blocks

// renderWidgetGroup renders core/widget-group (P5.78).
//
// Classic widget areas are out of scope for GoPress (widgets REST disabled;
// no sidebars / WP_Widget_Factory). Match an unresolved widget-group context
// by returning empty string so saved sidebar markup does not leak into
// front-end HTML.
func (e *Env) renderWidgetGroup(block Block) string {
	_ = e.ensure()
	_ = block
	// Widgets are gone — empty/placeholder (no sidebar chrome / inner widgets).
	return ""
}
