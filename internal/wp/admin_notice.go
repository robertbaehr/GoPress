package wp

import (
	"html"
	"strings"
)

// AdminNoticeArgs mirrors wp_get_admin_notice $args (P7.131).
type AdminNoticeArgs struct {
	Type              string
	Dismissible       bool
	ID                string
	AdditionalClasses []string
	Attributes        map[string]string
	ParagraphWrap     *bool // nil → true (WP default)
	SkipParagraphWrap bool  // convenience when ParagraphWrap pointer is awkward
}

func (a AdminNoticeArgs) paragraphWrap() bool {
	if a.SkipParagraphWrap {
		return false
	}
	if a.ParagraphWrap != nil {
		return *a.ParagraphWrap
	}
	return true
}

// GetAdminNotice is wp_get_admin_notice (P7.131).
// Builds `<div class="notice …">` markup with optional type/dismissible/id,
// applies wp_admin_notice_args then wp_admin_notice_markup filters.
func GetAdminNotice(c *Context, message string, args AdminNoticeArgs) string {
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_admin_notice_args", args, message)
		if a, ok := filtered.(AdminNoticeArgs); ok {
			args = a
		}
	}

	classes := []string{"notice"}
	if t := strings.TrimSpace(args.Type); t != "" && !strings.ContainsAny(t, " \t\n\r\f") {
		classes = append(classes, "notice-"+t)
	}
	if args.Dismissible {
		classes = append(classes, "is-dismissible")
	}
	for _, extra := range args.AdditionalClasses {
		extra = strings.TrimSpace(extra)
		if extra != "" {
			classes = append(classes, extra)
		}
	}

	var b strings.Builder
	b.WriteString(`<div class="`)
	b.WriteString(html.EscapeString(strings.Join(classes, " ")))
	b.WriteByte('"')
	if id := strings.TrimSpace(args.ID); id != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(id))
		b.WriteByte('"')
	}
	for name, value := range args.Attributes {
		name = strings.TrimSpace(name)
		if name == "" || name == "class" || name == "id" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(html.EscapeString(name))
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(strings.TrimSpace(value)))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	if args.paragraphWrap() {
		b.WriteString("<p>")
		b.WriteString(message)
		b.WriteString("</p>")
	} else {
		b.WriteString(message)
	}
	b.WriteString("</div>")
	markup := b.String()

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_admin_notice_markup", markup, message, args); v != nil {
			if s, ok := v.(string); ok {
				markup = s
			}
		}
	}
	return markup
}

// EmitAdminNotice is wp_admin_notice (P7.131): fires wp_admin_notice action,
// then buffers kses'd markup into the admin_notices capture via AdminNotice.
func EmitAdminNotice(c *Context, message string, args AdminNoticeArgs) {
	if c == nil {
		return
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("wp_admin_notice", message, args)
	}
	markup := GetAdminNotice(c, message, args)
	c.AdminNotice(KsesPostFiltered(c, markup))
}

// FormatSettingsSavedNotice returns the classic settings-saved admin notice (P7.131).
func FormatSettingsSavedNotice(c *Context) string {
	return GetAdminNotice(c, "<strong>Settings saved.</strong>", AdminNoticeArgs{
		Type:              "success",
		Dismissible:       true,
		ID:                "message",
		AdditionalClasses: []string{"updated"},
	})
}
