// wp_enqueue_scripts / wp_print_styles / wp_print_head_scripts (P3.119).
// Subset of general-template.php / script-loader.php / default-filters.php.
package wp

// Head mirrors wp_head(): fires the wp_head action (front document <head>).
func Head(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_head")
}

// EnqueueScripts mirrors wp_enqueue_scripts(): fires wp_enqueue_scripts (P3.119).
func EnqueueScripts(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_enqueue_scripts")
}

// PrintStyles mirrors wp_print_styles(): fires wp_print_styles (P3.119).
func PrintStyles(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_print_styles")
}

// PrintHeadScripts mirrors wp_print_head_scripts(): fires wp_print_head_scripts (P3.119).
func PrintHeadScripts(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoAction("wp_print_head_scripts")
}

// RegisterDefaultHeadHooks wires enqueue/print callbacks onto wp_head at the
// default-filters.php priorities (1 / 8 / 9).
func RegisterDefaultHeadHooks(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("wp_head", func(args ...any) {
		EnqueueScripts(c)
	}, 1, 0)
	c.Hooks.AddAction("wp_head", func(args ...any) {
		PrintStyles(c)
	}, 8, 0)
	c.Hooks.AddAction("wp_head", func(args ...any) {
		PrintHeadScripts(c)
	}, 9, 0)
}
