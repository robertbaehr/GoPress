package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// handlePluginsBulk runs activate-selected / deactivate-selected (P7.63).
// Returns a query fragment notice key on success, or ("", nil) when not a bulk action.
func handlePluginsBulk(r *http.Request, c *wp.Context) (string, error) {
	if r == nil || c == nil {
		return "", nil
	}
	_ = r.ParseForm()
	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" || action == "-1" {
		action = strings.TrimSpace(r.FormValue("action2"))
	}
	switch action {
	case "activate-selected", "deactivate-selected":
		// continue
	default:
		return "", nil
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "activate_plugins") {
		return "", fmt.Errorf("sorry, you are not allowed to manage plugins for this site")
	}
	nonce := r.FormValue("_wpnonce")
	if c.VerifyNonce(nonce, "bulk-plugins") == 0 {
		return "", fmt.Errorf("invalid nonce")
	}
	files := collectCheckedPlugins(r)
	if len(files) == 0 {
		return "", nil
	}
	n := 0
	for _, file := range files {
		switch action {
		case "activate-selected":
			if plugins.IsPluginActive(c, file) {
				continue
			}
			if err := plugins.ActivatePlugin(c, file); err == nil {
				n++
			}
		case "deactivate-selected":
			if !plugins.IsPluginActive(c, file) {
				continue
			}
			if err := plugins.DeactivatePlugin(c, file); err == nil {
				n++
			}
		}
	}
	if n == 0 {
		return "", nil
	}
	if action == "activate-selected" {
		return "activate=true&bulk=1", nil
	}
	return "deactivate=true&bulk=1", nil
}

func collectCheckedPlugins(r *http.Request) []string {
	if r == nil {
		return nil
	}
	raw := r.Form["checked[]"]
	if len(raw) == 0 {
		raw = r.Form["checked"]
	}
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || strings.Contains(s, "..") || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func writePluginsBulkNav(b *strings.Builder, which string) {
	id := "bulk-action-selector-" + which
	b.WriteString(`<div class="tablenav `)
	b.WriteString(which)
	b.WriteString(`"><div class="alignleft actions bulkactions">`)
	b.WriteString(`<label for="`)
	b.WriteString(id)
	b.WriteString(`" class="screen-reader-text">Select bulk action</label>`)
	b.WriteString(`<select name="`)
	if which == "bottom" {
		b.WriteString(`action2`)
	} else {
		b.WriteString(`action`)
	}
	b.WriteString(`" id="`)
	b.WriteString(id)
	b.WriteString(`">`)
	b.WriteString(`<option value="-1">Bulk actions</option>`)
	b.WriteString(`<option value="activate-selected">Activate</option>`)
	b.WriteString(`<option value="deactivate-selected">Deactivate</option>`)
	b.WriteString(`</select> `)
	b.WriteString(`<input type="submit" id="doaction`)
	if which == "bottom" {
		b.WriteString(`2`)
	}
	b.WriteString(`" class="button action" value="Apply">`)
	b.WriteString(`</div></div>`)
	b.WriteByte('\n')
}

func pluginsBulkNoticeHTML(notice string, bulk bool) string {
	switch notice {
	case "activate":
		if bulk {
			return `<div id="message" class="updated notice is-dismissible"><p>Selected plugins <strong>activated</strong>.</p></div>` + "\n"
		}
		return `<div id="message" class="updated notice is-dismissible"><p>Plugin <strong>activated</strong>.</p></div>` + "\n"
	case "deactivate":
		if bulk {
			return `<div id="message" class="updated notice is-dismissible"><p>Selected plugins <strong>deactivated</strong>.</p></div>` + "\n"
		}
		return `<div id="message" class="updated notice is-dismissible"><p>Plugin <strong>deactivated</strong>.</p></div>` + "\n"
	default:
		return ""
	}
}
