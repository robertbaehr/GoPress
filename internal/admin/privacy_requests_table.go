package admin

import (
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PrivacyRequest is one row in the privacy requests list table (P8.10 / P7.97).
type PrivacyRequest struct {
	ID        int64
	Email     string
	Status    string // request-pending | request-confirmed | request-failed | request-completed
	Requested time.Time
	NextSteps string // HTML (already escaped text or safe links)
	File      string // optional export filename
}

// PrivacyRequestsTable mirrors WP_Privacy_Requests_Table columns (P8.10).
type PrivacyRequestsTable struct {
	RequestType string // export_personal_data | remove_personal_data
	AdminURL    string
	Rows        []PrivacyRequest
}

// Columns returns column titles keyed by name (get_columns).
func (t PrivacyRequestsTable) Columns() map[string]string {
	return map[string]string{
		"cb":                `<input type="checkbox" />`,
		"email":             "Requester",
		"status":            "Status",
		"created_timestamp": "Requested",
		"next_steps":        "Next steps",
	}
}

// RenderHTML emits the wp-list-table for privacy requests.
func (t PrivacyRequestsTable) RenderHTML() string {
	return t.RenderHTMLWithFilter("")
}

// RenderHTMLWithFilter emits views + table; filterStatus is filter-status query (P7.97).
func (t PrivacyRequestsTable) RenderHTMLWithFilter(filterStatus string) string {
	cols := t.Columns()
	order := []string{"cb", "email", "status", "created_timestamp", "next_steps"}
	var b strings.Builder
	b.WriteString(t.renderViewsHTML(filterStatus))
	b.WriteString(`<table class="wp-list-table widefat fixed striped table-view-list privacy-requests requests-table">` + "\n")
	b.WriteString("<thead><tr>\n")
	for _, key := range order {
		b.WriteString(`<th scope="col" id="`)
		b.WriteString(html.EscapeString(key))
		b.WriteString(`" class="manage-column column-`)
		b.WriteString(html.EscapeString(key))
		b.WriteString(`">`)
		if key == "cb" {
			b.WriteString(cols[key])
		} else {
			b.WriteString(html.EscapeString(cols[key]))
		}
		b.WriteString("</th>\n")
	}
	b.WriteString("</tr></thead>\n<tbody id=\"the-list\">\n")
	rows := t.filteredRows(filterStatus)
	if len(rows) == 0 {
		b.WriteString(`<tr class="no-items"><td class="colspanchange" colspan="5">No requests found.</td></tr>` + "\n")
	}
	for i, row := range rows {
		statusClass := privacyStatusClass(row.Status)
		b.WriteString(`<tr class="`)
		b.WriteString(html.EscapeString(statusClass))
		b.WriteString(`">`)
		b.WriteString(`<th scope="row" class="check-column"><input type="checkbox" name="request_id[]" value="`)
		idVal := row.ID
		if idVal <= 0 {
			idVal = int64(i + 1)
		}
		b.WriteString(strconv.FormatInt(idVal, 10))
		b.WriteString(`"></th>`)
		b.WriteString(`<td class="email column-email" data-colname="Requester">`)
		b.WriteString(html.EscapeString(row.Email))
		if row.File != "" {
			b.WriteString(`<p class="request-file"><code>`)
			b.WriteString(html.EscapeString(row.File))
			b.WriteString(`</code></p>`)
		}
		b.WriteString(`</td>`)
		b.WriteString(`<td class="status column-status" data-colname="Status">`)
		b.WriteString(html.EscapeString(privacyStatusLabel(row.Status)))
		b.WriteString(`</td>`)
		b.WriteString(`<td class="created_timestamp column-created_timestamp" data-colname="Requested">`)
		if !row.Requested.IsZero() {
			b.WriteString(html.EscapeString(row.Requested.UTC().Format("2006/01/02 3:04 pm")))
		} else {
			b.WriteString("—")
		}
		b.WriteString(`</td>`)
		b.WriteString(`<td class="next_steps column-next_steps" data-colname="Next steps">`)
		if row.NextSteps != "" {
			b.WriteString(row.NextSteps)
		} else {
			b.WriteString("—")
		}
		b.WriteString(`</td></tr>` + "\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	return b.String()
}

// renderViewsHTML emits WP_Privacy_Requests_Table::get_views links (P7.97).
func (t PrivacyRequestsTable) renderViewsHTML(current string) string {
	current = strings.TrimSpace(current)
	counts := map[string]int{}
	for _, row := range t.Rows {
		counts[normalizePrivacyFilterStatus(row.Status)]++
	}
	total := len(t.Rows)
	adminURL := t.AdminURL
	if adminURL == "" {
		adminURL = "/wp-admin/erase-personal-data.php"
	}
	type view struct {
		key, label string
		count      int
		href       string
	}
	views := []view{
		{key: "", label: "All", count: total, href: adminURL},
	}
	for _, st := range []string{"request-pending", "request-confirmed", "request-failed", "request-completed"} {
		n := counts[st]
		if n == 0 {
			continue
		}
		views = append(views, view{
			key:   st,
			label: privacyStatusLabel(st),
			count: n,
			href:  adminURL + "?filter-status=" + st,
		})
	}
	var b strings.Builder
	b.WriteString(`<ul class="subsubsub">` + "\n")
	for i, v := range views {
		cls := ""
		if v.key == current || (current == "" && v.key == "") {
			cls = ` class="current"`
		}
		b.WriteString("<li class=\"")
		if v.key == "" {
			b.WriteString("all")
		} else {
			b.WriteString(html.EscapeString(v.key))
		}
		b.WriteString(`"><a href="`)
		b.WriteString(html.EscapeString(v.href))
		b.WriteString(`"`)
		b.WriteString(cls)
		b.WriteString(">")
		b.WriteString(html.EscapeString(v.label))
		b.WriteString(` <span class="count">(`)
		b.WriteString(strconv.Itoa(v.count))
		b.WriteString(`)</span></a>`)
		if i < len(views)-1 {
			b.WriteString(" |")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n")
	return b.String()
}

func (t PrivacyRequestsTable) filteredRows(filterStatus string) []PrivacyRequest {
	filterStatus = normalizePrivacyFilterStatus(filterStatus)
	if filterStatus == "" {
		return t.Rows
	}
	out := make([]PrivacyRequest, 0, len(t.Rows))
	for _, row := range t.Rows {
		if normalizePrivacyFilterStatus(row.Status) == filterStatus {
			out = append(out, row)
		}
	}
	return out
}

func normalizePrivacyFilterStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "request-pending", "pending":
		return "request-pending"
	case "request-confirmed", "confirmed":
		return "request-confirmed"
	case "request-failed", "failed":
		return "request-failed"
	case "request-completed", "complete", "completed":
		return "request-completed"
	default:
		return strings.TrimSpace(status)
	}
}

func privacyStatusClass(status string) string {
	switch status {
	case "request-pending", "pending":
		return "request-pending status-request-pending"
	case "request-confirmed", "confirmed":
		return "request-confirmed status-request-confirmed"
	case "request-failed", "failed":
		return "request-failed status-request-failed"
	case "request-completed", "complete", "completed":
		return "request-completed status-request-completed"
	default:
		return "status-" + strings.TrimSpace(status)
	}
}

func privacyStatusLabel(status string) string {
	switch status {
	case "request-pending", "pending":
		return "Pending confirmation"
	case "request-confirmed", "confirmed":
		return "Confirmed"
	case "request-failed", "failed":
		return "Failed"
	case "request-completed", "complete", "completed":
		return "Complete"
	default:
		if status == "" {
			return "—"
		}
		return status
	}
}

// renderExportPrivacyRequestsTable builds the export-personal-data list table (P8.10 / P7.98).
// filterStatus is the filter-status query arg (request-pending, …).
func renderExportPrivacyRequestsTable(c *wp.Context, pendingEmail, filterStatus string) string {
	table := PrivacyRequestsTable{
		RequestType: "export_personal_data",
		AdminURL:    "/wp-admin/export-personal-data.php",
	}
	seen := map[string]bool{}
	if c != nil {
		if rows, err := wp.ListUserRequests(c, "export_personal_data", 50); err == nil {
			for _, r := range rows {
				next := "—"
				switch r.Status {
				case "request-pending", "pending":
					next = "Waiting for confirmation"
				case "request-confirmed", "confirmed":
					next = "Download personal data"
				case "request-completed", "completed", "complete":
					next = "Complete"
				}
				table.Rows = append(table.Rows, PrivacyRequest{
					ID:        r.ID,
					Email:     r.Email,
					Status:    r.Status,
					Requested: r.Requested,
					NextSteps: next,
				})
				seen[strings.ToLower(r.Email)] = true
			}
		}
	}
	if pendingEmail != "" && !seen[strings.ToLower(pendingEmail)] {
		table.Rows = append(table.Rows, PrivacyRequest{
			Email:     pendingEmail,
			Status:    "request-pending",
			Requested: time.Now().UTC(),
			NextSteps: "Waiting for confirmation",
		})
	}
	for _, name := range wp.ListPersonalDataExportFiles(c, 20) {
		dl := wp.PersonalDataExportDownloadURL(name)
		table.Rows = append(table.Rows, PrivacyRequest{
			Email:     "—",
			Status:    "request-completed",
			File:      name,
			NextSteps: `<a class="export-personal-data-download" href="` + html.EscapeString(dl) + `">Download personal data</a>`,
		})
	}
	return table.RenderHTMLWithFilter(filterStatus)
}

// renderErasePrivacyRequestsTable lists stored user_request rows (P8.10 / P8.21 / P7.97).
// filterStatus is the filter-status query arg (request-pending, …).
func renderErasePrivacyRequestsTable(c *wp.Context, pendingEmail, filterStatus string) string {
	table := PrivacyRequestsTable{
		RequestType: "remove_personal_data",
		AdminURL:    "/wp-admin/erase-personal-data.php",
	}
	seen := map[string]bool{}
	if c != nil {
		if rows, err := wp.ListUserRequests(c, "remove_personal_data", 50); err == nil {
			for _, r := range rows {
				next := "—"
				switch r.Status {
				case "request-pending", "pending":
					next = "Waiting for confirmation"
				case "request-confirmed", "confirmed":
					next = "Erase personal data"
				case "request-completed", "completed", "complete":
					next = "Complete"
				}
				table.Rows = append(table.Rows, PrivacyRequest{
					ID:        r.ID,
					Email:     r.Email,
					Status:    r.Status,
					Requested: r.Requested,
					NextSteps: next,
				})
				seen[strings.ToLower(r.Email)] = true
			}
		}
	}
	if pendingEmail != "" && !seen[strings.ToLower(pendingEmail)] {
		table.Rows = append(table.Rows, PrivacyRequest{
			Email:     pendingEmail,
			Status:    "request-pending",
			Requested: time.Now().UTC(),
			NextSteps: "Waiting for confirmation",
		})
	}
	return table.RenderHTMLWithFilter(filterStatus)
}
