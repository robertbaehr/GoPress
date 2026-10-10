package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// GetUserOption reads a user option (get_user_option). Blog-specific keys are
// stored as {table_prefix}{option} in usermeta; falls back to the bare key.
func GetUserOption(c *Context, option string, userID int64) string {
	option = strings.TrimSpace(option)
	if c == nil || c.Meta == nil || option == "" {
		return ""
	}
	if userID <= 0 {
		if c.CurrentUser != nil {
			userID = c.CurrentUser.ID
		}
	}
	if userID <= 0 {
		return ""
	}
	prefix := "wp_"
	if c.DB != nil && c.DB.TablePrefix != "" {
		prefix = c.DB.TablePrefix
	}
	prefixed := prefix + option
	if v := strings.TrimSpace(fmt.Sprint(c.Meta.GetMetadata(MetaUser, userID, prefixed, true))); v != "" && v != "<nil>" {
		return applyUserOptionFilter(c, option, v)
	}
	if v := strings.TrimSpace(fmt.Sprint(c.Meta.GetMetadata(MetaUser, userID, option, true))); v != "" && v != "<nil>" {
		return applyUserOptionFilter(c, option, v)
	}
	return applyUserOptionFilter(c, option, "")
}

func applyUserOptionFilter(c *Context, option, value string) string {
	if c == nil || c.Hooks == nil {
		return value
	}
	v := c.Hooks.ApplyFilters("get_user_option_"+option, value, option)
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return value
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return value
	}
	return s
}

// UpdateUserOption writes a blog-specific user option (update_user_option with
// is_global=false) as {table_prefix}{option} usermeta.
func UpdateUserOption(c *Context, userID int64, option string, value string) error {
	option = strings.TrimSpace(option)
	if c == nil || c.Meta == nil || userID <= 0 || option == "" {
		return fmt.Errorf("wp: UpdateUserOption: invalid state")
	}
	prefix := "wp_"
	if c.DB != nil && c.DB.TablePrefix != "" {
		prefix = c.DB.TablePrefix
	}
	key := prefix + option
	_, _, _, err := c.Meta.UpdateMetadata(MetaUser, userID, key, value, "")
	return err
}

// EditPostsPerPage returns the Screen Options per-page value for a post type
// (edit_{post_type}_per_page / edit_posts_per_page filters). Default 20.
func EditPostsPerPage(c *Context, postType string) int {
	if postType == "" {
		postType = "post"
	}
	option := "edit_" + postType + "_per_page"
	perPage := 20
	if raw := strings.TrimSpace(GetUserOption(c, option, 0)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			perPage = n
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters(option, perPage); v != nil {
			switch t := v.(type) {
			case int:
				perPage = t
			case int64:
				perPage = int(t)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					perPage = n
				}
			}
		}
		if v := c.Hooks.ApplyFilters("edit_posts_per_page", perPage, postType); v != nil {
			switch t := v.(type) {
			case int:
				perPage = t
			case int64:
				perPage = int(t)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					perPage = n
				}
			}
		}
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 999 {
		perPage = 999
	}
	return perPage
}
