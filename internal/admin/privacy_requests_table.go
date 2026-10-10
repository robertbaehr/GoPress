package admin

import (
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PrivacyRequest is one row in the privacy requests list table (P8.10).
type PrivacyRequest struct {
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
	cols := t.Columns()
	order := []string{"cb", "email", "status", "created_timestamp", "next_steps"}
	var b strings.Builder
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
	if len(t.Rows) == 0 {
		b.WriteString(`<tr class="no-items"><td class="colspanchange" colspan="5">No requests found.</td></tr>` + "\n")
	}
	for i, row := range t.Rows {
		statusClass := privacyStatusClass(row.Status)
		b.WriteString(`<tr class="`)
		b.WriteString(html.EscapeString(statusClass))
		b.WriteString(`">`)
		b.WriteString(`<th scope="row" class="check-column"><input type="checkbox" name="request_id[]" value="`)
		b.WriteString(strconv.Itoa(i + 1))
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

// renderExportPrivacyRequestsTable builds the export-personal-data list table (P8.10).
func renderExportPrivacyRequestsTable(c *wp.Context, pendingEmail string) string {
	table := PrivacyRequestsTable{
		RequestType: "export_personal_data",
		AdminURL:    "/wp-admin/export-personal-data.php",
	}
	if pendingEmail != "" {
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
	return table.RenderHTML()
}

// renderErasePrivacyRequestsTable builds the erase-personal-data list table (P8.10).
func renderErasePrivacyRequestsTable(pendingEmail string) string {
	table := PrivacyRequestsTable{
		RequestType: "remove_personal_data",
		AdminURL:    "/wp-admin/erase-personal-data.php",
	}
	if pendingEmail != "" {
		table.Rows = append(table.Rows, PrivacyRequest{
			Email:     pendingEmail,
			Status:    "request-pending",
			Requested: time.Now().UTC(),
			NextSteps: "Waiting for confirmation",
		})
	}
	return table.RenderHTML()
}
