// is_login (P2.68) — subset of wp-includes/load.php.
package wp

// IsLogin reports whether the current request is for the login screen
// (is_login). True when Scope.IsLogin is set (wp-login.php entry).
func IsLogin(c *Context) bool {
	if c != nil && c.Scope != nil {
		return c.Scope.IsLogin
	}
	return false
}
