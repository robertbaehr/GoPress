package admin

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveUsers lists users and handles bulk delete stub (P7.60).
func (h *Handler) serveUsers(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method == http.MethodPost {
		h.handleUsersBulk(w, r, c)
		return
	}
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if action == "delete" {
		h.serveUsersDeleteConfirm(w, r, c)
		return
	}
	h.serveUsersList(w, r, c)
}

func (h *Handler) handleUsersBulk(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Sorry, you are not allowed to delete users.\n", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" || action == "-1" {
		action = strings.TrimSpace(r.FormValue("action2"))
	}
	// Confirmation form posts action=dodelete.
	if action == "" {
		action = strings.TrimSpace(r.FormValue("action"))
	}

	switch action {
	case "delete":
		if !c.UserCan(c.CurrentUser, "delete_users") {
			http.Error(w, "Sorry, you are not allowed to delete users.\n", http.StatusForbidden)
			return
		}
		ids := collectUserIDs(r)
		if len(ids) == 0 {
			http.Redirect(w, r, "/wp-admin/users.php", http.StatusFound)
			return
		}
		loc := "/wp-admin/users.php?action=delete"
		for _, id := range ids {
			loc += "&users[]=" + strconv.FormatInt(id, 10)
		}
		http.Redirect(w, r, loc, http.StatusFound)
		return

	case "dodelete":
		if !c.UserCan(c.CurrentUser, "delete_users") {
			http.Error(w, "Sorry, you are not allowed to delete users.\n", http.StatusForbidden)
			return
		}
		ids := collectUserIDs(r)
		deleted := 0
		update := "del"
		for _, id := range ids {
			if id == c.CurrentUser.ID {
				update = "err_admin_del"
				continue
			}
			if err := deleteAdminUser(c, id); err == nil {
				deleted++
			}
		}
		loc := "/wp-admin/users.php?update=" + update
		if deleted > 0 {
			loc += "&delete_count=" + strconv.Itoa(deleted)
		}
		http.Redirect(w, r, loc, http.StatusFound)
		return

	default:
		http.Redirect(w, r, "/wp-admin/users.php", http.StatusFound)
	}
}

func (h *Handler) serveUsersDeleteConfirm(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "delete_users") {
		http.Error(w, "Sorry, you are not allowed to delete users.\n", http.StatusForbidden)
		return
	}
	ids := collectUserIDsFromQuery(r)
	if len(ids) == 0 {
		http.Redirect(w, r, "/wp-admin/users.php", http.StatusFound)
		return
	}
	var b strings.Builder
	b.WriteString("<h1>Delete Users</h1>\n")
	if r.URL.Query().Get("error") != "" {
		b.WriteString(`<div id="message" class="error notice is-dismissible"><p><strong>Error:</strong> Please select an option.</p></div>` + "\n")
	}
	if len(ids) == 1 {
		b.WriteString("<p>You have specified this user for deletion:</p>\n")
	} else {
		b.WriteString("<p>You have specified these users for deletion:</p>\n")
	}
	b.WriteString(`<form method="post" name="updateusers" id="updateusers" action="/wp-admin/users.php" class="delete-and-reassign-users-form">` + "\n")
	b.WriteString(`<input type="hidden" name="action" value="dodelete" />` + "\n")
	b.WriteString("<ul>\n")
	for _, id := range ids {
		if id == c.CurrentUser.ID {
			continue
		}
		login := userLoginByID(c, id)
		b.WriteString("<li><input type=\"hidden\" name=\"users[]\" value=\"")
		b.WriteString(strconv.FormatInt(id, 10))
		b.WriteString("\"> ID #")
		b.WriteString(strconv.FormatInt(id, 10))
		b.WriteString(": ")
		b.WriteString(html.EscapeString(login))
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n")
	b.WriteString("<fieldset><p><legend>What should be done with content owned by these users?</legend></p>\n")
	b.WriteString("<ul style=\"list-style:none;\">\n")
	b.WriteString("<li><label><input type=\"radio\" name=\"delete_option\" value=\"delete\" checked=\"checked\"> Delete all content.</label></li>\n")
	b.WriteString("<li><label><input type=\"radio\" name=\"delete_option\" value=\"reassign\"> Attribute all content to:</label> ")
	b.WriteString("<select name=\"reassign_user\" id=\"reassign_user\"><option value=\"\">— Select —</option></select></li>\n")
	b.WriteString("</ul></fieldset>\n")
	b.WriteString(`<p class="submit"><input type="submit" name="submit" id="submit" class="button button-primary" value="Confirm Deletion"></p>` + "\n")
	b.WriteString("</form>\n")
	writeAdminPage(w, c, newAdminChrome(c, "Users", "users-php"), b.String())
}

