package wp

// CurrentAction returns the innermost tag on c.Hooks' current stack
// (current_action, P9.36). Nil context/hooks → "".
func CurrentAction(c *Context) string {
	if c == nil || c.Hooks == nil {
		return ""
	}
	return c.Hooks.CurrentAction()
}

// CurrentFilter returns the innermost tag on c.Hooks' current stack
// (current_filter, P9.36). Nil context/hooks → "".
func CurrentFilter(c *Context) string {
	if c == nil || c.Hooks == nil {
		return ""
	}
	return c.Hooks.CurrentFilter()
}

// DoingAction reports whether tag is on the current-filter stack via c.Hooks
// (doing_action, P9.36). Nil context/hooks → false. Empty tag → any current.
func DoingAction(c *Context, tag string) bool {
	if c == nil || c.Hooks == nil {
		return false
	}
	return c.Hooks.DoingAction(tag)
}

// DoingFilter reports whether tag is on the current-filter stack via c.Hooks
// (doing_filter, P9.36). Nil context/hooks → false. Empty tag → any current.
func DoingFilter(c *Context, tag string) bool {
	if c == nil || c.Hooks == nil {
		return false
	}
	return c.Hooks.DoingFilter(tag)
}

// CurrentFilters returns a copy of the nested current_filter stack via c.Hooks
// (P9.36). Nil context/hooks → nil.
func CurrentFilters(c *Context) []string {
	if c == nil || c.Hooks == nil {
		return nil
	}
	return c.Hooks.CurrentFilters()
}

// DidAction returns how many times tag has been fired via c.Hooks (did_action, P9.37).
// Nil context/hooks → 0.
func DidAction(c *Context, tag string) int {
	if c == nil || c.Hooks == nil {
		return 0
	}
	return c.Hooks.DidAction(tag)
}

// DidFilter returns how many times tag has been applied via c.Hooks (did_filter, P9.37).
// Nil context/hooks → 0.
func DidFilter(c *Context, tag string) int {
	if c == nil || c.Hooks == nil {
		return 0
	}
	return c.Hooks.DidFilter(tag)
}
