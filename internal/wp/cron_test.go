package wp_test

import (
	"context"
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

	hook := "gopress_test_cron_" + strings.ReplaceAll(t.Name(), "/", "_")
	ts := time.Now().Unix() + 3600
	if err := wp.ScheduleSingleEvent(c, ts, hook, []any{int64(42)}); err != nil {
		t.Fatal(err)
	}
	raw, _ := c.Options.GetOption("cron", "").(string)
	if raw == "" || !strings.Contains(raw, hook) {
		t.Fatalf("cron option missing hook %q: %q", hook, truncate(raw, 200))
	}
}

func TestScheduleEventRecurring(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()

	hook := "gopress_test_daily_" + strings.ReplaceAll(t.Name(), "/", "_")
	ts := time.Now().Unix() + 7200
	if err := wp.ScheduleEvent(c, ts, "daily", hook, nil); err != nil {
		t.Fatal(err)
	}
	raw, _ := c.Options.GetOption("cron", "").(string)
	if !strings.Contains(raw, hook) || !strings.Contains(raw, "daily") {
		t.Fatalf("cron option missing recurring event: %q", truncate(raw, 240))
	}
}

func TestSpawnCron(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()
	if !wp.SpawnCron(c) {
		t.Fatal("SpawnCron expected true")
	}
	if v := c.Options.GetOption("doing_cron", nil); v == nil {
		t.Fatal("doing_cron not set")
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