func (h *Handler) serveUsersList(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	users, err := listUsers(c, 20)
	if err != nil {
		http.Error(w, "users: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<h1 class=\"wp-heading-inline\">Users</h1>\n")
	b.WriteString("<a href=\"/wp-admin/user-new.php\" class=\"page-title-action\">Add New User</a>\n")
	b.WriteString("<hr class=\"wp-header-end\">\n")
	if update := r.URL.Query().Get("update"); update == "add" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>New user created.</p></div>` + "\n")
	} else if update == "del" {
		n := r.URL.Query().Get("delete_count")
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		if n == "1" {
			b.WriteString("User deleted.")
		} else {
			b.WriteString(html.EscapeString(n))
			b.WriteString(" users deleted.")
		}
		b.WriteString(`</p></div>` + "\n")
	} else if update == "err_admin_del" {
		b.WriteString(`<div id="message" class="error notice is-dismissible"><p>You cannot delete the current user.</p></div>` + "\n")
	}

	b.WriteString(`<form method="post" action="/wp-admin/users.php" id="users-filter">` + "\n")
	b.WriteString(`<div class="tablenav top"><div class="alignleft actions bulkactions">`)
	b.WriteString(`<label for="bulk-action-selector-top" class="screen-reader-text">Select bulk action</label>`)
	b.WriteString(`<select name="action" id="bulk-action-selector-top">`)
	b.WriteString(`<option value="-1">Bulk actions</option>`)
	b.WriteString(`<option value="delete">Delete</option>`)
	b.WriteString(`</select> `)
	b.WriteString(`<input type="submit" id="doaction" class="button action" value="Apply" />`)
	b.WriteString(`</div></div>` + "\n")

	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list users\">\n")
	b.WriteString("<thead><tr>")
	b.WriteString(`<td id="cb" class="manage-column column-cb check-column"><input id="cb-select-all-1" type="checkbox" /></td>`)
	b.WriteString("<th scope=\"col\" class=\"manage-column column-username column-primary\">Username</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-name\">Name</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-email\">Email</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-role\">Role</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-posts\">Posts</th>")
	b.WriteString("</tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	for _, u := range users {
		b.WriteString("<tr id=\"user-")
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString("\">")
		b.WriteString(`<th scope="row" class="check-column"><input type="checkbox" name="users[]" value="`)
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString(`" /></th>`)
		b.WriteString("<td class=\"username column-username has-row-actions column-primary\" data-colname=\"Username\"><strong><a href=\"/wp-admin/user-edit.php?user_id=")
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(u.Login))
		b.WriteString("</a></strong>")
		b.WriteString("<div class=\"row-actions\"><span class=\"edit\"><a href=\"/wp-admin/user-edit.php?user_id=")
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString("\">Edit</a></span>")
		if c != nil && c.CurrentUser != nil && u.ID != c.CurrentUser.ID {
			b.WriteString(` | <span class="delete"><a class="submitdelete" href="/wp-admin/users.php?action=delete&amp;users[]=`)
			b.WriteString(strconv.FormatInt(u.ID, 10))
			b.WriteString(`">Delete</a></span>`)
		}
		b.WriteString("</div></td>")
		b.WriteString("<td class=\"name column-name\" data-colname=\"Name\">")
		b.WriteString(html.EscapeString(u.Name))
		b.WriteString("</td>")
		b.WriteString("<td class=\"email column-email\" data-colname=\"Email\"><a href=\"mailto:")
		b.WriteString(html.EscapeString(u.Email))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(u.Email))
		b.WriteString("</a></td>")
		b.WriteString("<td class=\"role column-role\" data-colname=\"Role\">")
		b.WriteString(html.EscapeString(u.Role))
		b.WriteString("</td>")
		b.WriteString("<td class=\"posts column-posts\" data-colname=\"Posts\">")
		b.WriteString(strconv.FormatInt(u.Posts, 10))
		b.WriteString("</td>")
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString(`<div class="tablenav bottom"><div class="alignleft actions bulkactions">`)
	b.WriteString(`<label for="bulk-action-selector-bottom" class="screen-reader-text">Select bulk action</label>`)
	b.WriteString(`<select name="action2" id="bulk-action-selector-bottom">`)
	b.WriteString(`<option value="-1">Bulk actions</option>`)
	b.WriteString(`<option value="delete">Delete</option>`)
	b.WriteString(`</select> `)
	b.WriteString(`<input type="submit" id="doaction2" class="button action" value="Apply" />`)
	b.WriteString(`</div></div>` + "\n")
	b.WriteString("</form>\n")
	writeAdminPage(w, c, newAdminChrome(c, "Users", "users-php"), b.String())
}

func collectUserIDs(r *http.Request) []int64 {
	if r == nil {
		return nil
	}
	raw := r.Form["users[]"]
	if len(raw) == 0 {
		raw = r.Form["users"]
	}
	if len(raw) == 0 {
		if u := strings.TrimSpace(r.FormValue("user")); u != "" {
			raw = []string{u}
		}
	}
	return parseUserIDList(raw)
}

func collectUserIDsFromQuery(r *http.Request) []int64 {
	if r == nil || r.URL == nil {
		return nil
	}
	q := r.URL.Query()
	raw := q["users[]"]
	if len(raw) == 0 {
		raw = q["users"]
	}
	if len(raw) == 0 {
		if u := strings.TrimSpace(q.Get("user")); u != "" {
			raw = []string{u}
		}
	}
	return parseUserIDList(raw)
}

func parseUserIDList(raw []string) []int64 {
	out := make([]int64, 0, len(raw))
	seen := map[int64]bool{}
	for _, s := range raw {
		id, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func userLoginByID(c *wp.Context, id int64) string {
	if c == nil || c.DB == nil || id <= 0 {
		return ""
	}
	q, args, err := c.DB.Prepare("SELECT user_login FROM "+c.DB.Users()+" WHERE ID = %d LIMIT 1", id)
	if err != nil {
		return ""
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return ""
	}
	return asString(row["user_login"])
}

// deleteAdminUser removes a user and usermeta (wp_delete_user subset stub, P7.60).
// Content reassignment is not implemented; posts remain attributed to the deleted ID.
func deleteAdminUser(c *wp.Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid user")
	}
	mq, margs, err := c.DB.Prepare("DELETE FROM "+c.DB.UserMeta()+" WHERE user_id = %d", id)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(mq, margs...); err != nil {
		return err
	}
	uq, uargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Users()+" WHERE ID = %d", id)
	if err != nil {
		return err
	}
	res, err := c.DB.Exec(uq, uargs...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	if c.Users != nil && c.Users.Cache != nil {
		c.Users.Cache.Delete(strconv.FormatInt(id, 10), "users")
	}
	return nil
}
