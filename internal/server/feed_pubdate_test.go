package server

import (
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
	// Exercise TZ conversion without mutating the shared fixture DB
	// (serve process may cache options). Use wp.GetGMTFromDate directly.
	converted, err := wp.GetGMTFromDate("2024-06-15 08:00:00", "America/New_York", "Y-m-d H:i:s")
	if err != nil {
		t.Fatal(err)
	}
	if converted != "2024-06-15 12:00:00" {
		t.Fatalf("GetGMTFromDate got %q want 2024-06-15 12:00:00", converted)
	}
	got := feedRFC822(nil, converted, "")
	want := "Sat, 15 Jun 2024 12:00:00 +0000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFeedAtomDateUTC(t *testing.T) {
	got := feedAtomDate(nil, "2024-06-15 12:00:00", "")
	if got != "2024-06-15T12:00:00Z" {
		t.Fatalf("got %q", got)
	}
}
