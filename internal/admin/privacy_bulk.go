package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// processPrivacyRequestsBulkAction handles WP_Privacy_Requests_Table::process_bulk_action (P7.150).
// Returns a notice string when a bulk action ran; empty when not a bulk POST.
func processPrivacyRequestsBulkAction(r *http.Request, c *wp.Context) string {
	if r == nil || r.Method != http.MethodPost {
		return ""
	}
	_ = r.ParseForm()
	action := strings.TrimSpace(r.Form.Get("action"))
	if action == "" || action == "-1" {
		action = strings.TrimSpace(r.Form.Get("action2"))
	}
	switch action {
	case "resend", "complete", "delete":
	default:
		return ""
	}
	ids := parsePrivacyRequestIDs(r)
	if len(ids) == 0 {
		return ""
	}
	count := 0
	failures := 0
	for _, id := range ids {
		var err error
		switch action {
		case "resend":
			err = wp.ResendUserRequestConfirmation(c, id)
		case "complete":
			err = wp.CompleteUserRequest(c, id)
		case "delete":
			err = wp.DeleteUserRequest(c, id)
		}
		if err != nil {
			failures++
		} else {
			count++
		}
	}
	switch action {
	case "resend":
		if count > 0 && failures == 0 {
			return strconv.Itoa(count) + " confirmation request(s) re-sent successfully."
		}
		if count > 0 {
			return strconv.Itoa(count) + " confirmation request(s) re-sent; " + strconv.Itoa(failures) + " failed."
		}
		return strconv.Itoa(failures) + " confirmation request(s) failed to resend."
	case "complete":
		return strconv.Itoa(count) + " request(s) marked as complete."
	case "delete":
		if count > 0 && failures == 0 {
			return strconv.Itoa(count) + " request(s) deleted successfully."
		}
		if count > 0 {
			return strconv.Itoa(count) + " request(s) deleted; " + strconv.Itoa(failures) + " failed."
		}
		return strconv.Itoa(failures) + " request(s) failed to delete."
	default:
		return ""
	}
}

func parsePrivacyRequestIDs(r *http.Request) []int64 {
	if r == nil {
		return nil
	}
	raw := r.Form["request_id[]"]
	if len(raw) == 0 {
		raw = r.Form["request_id"]
	}
	var out []int64
	seen := map[int64]bool{}
	for _, s := range raw {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// privacyListSortFromRequest reads orderby/order query args for privacy tables (P7.150).
func privacyListSortFromRequest(r *http.Request) (orderby, order string) {
	if r == nil {
		return "", ""
	}
	orderby = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("orderby")))
	switch orderby {
	case "requester", "requested":
	default:
		orderby = ""
	}
	order = strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("order")))
	if order != "ASC" && order != "DESC" {
		order = ""
	}
	return orderby, order
}
