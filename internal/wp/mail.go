// Mail stub (P3.9) — wp_mail records sends; never opens a network connection
// when WP_HTTP_BLOCK_EXTERNAL / Config.BlockExternal is set.
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

// Mail sends (or records) an email — wp_mail stub.
// When Config.BlockExternal is true, the message is logged/recorded only (no SMTP).
// When BlockExternal is false, the stub still records without network (real transport is out of scope).
func Mail(c *Context, to []string, subject, message string, headers []string) (bool, error) {
	if len(to) == 0 {
		return false, fmt.Errorf("wp: Mail: empty recipient")
	}
	msg := MailMessage{
		To:      append([]string(nil), to...),
		Subject: subject,
		Message: message,
		Headers: append([]string(nil), headers...),
	}
	if c != nil && c.Mailer != nil {
		c.Mailer.record(msg)
	}
	blocked := c != nil && c.Config != nil && c.Config.BlockExternal
	if blocked {
		log.Printf("wp_mail blocked-external: to=%v subject=%q", to, subject)
		return true, nil
	}
	log.Printf("wp_mail stub (no transport): to=%v subject=%q", to, subject)
	return true, nil
}
