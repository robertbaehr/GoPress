// Mail stub (P3.9+) — wp_mail records sends; never opens a network connection
// when WP_HTTP_BLOCK_EXTERNAL / Config.BlockExternal is set.
// P2.15: pluggable pre_wp_mail / wp_mail filters.
// P2.98: From/Content-Type/charset filters + header parse + attachments record.
package wp

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
)

// MailMessage is one recorded wp_mail invocation.
type MailMessage struct {
	To          []string
	Subject     string
	Message     string
	Headers     []string
	Attachments []string
	FromEmail   string
	FromName    string
	ContentType string
	Charset     string
}

// Mailer records outbound mail for tests and debugging.
type Mailer struct {
	mu   sync.Mutex
	sent []MailMessage
}

// NewMailer returns an empty mail recorder.
func NewMailer() *Mailer {
	return &Mailer{}
}

// Sent returns a copy of recorded messages.
func (m *Mailer) Sent() []MailMessage {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MailMessage, len(m.sent))
	copy(out, m.sent)
	return out
}

// Clear drops recorded messages.
func (m *Mailer) Clear() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.sent = nil
	m.mu.Unlock()
}

func (m *Mailer) record(msg MailMessage) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.sent = append(m.sent, msg)
	m.mu.Unlock()
}

// MailAttrs is the wp_mail / pre_wp_mail attribute bag (pluggable.php subset).
type MailAttrs struct {
	To          []string
	Subject     string
	Message     string
	Headers     []string
	Attachments []string
}

// Mail sends (or records) an email — wp_mail stub.
// Applies the "wp_mail" filter to attributes, then "pre_wp_mail" short-circuit
// (returning a non-nil bool skips the stub transport / recorder).
// When Config.BlockExternal is true, the message is logged/recorded only (no SMTP).
// When BlockExternal is false, the stub still records without network (real transport is out of scope).
func Mail(c *Context, to []string, subject, message string, headers []string) (bool, error) {
	return MailFull(c, to, subject, message, headers, nil)
}

// MailFull is wp_mail with optional attachments (P2.98).
func MailFull(c *Context, to []string, subject, message string, headers, attachments []string) (bool, error) {
	atts := MailAttrs{
		To:          append([]string(nil), to...),
		Subject:     subject,
		Message:     message,
		Headers:     append([]string(nil), headers...),
		Attachments: append([]string(nil), attachments...),
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_mail", atts); v != nil {
			if filtered, ok := v.(MailAttrs); ok {
				atts = filtered
			}
		}
		// pre_wp_mail: non-nil short-circuits wp_mail (WP returns the filtered value).
		if v := c.Hooks.ApplyFilters("pre_wp_mail", nil, atts); v != nil {
			switch t := v.(type) {
			case bool:
				return t, nil
			case error:
				return false, t
			default:
				return true, nil
			}
		}
	}
	if len(atts.To) == 0 {
		return false, fmt.Errorf("wp: Mail: empty recipient")
	}

	fromEmail, fromName, contentType, charset := parseMailHeaders(atts.Headers)
	if fromEmail == "" {
		fromEmail = defaultMailFrom(c)
	}
	if fromName == "" {
		fromName = "WordPress"
	}
	if contentType == "" {
		contentType = "text/plain"
	}
	if charset == "" {
		charset = defaultMailCharset(c)
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_mail_from", fromEmail); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				fromEmail = strings.TrimSpace(s)
			}
		}
		if v := c.Hooks.ApplyFilters("wp_mail_from_name", fromName); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				fromName = strings.TrimSpace(s)
			}
		}
		if v := c.Hooks.ApplyFilters("wp_mail_content_type", contentType); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				contentType = strings.TrimSpace(s)
			}
		}
		if v := c.Hooks.ApplyFilters("wp_mail_charset", charset); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				charset = strings.TrimSpace(s)
			}
		}
	}

	msg := MailMessage{
		To:          append([]string(nil), atts.To...),
		Subject:     atts.Subject,
		Message:     atts.Message,
		Headers:     append([]string(nil), atts.Headers...),
		Attachments: append([]string(nil), atts.Attachments...),
		FromEmail:   fromEmail,
		FromName:    fromName,
		ContentType: contentType,
		Charset:     charset,
	}
	if c != nil && c.Mailer != nil {
		c.Mailer.record(msg)
	}
	blocked := c != nil && c.Config != nil && c.Config.BlockExternal
	if blocked {
		log.Printf("wp_mail blocked-external: to=%v subject=%q from=%q", atts.To, atts.Subject, fromEmail)
		return true, nil
	}
	log.Printf("wp_mail stub (no transport): to=%v subject=%q from=%q", atts.To, atts.Subject, fromEmail)
	return true, nil
}

// parseMailHeaders extracts From / Content-Type / charset from wp_mail headers (P2.98).
func parseMailHeaders(headers []string) (fromEmail, fromName, contentType, charset string) {
	for _, h := range headers {
		for _, line := range strings.Split(h, "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if line == "" {
				continue
			}
			colon := strings.IndexByte(line, ':')
			if colon <= 0 {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(line[:colon]))
			val := strings.TrimSpace(line[colon+1:])
			switch name {
			case "from":
				fromName, fromEmail = splitMailAddress(val)
			case "content-type":
				contentType, charset = parseContentType(val)
			}
		}
	}
	return fromEmail, fromName, contentType, charset
}

func splitMailAddress(raw string) (name, email string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	// "Name <addr@host>" or bare addr@host
	if i := strings.LastIndex(raw, "<"); i >= 0 && strings.HasSuffix(raw, ">") {
		name = strings.TrimSpace(raw[:i])
		name = strings.Trim(name, `"'`)
		name = strings.TrimSpace(name)
		email = strings.TrimSpace(raw[i+1 : len(raw)-1])
		return name, email
	}
	return "", raw
}

func parseContentType(raw string) (contentType, charset string) {
	parts := strings.Split(raw, ";")
	if len(parts) == 0 {
		return "", ""
	}
	contentType = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(p), "charset=") {
			charset = strings.TrimSpace(strings.Trim(p[len("charset="):], `"'`))
		}
	}
	return contentType, charset
}

func defaultMailFrom(c *Context) string {
	host := "localhost"
	if c != nil && c.Options != nil {
		home := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("home", "")))
		if home == "" {
			home = strings.TrimSpace(fmt.Sprint(c.Options.GetOption("siteurl", "")))
		}
		if u, err := url.Parse(home); err == nil && u.Host != "" {
			host = u.Hostname()
			host = strings.TrimPrefix(strings.ToLower(host), "www.")
		}
	}
	if host == "" {
		host = "localhost"
	}
	return "wordpress@" + host
}

func defaultMailCharset(c *Context) string {
	if c != nil && c.Options != nil {
		if v := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blog_charset", ""))); v != "" && v != "<nil>" {
			return v
		}
	}
	return "UTF-8"
}
