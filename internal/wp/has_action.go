package wp

import "strings"

// BeginNoticeCapture starts buffering AdminNotice / WriteNotice HTML on c.Hooks (P9.44).
// Nil context/hooks is a no-op.
func BeginNoticeCapture(c *Context, buf *strings.Builder) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.BeginNoticeCapture(buf)
}

// EndNoticeCapture clears the notice capture buffer on c.Hooks (P9.44).
// Nil context/hooks is a no-op.
func EndNoticeCapture(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.EndNoticeCapture()
}

// WriteNotice appends HTML to the active notice capture buffer on c.Hooks (P9.44).
// Nil context/hooks or empty html is a no-op.
func WriteNotice(c *Context, html string) {
	if c == nil || c.Hooks == nil || html == "" {
		return
	}
	c.Hooks.WriteNotice(html)
}

// DoAction fires an action on c.Hooks (do_action, P9.43).
// Nil context/hooks or empty tag is a no-op.
func DoAction(c *Context, tag string, args ...any) {
	if c == nil || c.Hooks == nil || tag == "" {
		return
	}
	c.Hooks.DoAction(tag, args...)
}

// ApplyFilters applies a filter on c.Hooks (apply_filters, P9.43).
// Nil context/hooks returns value unchanged.
func ApplyFilters(c *Context, tag string, value any, args ...any) any {
	if c == nil || c.Hooks == nil || tag == "" {
		return value
	}
	return c.Hooks.ApplyFilters(tag, value, args...)
}

// AddAction registers an action callback on c.Hooks (add_action, P9.42).
// Nil context/hooks, empty tag, or nil fn is a no-op.
func AddAction(c *Context, tag string, fn ActionFunc, priority int, acceptedArgs int) {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return
	}
	c.Hooks.AddAction(tag, fn, priority, acceptedArgs)
}

// AddFilter registers a filter callback on c.Hooks (add_filter, P9.42).
// Nil context/hooks, empty tag, or nil fn is a no-op.
func AddFilter(c *Context, tag string, fn FilterFunc, priority int, acceptedArgs int) {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return
	}
	c.Hooks.AddFilter(tag, fn, priority, acceptedArgs)
}

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

// HasFilterPriority reports whether fn is hooked on tag and returns its priority
// (has_filter with callback, P9.38). priorityExact < 0 means any priority.
func HasFilterPriority(c *Context, tag string, fn FilterFunc, priorityExact int) (priority int, ok bool) {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return 0, false
	}
	return c.Hooks.HasFilterFunc(tag, fn, priorityExact)
}

// RemoveAction removes an action callback at priority from c.Hooks (remove_action, P9.38).
// Nil context/hooks, empty tag, or nil fn → false.
func RemoveAction(c *Context, tag string, fn ActionFunc, priority int) bool {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return false
	}
	return c.Hooks.RemoveAction(tag, fn, priority)
}

// RemoveFilter removes a filter callback at priority from c.Hooks (remove_filter, P9.38).
// Nil context/hooks, empty tag, or nil fn → false.
func RemoveFilter(c *Context, tag string, fn FilterFunc, priority int) bool {
	if c == nil || c.Hooks == nil || tag == "" || fn == nil {
		return false
	}
	return c.Hooks.RemoveFilter(tag, fn, priority)
}

// RemoveAllFilters removes all filter callbacks on tag via c.Hooks (remove_all_filters, P9.40).
// priorityExact < 0 removes every priority. Nil context/hooks or empty tag → false.
func RemoveAllFilters(c *Context, tag string, priorityExact int) bool {
	if c == nil || c.Hooks == nil || tag == "" {
		return false
	}
	return c.Hooks.RemoveAllFilters(tag, priorityExact)
}

// RemoveAllActions removes all action callbacks on tag via c.Hooks (remove_all_actions, P9.40).
// priorityExact < 0 removes every priority. Nil context/hooks or empty tag → false.
func RemoveAllActions(c *Context, tag string, priorityExact int) bool {
	if c == nil || c.Hooks == nil || tag == "" {
		return false
	}
	return c.Hooks.RemoveAllActions(tag, priorityExact)
}

// ApplyFiltersDeprecated runs apply_filters_deprecated via c.Hooks (P9.41).
// Nil context/hooks returns args[0] when present, else nil.
func ApplyFiltersDeprecated(c *Context, hookName string, args []any, version, replacement, message string) any {
	if c == nil || c.Hooks == nil {
		if len(args) > 0 {
			return args[0]
		}
		return nil
	}
	return c.Hooks.ApplyFiltersDeprecated(hookName, args, version, replacement, message)
}

// DoActionDeprecated runs do_action_deprecated via c.Hooks (P9.41).
// Nil context/hooks is a no-op.
func DoActionDeprecated(c *Context, hookName string, args []any, version, replacement, message string) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.DoActionDeprecated(hookName, args, version, replacement, message)
}

// DeprecatedHookNotify records a deprecated-hook notice via c.Hooks (P9.41).
// Nil context/hooks clears LastDeprecatedHookNotice.
func DeprecatedHookNotify(c *Context, hookName, version, replacement, message string) {
	if c == nil || c.Hooks == nil {
		LastDeprecatedHookNotice = ""
		return
	}
	c.Hooks.DeprecatedHook(hookName, version, replacement, message)
}
