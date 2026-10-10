package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionTokensDestroyOthersAndAll(t *testing.T) {
	c := bootWP(t)
	now := time.Now().Unix()
	uid := int64(1)

	t1, err := c.CreateSessionToken(uid, now+3600, now)
	if err != nil {
		t.Fatalf("t1: %v", err)
	}
	t2, err := c.CreateSessionToken(uid, now+3600, now)
	if err != nil {
		t.Fatalf("t2: %v", err)
	}
	all, err := c.GetAllSessionTokens(uid, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 2 {
		t.Fatalf("expected >=2 sessions, got %d", len(all))
	}
	if err := c.DestroyOtherSessionTokens(uid, t1); err != nil {
		t.Fatalf("destroy others: %v", err)
	}
	ok, err := c.VerifySessionToken(uid, t1, now)
	if err != nil || !ok {
		t.Fatalf("kept token missing: ok=%v err=%v", ok, err)
	}
	ok, err = c.VerifySessionToken(uid, t2, now)
	if err != nil || ok {
		t.Fatalf("other token should be gone: ok=%v err=%v", ok, err)
	}
	if err := c.DestroyAllSessionTokens(uid); err != nil {
		t.Fatalf("destroy all: %v", err)
	}
	remaining, err := c.GetAllSessionTokens(uid, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty after destroy all, got %#v", remaining)
	}
	ok, _ = c.VerifySessionToken(uid, t1, now)
	if ok {
		t.Fatal("token should be invalid after destroy all")
	}
}

func TestSessionTokensStoreIPAndUA(t *testing.T) {
	c := bootWP(t)
	now := time.Now().Unix()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:443"
	req.Header.Set("User-Agent", "GoPressSessionTest/1.0")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)

	token, err := rc.CreateSessionToken(1, now+3600, now)
	if err != nil {
		t.Fatal(err)
	}
	all, err := rc.GetAllSessionTokens(1, now)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range all {
		if s.IP == "203.0.113.9" && s.UA == "GoPressSessionTest/1.0" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ip/ua on session, got %#v (token=%s)", all, token)
	}
	_ = rc.DestroyAllSessionTokens(1)
}

func TestSessionTokensExpiredFiltered(t *testing.T) {
	c := bootWP(t)
	now := time.Now().Unix()
	token, err := c.CreateSessionToken(1, now-10, now-20)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := c.VerifySessionToken(1, token, now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expired token should fail verify")
	}
	all, err := c.GetAllSessionTokens(1, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range all {
		if s.Expiration < now {
			t.Fatalf("GetAll should omit expired: %#v", s)
		}
	}
}
