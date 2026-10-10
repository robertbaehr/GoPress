package wp

// GetCurrentUser returns the authenticated user for this request
// (wp_get_current_user subset). Returns nil when anonymous.
func GetCurrentUser(c *Context) *User {
	if c == nil {
		return nil
	}
	return c.CurrentUser
}

// GetCurrentUserID returns the current user's ID, or 0 when not logged in
// (get_current_user_id / wp_get_current_user()->ID).
func GetCurrentUserID(c *Context) int64 {
	u := GetCurrentUser(c)
	if u == nil {
		return 0
	}
	return u.ID
}
