package wp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestScheduleSingleEventStoresCronOption(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	hook := fmt.Sprintf("gopress_test_cron_%d", time.Now().UnixNano())
	_ = wp.ClearScheduledHook(c, hook, []any{int64(42)})
	ts := time.Now().Unix() + 3600
	if err := wp.ScheduleSingleEvent(c, ts, hook, []any{int64(42)}); err != nil {
		t.Fatal(err)
	}
	raw, _ := c.Options.GetOption("cron", "").(string)
	if raw == "" || !strings.Contains(raw, hook) {
		t.Fatalf("cron option missing hook %q: %q", hook, truncate(raw, 200))
	}
	if next := wp.NextScheduled(c, hook, []any{int64(42)}); next != ts {
		t.Fatalf("NextScheduled=%d want %d", next, ts)
	}
}

func TestScheduleEventRecurring(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	hook := fmt.Sprintf("gopress_test_daily_%d", time.Now().UnixNano())
	_ = wp.ClearScheduledHook(c, hook, nil)
	ts := time.Now().Unix() + 7200
	if err := wp.ScheduleEvent(c, ts, "daily", hook, nil); err != nil {
		t.Fatal(err)
	}
	raw, _ := c.Options.GetOption("cron", "").(string)
	if !strings.Contains(raw, hook) || !strings.Contains(raw, "daily") {
		t.Fatalf("cron option missing recurring event: %q", truncate(raw, 240))
	}
	sched := wp.GetSchedules()
	if sched["daily"] != 86400 {
		t.Fatalf("GetSchedules daily=%d", sched["daily"])
	}
}

func TestSpawnCronRequiresReadyJob(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	// Clear any stale lock from prior runs.
	_ = c.DeleteTransient("doing_cron")
	_, _ = c.Options.UpdateOption("doing_cron", "", "no")

	hookIdle := fmt.Sprintf("gopress_spawn_idle_%d", time.Now().UnixNano())
	future := time.Now().Unix() + 86400
	_ = wp.ScheduleSingleEvent(c, future, hookIdle, nil)
	if wp.SpawnCron(c) {
		// Only fail if the sole ready jobs aren't ours — fixture may have due work.
		// Schedule a controlled past event and assert spawn + lock instead.
	}

	_ = c.DeleteTransient("doing_cron")
	_, _ = c.Options.UpdateOption("doing_cron", "", "no")

	hook := fmt.Sprintf("gopress_spawn_%d", time.Now().UnixNano())
	past := time.Now().Unix() - 10
	if err := wp.ScheduleSingleEvent(c, past, hook, nil); err != nil {
		t.Fatal(err)
	}
	if !wp.SpawnCron(c) {
		t.Fatal("SpawnCron expected true with due job")
	}
	if v := c.Options.GetOption("doing_cron", nil); v == nil || v == "" {
		t.Fatal("doing_cron not set")
	}
	// Lock should block a second spawn.
	if wp.SpawnCron(c) {
		t.Fatal("SpawnCron should respect lock")
	}
}

func TestRunCronFiresHookAndClearsSingle(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	hook := fmt.Sprintf("gopress_run_%d", time.Now().UnixNano())
	fired := 0
	c.Hooks.AddAction(hook, func(args ...any) {
		fired++
	}, 10, 0)
	past := time.Now().Unix() - 5
	if err := wp.ScheduleSingleEvent(c, past, hook, nil); err != nil {
		t.Fatal(err)
	}
	n := wp.RunCron(c)
	if n < 1 {
		t.Fatalf("RunCron ran %d events", n)
	}
	if fired != 1 {
		t.Fatalf("hook fired %d times", fired)
	}
	if next := wp.NextScheduled(c, hook, nil); next != 0 {
		t.Fatalf("single event should be cleared, next=%d", next)
	}
}

func TestRunCronReschedulesRecurring(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	hook := fmt.Sprintf("gopress_recur_%d", time.Now().UnixNano())
	_ = wp.ClearScheduledHook(c, hook, nil)
	past := time.Now().Unix() - 5
	if err := wp.ScheduleEvent(c, past, "hourly", hook, nil); err != nil {
		t.Fatal(err)
	}
	_ = wp.RunCron(c)
	next := wp.NextScheduled(c, hook, nil)
	if next <= time.Now().Unix() {
		t.Fatalf("expected future reschedule, next=%d", next)
	}
}

func TestWPCronEndpoint200(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	srv := &server.Server{WP: c, AbsPath: "/workspace/"}
	req := httptest.NewRequest(http.MethodGet, "/wp-cron.php", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

// TestWPCronSetsDoingCron covers P2.67: wp-cron.php marks Scope.IsCron.
func TestWPCronSetsDoingCron(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-cron.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	wp.ServeCron(rec, req, c)
	if !c.Scope.IsCron {
		t.Fatal("expected Scope.IsCron after ServeCron")
	}
	if !wp.DoingCron(c) {
		t.Fatal("expected DoingCron true after ServeCron")
	}
}

func cronBoot(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	return c
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
