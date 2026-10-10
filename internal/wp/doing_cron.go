// wp_doing_cron (P2.67) — subset of wp-includes/load.php.
package wp

// DoingCron reports whether the current request is a WordPress cron request
// (wp_doing_cron). True when Scope.IsCron is set (wp-cron.php entry).
// Filterable via wp_doing_cron.
func DoingCron(c *Context) bool {
	doing := false
	if c != nil && c.Scope != nil {
		doing = c.Scope.IsCron
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_doing_cron", doing); v != nil {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return doing
}
