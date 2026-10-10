package wp

// GetCurrentUser returns the authenticated user for this request
// (wp_get_current_user subset). Returns nil when anonymous or ID <= 0.
func GetCurrentUser(c *Context) *User {
	if c == nil || c.CurrentUser == nil || c.CurrentUser.ID <= 0 {
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

// SetCurrentUser sets Context.CurrentUser by ID (wp_set_current_user subset).
// ID <= 0 clears the user (anonymous). Fires the set_current_user action with
// the resulting user ID (0 when cleared). Returns the new current user or nil.
func SetCurrentUser(c *Context, id int64) *User {
	if c == nil {
		return nil
	}
	// Short-circuit when already set to the same user (WP wp_set_current_user).
	if id > 0 && c.CurrentUser != nil && c.CurrentUser.ID == id {
		return c.CurrentUser
	}
	if id <= 0 {
		c.CurrentUser = nil
		if c.Hooks != nil {
			c.Hooks.DoAction("set_current_user", int64(0))
		}
		return nil
	}
	user, err := c.GetUserBy("id", id)
	if err != nil || user == nil || user.ID <= 0 {
		c.CurrentUser = nil
		if c.Hooks != nil {
			c.Hooks.DoAction("set_current_user", int64(0))
		}
		return nil
	}
	c.CurrentUser = user
	if c.Hooks != nil {
		c.Hooks.DoAction("set_current_user", user.ID)
	}
	return user
}
