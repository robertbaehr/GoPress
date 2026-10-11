package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.142: changing new_admin_email stages confirmation; adminhash confirms.
func TestAdminEmailVerifyConfirmP7142(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	origEmail := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", "")))
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("admin_email", origEmail, "yes")
		_, _ = c.Options.DeleteOption("adminhash")
		_, _ = c.Options.DeleteOption("new_admin_email")
	})

	form := url.Values{}
	form.Set("option_page", "general")
	form.Set("new_admin_email", "pending-admin@example.test")
	form.Set("submit", "Save Changes")
	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options.php") {
		t.Fatal("options.php not handled")
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", ""))); got != origEmail {
		t.Fatalf("admin_email updated early: %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("new_admin_email", ""))); got != "pending-admin@example.test" {
		t.Fatalf("new_admin_email=%v", got)
	}
	if len(c.Mailer.Sent()) == 0 {
		t.Fatal("expected confirmation email")
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-general.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-general.php") {
		t.Fatal("options-general.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "pending-admin@example.test") || !strings.Contains(body, "pending change") {
		t.Fatalf("expected pending notice: %q", body[:min(600, len(body))])
	}

	raw := c.Options.GetOption("adminhash", nil)
	details, ok := unserializeAdminHash(t, raw)
	if !ok {
		t.Fatalf("adminhash missing/invalid: %#v", raw)
	}
	hash := strings.TrimSpace(fmt.Sprint(details["hash"]))
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options.php?adminhash="+url.QueryEscape(hash), nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options.php") {
		t.Fatal("adminhash not handled")
	}
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "admin_email_updated=true") {
		t.Fatalf("confirm redirect: status=%d loc=%q", rec.Code, rec.Header().Get("Location"))
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", ""))); got != "pending-admin@example.test" {
		t.Fatalf("admin_email after confirm=%v", got)
	}
}

// P7.142: dismiss=new_admin_email clears pending change.
func TestAdminEmailVerifyDismissP7142(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	origEmail := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", "")))
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("admin_email", origEmail, "yes")
		_, _ = c.Options.DeleteOption("adminhash")
		_, _ = c.Options.DeleteOption("new_admin_email")
	})
	_, _ = c.Options.UpdateOption("new_admin_email", "dismiss-me@example.test", "yes")
	serialized, err := wp.SerializePHP(map[string]any{"hash": "abc", "newemail": "dismiss-me@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.Options.UpdateOption("adminhash", serialized, "no")

	h := &admin.Handler{}
	// Authenticate before CreateNonce so uid matches VerifyNonce after requireLogin.
	probe := httptest.NewRequest(http.MethodGet, "/wp-admin/options-general.php", nil)
	probe.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	probeRec := httptest.NewRecorder()
	pc := c.ForRequest(probeRec, probe)
	pc.AuthenticateRequest()
	nonce := pc.CreateNonce("dismiss-1-new_admin_email")
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options.php?dismiss=new_admin_email&_wpnonce="+url.QueryEscape(nonce), nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options.php") {
		t.Fatal("dismiss not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("new_admin_email", ""))); got != "" && got != "<nil>" {
		t.Fatalf("new_admin_email still set: %v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", ""))); got != origEmail {
		t.Fatalf("admin_email changed on dismiss: %v", got)
	}
}

func unserializeAdminHash(t *testing.T, raw any) (map[string]any, bool) {
	t.Helper()
	switch v := raw.(type) {
	case map[string]any:
		return v, true
	case map[any]any:
		out := map[string]any{}
		for k, val := range v {
			out[fmt.Sprint(k)] = val
		}
		return out, true
	case string:
		parsed, err := wp.UnserializePHP(v)
		if err != nil {
			t.Fatalf("unserialize: %v", err)
		}
		switch m := parsed.(type) {
		case map[string]any:
			return m, true
		case map[any]any:
			out := map[string]any{}
			for k, val := range m {
				out[fmt.Sprint(k)] = val
			}
			return out, true
		}
		return nil, false
	default:
		return nil, false
	}
}
