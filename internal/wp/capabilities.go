// Roles and capabilities (P2.3) — load wp_user_roles, UserCan / map_meta_cap subset.
package wp

import (
	"fmt"
	"strings"
)

// Role is a WordPress role (name + capability map).
type Role struct {
	Name         string
	Capabilities map[string]bool
}

// Roles holds site roles from the wp_user_roles option and applies UserCan checks.
type Roles struct {
	Options *Options
	roleMap map[string]*Role
}

// Caps denied by GoPress / oracle parity (gopress-parity mu-plugin).
var deniedCaps = map[string]bool{
	"install_plugins":  true,
	"upload_plugins":   true,
	"edit_plugins":     true,
	"delete_plugins":   true,
	"update_plugins":   true,
	"install_themes":   true,
	"upload_themes":    true,
	"edit_themes":      true,
	"update_themes":    true,
	"update_core":      true,
	"update_languages": true,
	"customize":        true,
}

// NewRoles builds a Roles API. Call Load before use.
func NewRoles(opts *Options) *Roles {
	return &Roles{Options: opts, roleMap: map[string]*Role{}}
}

// Load reads and unserializes the wp_user_roles option.
func (r *Roles) Load() error {
	if r == nil || r.Options == nil {
		return fmt.Errorf("wp: Roles.Load: nil Roles/Options")
	}
	raw := r.Options.GetOption("wp_user_roles", "")
	s, _ := raw.(string)
	if s == "" {
		r.roleMap = map[string]*Role{}
		return nil
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return fmt.Errorf("wp: Roles.Load: %w", err)
	}
	outer, ok := v.(map[any]any)
	if !ok {
		return fmt.Errorf("wp: Roles.Load: expected array")
	}
	out := make(map[string]*Role, len(outer))
	for k, roleVal := range outer {
		slug := fmt.Sprint(k)
		rm, ok := roleVal.(map[any]any)
		if !ok {
			continue
		}
		name := ""
		if n, ok := rm["name"].(string); ok {
			name = n
		}
		caps := map[string]bool{}
		if c, ok := rm["capabilities"]; ok {
			caps = phpMapStringBool(c)
		}
		out[slug] = &Role{Name: name, Capabilities: caps}
	}
	r.roleMap = out
	return nil
}

// Role returns a role by slug, or nil.
func (r *Roles) Role(slug string) *Role {
	if r == nil {
		return nil
	}
	return r.roleMap[slug]
}

// All returns a copy of the role map keyed by slug.
func (r *Roles) All() map[string]*Role {
	if r == nil {
		return nil
	}
	out := make(map[string]*Role, len(r.roleMap))
	for k, v := range r.roleMap {
		out[k] = v
	}
	return out
}

// InitUser loads wp_capabilities usermeta and merges role caps into user.AllCaps.
func (r *Roles) InitUser(user *User, meta *Meta) error {
	if user == nil {
		return fmt.Errorf("wp: InitUser: nil user")
	}
	if meta == nil {
		user.Caps = map[string]bool{}
		user.AllCaps = map[string]bool{}
		user.Roles = nil
		return nil
	}
	raw := meta.GetMetadata(MetaUser, user.ID, "wp_capabilities", true)
	s, _ := raw.(string)
	caps := map[string]bool{}
	if s != "" {
		v, err := phpUnserialize(s)
		if err != nil {
			return fmt.Errorf("wp: InitUser: capabilities: %w", err)
		}
		caps = phpMapStringBool(v)
	}
	user.Caps = caps

	all := make(map[string]bool)
	var roles []string
	for name, granted := range caps {
		if !granted {
			continue
		}
		if role := r.Role(name); role != nil {
			roles = append(roles, name)
			for cap, ok := range role.Capabilities {
				if ok {
					all[cap] = true
				}
			}
		} else {
			// Direct capability grant on the user.
			all[name] = true
		}
	}
	user.Roles = roles
	user.AllCaps = all
	return nil
}

// MapMetaCap returns primitive caps required for cap (subset + GoPress denials).
// Object IDs (e.g. post ID for edit_post) are accepted for API compatibility but
// ignored here — use Context.MapMetaCap for ownership-aware meta caps (P1.13).
func (r *Roles) MapMetaCap(cap string, userID int64, _ ...int64) []string {
	_ = userID
	if deniedCaps[cap] {
		return []string{"do_not_allow"}
	}
	switch cap {
	case "edit_post", "edit_page", "delete_post", "delete_page":
		// Without a Context/post lookup, meta caps cannot be resolved safely.
		return []string{"do_not_allow"}
	case "manage_links":
		if r != nil && r.Options != nil {
			v := r.Options.GetOption("link_manager_enabled", "0")
			if optionTruthy(v) {
				return []string{cap}
			}
		}
		return []string{"do_not_allow"}
	case "customize":
		// WordPress maps customize → edit_theme_options; parity filter denies first.
		return []string{"do_not_allow"}
	default:
		return []string{cap}
	}
}

// UserCan reports whether user has capability (current_user_can / user_can subset).
// For object meta caps (edit_post / delete_post), prefer Context.UserCan with an ID.
func (r *Roles) UserCan(user *User, cap string, args ...int64) bool {
	if user == nil || cap == "" {
		return false
	}
	if user.AllCaps == nil {
		return false
	}
	required := r.MapMetaCap(cap, user.ID, args...)
	return userHasCaps(user, required)
}

func userHasCaps(user *User, required []string) bool {
	for _, need := range required {
		if need == "do_not_allow" {
			return false
		}
		if need == "exist" {
			continue
		}
		if user == nil || !user.AllCaps[need] {
			return false
		}
	}
	return true
}

