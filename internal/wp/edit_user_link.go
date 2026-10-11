package wp

import "strconv"

// GetEditUserLink builds get_edit_user_link (P3.159).
// userID 0 uses the current user. Requires edit_user capability.
// Self-edit → profile.php; otherwise user-edit.php?user_id=N.
// Applies the get_edit_user_link filter.
func GetEditUserLink(c *Context, userID int64) string {
	if c == nil {
		return ""
	}
	if userID <= 0 {
		if c.CurrentUser == nil {
			return ""
		}
		userID = c.CurrentUser.ID
	}
	if userID <= 0 {
		return ""
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_user", userID) {
		return ""
	}
	user, err := c.GetUserBy("id", strconv.FormatInt(userID, 10))
	if err != nil || user == nil {
		return ""
	}
	var link string
	if c.CurrentUser.ID == user.ID {
		link = AdminURL(c, "profile.php")
	} else {
		link = AdminURL(c, "user-edit.php?user_id="+strconv.FormatInt(user.ID, 10))
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_edit_user_link", link, user.ID); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}
