package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedRFC822PrefersGMTAsUTC(t *testing.T) {
	c := &wp.Context{Config: &config.Config{}}
	got := feedRFC822(c, "2024-06-15 12:00:00", "2024-06-15 08:00:00")
	want := "Sat, 15 Jun 2024 12:00:00 +0000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFeedRFC822SkipsZeroGMT(t *testing.T) {
	c := &wp.Context{Config: &config.Config{}}
	got := feedRFC822(c, "0000-00-00 00:00:00", "2024-06-15 12:00:00")
	want := "Sat, 15 Jun 2024 12:00:00 +0000"
	if got != want {
		t.Fatalf("zero GMT should fall back to local-as-UTC, got %q", got)
	}
}

func TestFeedRFC822LocalViaTimezoneString(t *testing.T) {
	// America/New_York is UTC-4 in June (EDT). Local 08:00 → GMT 12:00.
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
	}
	boot, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer boot.DB.Close()
	prevTZ := fmt.Sprint(boot.Options.GetOption("timezone_string", ""))
	prevOff := fmt.Sprint(boot.Options.GetOption("gmt_offset", "0"))
	if prevTZ == "<nil>" {
		prevTZ = ""
	}
	if prevOff == "<nil>" {
		prevOff = "0"
	}
	if _, err := boot.Options.UpdateOption("timezone_string", "America/New_York", "yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := boot.Options.UpdateOption("gmt_offset", "0", "yes"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = boot.Options.UpdateOption("timezone_string", prevTZ, "yes")
		_, _ = boot.Options.UpdateOption("gmt_offset", prevOff, "yes")
	})

	got := feedRFC822(boot, "", "2024-06-15 08:00:00")
	want := "Sat, 15 Jun 2024 12:00:00 +0000"
	if got != want {
		t.Fatalf("local via timezone_string: got %q want %q", got, want)
	}
}

func TestFeedAtomDateUTC(t *testing.T) {
	got := feedAtomDate(nil, "2024-06-15 12:00:00", "")
	if got != "2024-06-15T12:00:00Z" {
		t.Fatalf("got %q", got)
	}
}
