package wp

// HasAction reports whether tag has at least one registered callback on c.Hooks
// (has_action, P9.35). Nil context/hooks or empty tag → false.
func HasAction(c *Context, tag string) bool {
	if c == nil || c.Hooks == nil || tag == "" {
		return false
	}
	return c.Hooks.HasAction(tag)
}

// HasFilter reports whether tag has at least one registered callback on c.Hooks
// (has_filter, P9.35). Nil context/hooks or empty tag → false.
func HasFilter(c *Context, tag string) bool {
	if c == nil || c.Hooks == nil || tag == "" {
		return false
	}
	return c.Hooks.HasFilter(tag)
}

// HasActionPriority reports whether fn is hooked on tag and returns its priority
// (has_action with callback, P9.35). priorityExact < 0 means any priority.
func HasActionPriority(c *Context, tag string, fn ActionFunc, priorityExact int) (priority int, ok bool) {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return 0, false
	}
	return c.Hooks.HasActionFunc(tag, fn, priorityExact)
}
