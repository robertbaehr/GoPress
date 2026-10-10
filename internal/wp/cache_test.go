package wp_test

import (
	"context"
	"sync"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestObjectCacheGetSetDelete(t *testing.T) {
	c := wp.NewObjectCache()
	if _, ok := c.Get("k", "g"); ok {
		t.Fatal("expected miss")
	}
	if !c.Set("k", "v", "g") {
		t.Fatal("Set failed")
	}
	got, ok := c.Get("k", "g")
	if !ok || got != "v" {
		t.Fatalf("Get=%v ok=%v", got, ok)
	}
	// Groups are isolated.
	if _, ok := c.Get("k", "other"); ok {
		t.Fatal("leak across groups")
	}
	if !c.Delete("k", "g") {
		t.Fatal("Delete failed")
	}
	if c.Delete("k", "g") {
		t.Fatal("second Delete should fail")
	}
}

func TestObjectCacheDefaultGroup(t *testing.T) {
	c := wp.NewObjectCache()
	c.Set("a", 1, "")
	got, ok := c.Get("a", "default")
	if !ok || got != 1 {
		t.Fatalf("default group: %v %v", got, ok)
	}
}

func TestObjectCacheAddReplace(t *testing.T) {
	c := wp.NewObjectCache()
	if !c.Add("k", "one", "g") {
		t.Fatal("Add should succeed")
	}
	if c.Add("k", "two", "g") {
		t.Fatal("Add should fail when present")
	}
	v, _ := c.Get("k", "g")
	if v != "one" {
		t.Fatalf("value=%v", v)
	}
	if !c.Replace("k", "two", "g") {
		t.Fatal("Replace should succeed")
	}
	v, _ = c.Get("k", "g")
	if v != "two" {
		t.Fatalf("value=%v", v)
	}
	if c.Replace("missing", "x", "g") {
		t.Fatal("Replace missing should fail")
	}
}

func TestObjectCacheIncrDecr(t *testing.T) {
	c := wp.NewObjectCache()
	if _, ok := c.Incr("n", 1, "g"); ok {
		t.Fatal("Incr missing should fail")
	}
	c.Set("n", 5, "g")
	n, ok := c.Incr("n", 3, "g")
	if !ok || n != 8 {
		t.Fatalf("Incr=%d ok=%v", n, ok)
	}
	n, ok = c.Decr("n", 10, "g")
	if !ok || n != 0 {
		t.Fatalf("Decr clamp=%d ok=%v", n, ok)
	}
	c.Set("s", "not-a-number", "g")
	n, ok = c.Incr("s", 2, "g")
	if !ok || n != 2 {
		t.Fatalf("non-numeric Incr=%d ok=%v", n, ok)
	}
}

func TestObjectCacheFlush(t *testing.T) {
	c := wp.NewObjectCache()
	c.Set("a", 1, "g1")
	c.Set("b", 2, "g2")
	c.FlushGroup("g1")
	if _, ok := c.Get("a", "g1"); ok {
		t.Fatal("FlushGroup failed")
	}
	if _, ok := c.Get("b", "g2"); !ok {
		t.Fatal("other group cleared")
	}
	c.Flush()
	if _, ok := c.Get("b", "g2"); ok {
		t.Fatal("Flush failed")
	}
}

func TestObjectCacheConcurrent(t *testing.T) {
	c := wp.NewObjectCache()
	c.Set("n", int64(0), "g")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Incr("n", 1, "g")
		}()
	}
	wg.Wait()
	v, ok := c.Get("n", "g")
	if !ok || v.(int64) != 100 {
		t.Fatalf("got %v ok=%v", v, ok)
	}
}

func TestContextCacheField(t *testing.T) {
	ctx := wp.NewContext(context.Background())
	cache := wp.NewObjectCache()
	ctx.Cache = cache
	if ctx.Cache != cache {
		t.Fatal("Cache not wired")
	}
}

func TestObjectCacheGlobalAndNonPersistentGroups(t *testing.T) {
	c := wp.NewObjectCache()
	if c.IsGlobalGroup("users") {
		t.Fatal("users should not be global before registration")
	}
	c.AddGlobalGroups("users", "useremail")
	if !c.IsGlobalGroup("users") || !c.IsGlobalGroup("useremail") {
		t.Fatal("expected global groups registered")
	}
	c.AddNonPersistentGroups("counts", "plugins")
	if !c.IsNonPersistentGroup("counts") || !c.IsNonPersistentGroup("plugins") {
		t.Fatal("expected non-persistent groups registered")
	}
	// Groups remain usable for Get/Set.
	c.Set("1", "admin", "users")
	v, ok := c.Get("1", "users")
	if !ok || v != "admin" {
		t.Fatalf("Get users: %v %v", v, ok)
	}
}

func TestBootRegistersDefaultCacheGroups(t *testing.T) {
	c := cronBoot(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	if c.Cache == nil {
		t.Fatal("nil cache")
	}
	if !c.Cache.IsGlobalGroup("users") {
		t.Fatal("expected users global group from boot")
	}
	if !c.Cache.IsNonPersistentGroup("counts") {
		t.Fatal("expected counts non-persistent group from boot")
	}
}
