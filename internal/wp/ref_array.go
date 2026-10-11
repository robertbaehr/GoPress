package wp

// DoActionRefArray fires tag with unpacked args via c.Hooks (do_action_ref_array, P9.39).
// Nil context/hooks or empty tag → no-op.
func DoActionRefArray(c *Context, tag string, args []any) {
	if c == nil || c.Hooks == nil || tag == "" {
		return
	}
	c.Hooks.DoActionRefArray(tag, args)
}

// ApplyFiltersRefArray filters value/args via c.Hooks (apply_filters_ref_array, P9.39).
// Nil context/hooks → first arg (or nil when args empty). Empty tag → first arg unchanged.
func ApplyFiltersRefArray(c *Context, tag string, args []any) any {
	if len(args) == 0 {
		if c == nil || c.Hooks == nil || tag == "" {
			return nil
		}
		return c.Hooks.ApplyFiltersRefArray(tag, args)
	}
	if c == nil || c.Hooks == nil || tag == "" {
		return args[0]
	}
	return c.Hooks.ApplyFiltersRefArray(tag, args)
}
