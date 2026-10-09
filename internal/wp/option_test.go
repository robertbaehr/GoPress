package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetOptionFromAlloptionsCache(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("alloptions", map[string]string{"blogname": "Cached Site"}, "options")
	opts := wp.NewOptions(nil, cache)
	got := opts.GetOption("blogname", "missing")
	if got != "Cached Site" {
		t.Fatalf("got %v", got)
	}
}

func TestGetOptionNotoptions(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("notoptions", map[string]bool{"nope": true}, "options")
	opts := wp.NewOptions(nil, cache)
	got := opts.GetOption("nope", "default")
	if got != "default" {
		t.Fatalf("got %v", got)
	}
}

func TestGetOptionIndividualCache(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("alloptions", map[string]string{}, "options")
	cache.Set("my_opt", "from-cache", "options")
	opts := wp.NewOptions(nil, cache)
	got := opts.GetOption("my_opt", nil)
	if got != "from-cache" {
		t.Fatalf("got %v", got)
	}
}

func TestContextOptionsField(t *testing.T) {
	c := wp.NewContext(context.Background())
	opts := wp.NewOptions(nil, wp.NewObjectCache())
	c.Options = opts
	if c.Options != opts {
		t.Fatal("Options not wired")
	}
}

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	var (
		handle *db.DB
		err    error
	)
	if dsn := os.Getenv("GOPRESS_TEST_DSN"); dsn != "" {
		handle, err = db.OpenDSN(dsn, "wp_", "utf8mb4")
	} else {
		cfg := &config.Config{
			DBName:      "gopress_ref",
			DBUser:      "gopress",
			DBPassword:  "gopress",
			DBHost:      "127.0.0.1:3306",
			DBCharset:   "utf8mb4",
			TablePrefix: "wp_",
		}
		handle, err = db.Open(cfg)
	}
	if err != nil {
		t.Skipf("DB unavailable: %v", err)
	}
	return handle
}

func TestGetOptionBlognameIntegration(t *testing.T) {
	handle := openTestDB(t)
	defer handle.Close()

	cache := wp.NewObjectCache()
	opts := wp.NewOptions(handle, cache)
	if err := opts.LoadAutoload(); err != nil {
		t.Fatal(err)
	}
	got := opts.GetOption("blogname", "")
	s, _ := got.(string)
	if s == "" {
		t.Fatalf("blogname empty; got %#v", got)
	}
	t.Logf("blogname=%q", s)

	// Second hit should come from alloptions cache without needing DB.
	opts2 := wp.NewOptions(nil, cache)
	got2 := opts2.GetOption("blogname", "")
	if got2 != got {
		t.Fatalf("cache miss after LoadAutoload: %v vs %v", got2, got)
	}
}

func TestAddUpdateDeleteOptionIntegration(t *testing.T) {
	handle := openTestDB(t)
	defer handle.Close()

	cache := wp.NewObjectCache()
	opts := wp.NewOptions(handle, cache)
	name := "gopress_p17_test_option"

	_, _ = opts.DeleteOption(name)
	ok, err := opts.AddOption(name, "one", "off")
	if err != nil || !ok {
		t.Fatalf("AddOption: ok=%v err=%v", ok, err)
	}
	if got := opts.GetOption(name, nil); got != "one" {
		t.Fatalf("after add: %v", got)
	}
	ok, err = opts.UpdateOption(name, "two", "")
	if err != nil || !ok {
		t.Fatalf("UpdateOption: ok=%v err=%v", ok, err)
	}
	if got := opts.GetOption(name, nil); got != "two" {
		t.Fatalf("after update: %v", got)
	}
	ok, err = opts.DeleteOption(name)
	if err != nil || !ok {
		t.Fatalf("DeleteOption: ok=%v err=%v", ok, err)
	}
	if got := opts.GetOption(name, "gone"); got != "gone" {
		t.Fatalf("after delete: %v", got)
	}
}
