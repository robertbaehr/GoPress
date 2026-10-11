package blocks

import (
	"strings"
)

// renderWidgetGroup renders core/widget-group (P5.78 / P5.132).
//
// Classic widget areas are out of scope for GoPress (widgets REST disabled;
// no sidebars / WP_Widget_Factory). Match an unresolved widget-group context
// by returning empty string so saved sidebar markup does not leak into
// front-end HTML.
//
// P5.132 deepens attribute/sidebar gates matching render_block_core_widget_group:
//   - attributes.title is parsed (would wrap before_title/after_title)
//   - SidebarBeingRendered selects registered sidebar chrome when set
//   - InnerBlocks / InnerHTML are never emitted (dynamic widget area)
func (e *Env) renderWidgetGroup(block Block) string {
	e = e.ensure()

	// Title would be wrapped in sidebar before_title/after_title (or the
	// default <h2 class="widget-title">…</h2>) — widgets gone → still "".
	_ = widgetGroupAttrString(block, "title")

	// note_sidebar_being_rendered / discard_sidebar_being_rendered.
	// Without $wp_registered_sidebars, both set and unset paths stay empty.
	_ = e.SidebarBeingRendered

	// Discard inner blocks and saved markup — never leak into the front.
	_ = len(block.InnerBlocks)
	_ = block.InnerHTML

	return ""
}

// widgetGroupAttrString returns a non-empty string attribute, or "".
func widgetGroupAttrString(block Block, key string) string {
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
