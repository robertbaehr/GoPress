// wp_footer / wp_print_footer_scripts (P3.118) — subset of general-template.php / script-loader.php.
package wp

// Footer mirrors wp_footer(): fires the wp_footer action before </body>.
func Footer(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_footer")
}

// PrintFooterScripts mirrors wp_print_footer_scripts(): fires wp_print_footer_scripts (P3.118).
func PrintFooterScripts(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_print_footer_scripts")
}

// RegisterDefaultFooterHooks wires wp_print_footer_scripts onto wp_footer at priority 20
// (default-filters.php).
func RegisterDefaultFooterHooks(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("wp_footer", func(args ...any) {
		PrintFooterScripts(c)
	}, 20, 0)
}
