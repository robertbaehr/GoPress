// Mail stub (P3.9+) — wp_mail records sends; never opens a network connection
// when WP_HTTP_BLOCK_EXTERNAL / Config.BlockExternal is set.
// P2.15: pluggable pre_wp_mail / wp_mail filters.
package wp

import (
	"fmt"
	"log"
	"sync"
)

// MailMessage is one recorded wp_mail invocation.
type MailMessage struct {
	To      []string
	Subject string
	Message string
	Headers []string
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
	atts := MailAttrs{
		To:      append([]string(nil), to...),
		Subject: subject,
		Message: message,
		Headers: append([]string(nil), headers...),
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
	msg := MailMessage{
		To:      append([]string(nil), atts.To...),
		Subject: atts.Subject,
		Message: atts.Message,
		Headers: append([]string(nil), atts.Headers...),
	}
	if c != nil && c.Mailer != nil {
		c.Mailer.record(msg)
	}
	blocked := c != nil && c.Config != nil && c.Config.BlockExternal
	if blocked {
		log.Printf("wp_mail blocked-external: to=%v subject=%q", atts.To, atts.Subject)
		return true, nil
	}
	log.Printf("wp_mail stub (no transport): to=%v subject=%q", atts.To, atts.Subject)
	return true, nil
}
