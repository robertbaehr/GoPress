package wp_test

import (
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNonceTickCeilDivision(t *testing.T) {
	const half = 43200 // DefaultNonceLife / 2
	cases := []struct {
		now  int64
		want int64
	}{
		{1, 1},
		{half, 1},
		{half + 1, 2},
		{86400, 2},
		{86400 + 1, 3},
	}
	for _, tc := range cases {
		if got := wp.NonceTick(tc.now, "x"); got != tc.want {
			t.Fatalf("now=%d got=%d want=%d", tc.now, got, tc.want)
		}
	}
}

func TestVerifyNonceCurrentAndPreviousTick(t *testing.T) {
	c := bootWP(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	c.CurrentUser = user
	c.SessionToken = "session-token-for-nonce"

	now := time.Now().Unix()
	tick := wp.NonceTick(now, "update-post")
	cur := c.Auth.CreateNonce(tick, "update-post", int(user.ID), c.SessionToken)
	prev := c.Auth.CreateNonce(tick-1, "update-post", int(user.ID), c.SessionToken)

	if got := c.VerifyNonce(cur, "update-post"); got != 1 {
		t.Fatalf("current tick: got %d want 1", got)
	}
	if got := c.VerifyNonce(prev, "update-post"); got != 2 {
		t.Fatalf("previous tick: got %d want 2", got)
	}
	if got := c.VerifyNonce("deadbeef01", "update-post"); got != 0 {
		t.Fatalf("bogus: got %d want 0", got)
	}
	if got := c.VerifyNonce("", "update-post"); got != 0 {
		t.Fatalf("empty: got %d want 0", got)
	}
	// Wrong action must fail.
	if got := c.VerifyNonce(cur, "other-action"); got != 0 {
		t.Fatalf("wrong action: got %d want 0", got)
	}
}

func TestCreateNonceRoundTrip(t *testing.T) {
	c := bootWP(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	c.CurrentUser = user
	c.SessionToken = set.Token
	nonce := c.CreateNonce("wp_rest")
	if nonce == "" || len(nonce) != 10 {
		t.Fatalf("CreateNonce=%q", nonce)
	}
	if got := c.VerifyNonce(nonce, "wp_rest"); got != 1 {
		t.Fatalf("VerifyNonce=%d want 1 (session=%q)", got, c.SessionToken)
	}
}

func TestNonceLifeFilterShortensTick(t *testing.T) {
	// P2.99: nonce_life filter changes the tick window.
	c := bootWP(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	c.CurrentUser = user
	c.SessionToken = "nonce-life-session"

	const shortLife int64 = 3600 // 1 hour → half = 1800
	seenAction := ""
	c.Hooks.AddFilter("nonce_life", func(value any, args ...any) any {
		if len(args) > 0 {
			if s, ok := args[0].(string); ok {
				seenAction = s
			}
		}
		return shortLife
	}, 10, 2)

	now := int64(1801)
	got := c.NonceTick(now, "update-post")
	want := wp.NonceTickLife(now, "update-post", shortLife)
	if got != want {
		t.Fatalf("filtered tick=%d want=%d", got, want)
	}
	if seenAction != "update-post" {
		t.Fatalf("filter action=%q", seenAction)
	}
	// Default (unfiltered) tick for same now differs when life is shortened.
	if def := wp.NonceTick(now, "update-post"); def == got {
		t.Fatalf("expected filtered tick to differ from default %d", def)
	}

	nonce := c.CreateNonce("update-post")
	if got := c.VerifyNonce(nonce, "update-post"); got != 1 {
		t.Fatalf("VerifyNonce=%d want 1", got)
	}
}
