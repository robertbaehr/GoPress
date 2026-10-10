// wp_doing_ajax (P2.66) — subset of wp-includes/load.php.
package wp

// DoingAjax reports whether the current request is a WordPress Ajax request
// (wp_doing_ajax). True when Scope.IsAjax is set (admin-ajax.php entry).
// Filterable via wp_doing_ajax.
func DoingAjax(c *Context) bool {
	doing := false
	if c != nil && c.Scope != nil {
		doing = c.Scope.IsAjax
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_doing_ajax", doing); v != nil {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return doing
}