// MapMetaCap maps a capability to primitives, including edit_post/delete_post
// ownership based on post_author (P1.13) and edit_user / app-password meta caps (P2.23).
func (c *Context) MapMetaCap(cap string, userID int64, args ...int64) []string {
	if c == nil || c.Roles == nil {
		return []string{"do_not_allow"}
	}
	switch cap {
	case "edit_post", "edit_page":
		return c.mapEditDeletePost(cap, userID, args...)
	case "delete_post", "delete_page":
		return c.mapEditDeletePost(cap, userID, args...)
	case "edit_user":
		return c.mapEditUser(userID, args...)
	case "create_app_password", "list_app_passwords", "read_app_password",
		"edit_app_password", "delete_app_passwords", "delete_app_password":
		return c.mapEditUser(userID, args...)
	default:
		return c.Roles.MapMetaCap(cap, userID, args...)
	}
}

// mapEditUser mirrors WP map_meta_cap('edit_user'): self-edit needs no primitive;
// otherwise edit_users is required.
func (c *Context) mapEditUser(userID int64, args ...int64) []string {
	if userID < 1 {
		return []string{"do_not_allow"}
	}
	if len(args) > 0 && args[0] == userID {
		return []string{"exist"}
	}
	if len(args) == 0 {
		return []string{"edit_users"}
	}
	return []string{"edit_users"}
}

func (c *Context) mapEditDeletePost(cap string, userID int64, args ...int64) []string {
	if len(args) == 0 || args[0] <= 0 {
		return []string{"do_not_allow"}
	}
	post, err := GetPostByID(c, args[0])
	if err != nil || post == nil {
		return []string{"do_not_allow"}
	}
	isEdit := strings.HasPrefix(cap, "edit_")
	// Revisions: edit maps to the parent post; delete is never allowed.
	if post.Type == "revision" {
		if !isEdit {
			return []string{"do_not_allow"}
		}
		if post.Parent <= 0 {
			return []string{"do_not_allow"}
		}
		parent, err := GetPostByID(c, post.Parent)
		if err != nil || parent == nil {
			return []string{"do_not_allow"}
		}
		post = parent
	}
	caps := postTypeCaps(post.Type)
	owns := post.Author != 0 && post.Author == userID
	if owns {
		if isEdit {
			if post.Status == "publish" || post.Status == "future" {
				return []string{caps.editPublished}
			}
			return []string{caps.edit}
		}
		if post.Status == "publish" || post.Status == "future" {
			return []string{caps.deletePublished}
		}
		return []string{caps.delete}
	}
	// Editing / deleting someone else's post.
	if isEdit {
		out := []string{caps.editOthers}
		if post.Status == "publish" || post.Status == "future" {
			out = append(out, caps.editPublished)
		} else if post.Status == "private" {
			out = append(out, caps.editPrivate)
		}
		return out
	}
	out := []string{caps.deleteOthers}
	if post.Status == "publish" || post.Status == "future" {
		out = append(out, caps.deletePublished)
	} else if post.Status == "private" {
		out = append(out, caps.deletePrivate)
	}
	return out
}

type postCapSet struct {
	edit, editOthers, editPublished, editPrivate         string
	delete, deleteOthers, deletePublished, deletePrivate string
}

func postTypeCaps(postType string) postCapSet {
	if postType == "page" {
		return postCapSet{
			edit: "edit_pages", editOthers: "edit_others_pages",
			editPublished: "edit_published_pages", editPrivate: "edit_private_pages",
			delete: "delete_pages", deleteOthers: "delete_others_pages",
			deletePublished: "delete_published_pages", deletePrivate: "delete_private_pages",
		}
	}
	return postCapSet{
		edit: "edit_posts", editOthers: "edit_others_posts",
		editPublished: "edit_published_posts", editPrivate: "edit_private_posts",
		delete: "delete_posts", deleteOthers: "delete_others_posts",
		deletePublished: "delete_published_posts", deletePrivate: "delete_private_posts",
	}
}

// UserCan is a Context helper: ensures caps are loaded, then checks.
// Optional args are object IDs (e.g. post ID for edit_post / delete_post).
func (c *Context) UserCan(user *User, cap string, args ...int64) bool {
	if c == nil || c.Roles == nil || user == nil {
		return false
	}
	if user.AllCaps == nil {
		_ = c.Roles.InitUser(user, c.Meta)
	}
	required := c.MapMetaCap(cap, user.ID, args...)
	return userHasCaps(user, required)
}

// CurrentUserCan mirrors current_user_can() for the authenticated user (P2.23).
func (c *Context) CurrentUserCan(cap string, args ...int64) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, cap, args...)
}

// GetUserBy loads a user and initializes role/capability maps when Roles/Meta exist.
func (c *Context) GetUserBy(field string, value any) (*User, error) {
	if c == nil || c.Users == nil {
		return nil, fmt.Errorf("wp: GetUserBy: nil Context/Users")
	}
	user, err := c.Users.GetUserBy(field, value)
	if err != nil || user == nil {
		return user, err
	}
	if c.Roles != nil {
		if err := c.Roles.InitUser(user, c.Meta); err != nil {
			return user, err
		}
	}
	return user, nil
}

func optionTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case int64:
		return t != 0
	case int:
		return t != 0
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0" && !strings.EqualFold(s, "false") && !strings.EqualFold(s, "off") && !strings.EqualFold(s, "no")
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		return s != "" && s != "0"
	}
}
