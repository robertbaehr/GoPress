package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetMetadataFromCache(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("1", map[string][]string{
		"_edit_lock": {"123:1"},
		"custom":     {"a", "b"},
	}, "post_meta")
	m := wp.NewMeta(nil, cache)

	got := m.GetMetadata(wp.MetaPost, 1, "_edit_lock", true)
	if got != "123:1" {
		t.Fatalf("single: got %#v", got)
	}
	multi := m.GetMetadata(wp.MetaPost, 1, "custom", false)
	vals, ok := multi.([]string)
	if !ok || len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Fatalf("multi: got %#v", multi)
	}
	all := m.GetMetadata(wp.MetaPost, 1, "", false)
	allMap, ok := all.(map[string][]string)
	if !ok || len(allMap) != 2 {
		t.Fatalf("all: got %#v", all)
	}
}

func TestGetMetadataMissingDefaults(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("9", map[string][]string{}, "post_meta")
	m := wp.NewMeta(nil, cache)
	if got := m.GetMetadata(wp.MetaPost, 9, "missing", true); got != "" {
		t.Fatalf("single default: %#v", got)
	}
	got := m.GetMetadata(wp.MetaPost, 9, "missing", false)
	vals, ok := got.([]string)
	if !ok || len(vals) != 0 {
		t.Fatalf("multi default: %#v", got)
	}
}

func TestGetMetadataInvalidArgs(t *testing.T) {
	m := wp.NewMeta(nil, wp.NewObjectCache())
	if got := m.GetMetadata(wp.MetaPost, 0, "k", true); got != "" {
		t.Fatalf("zero id: %#v", got)
	}
	if got := m.GetMetadata(wp.MetaType("nope"), 1, "k", true); got != "" {
		t.Fatalf("bad type: %#v", got)
	}
}

func TestContextMetaField(t *testing.T) {
	c := wp.NewContext(context.Background())
	meta := wp.NewMeta(nil, wp.NewObjectCache())
	c.Meta = meta
	if c.Meta != meta {
		t.Fatal("Meta not wired")
	}
}

func TestMetaPrepareQueries(t *testing.T) {
	// Exercise db.Prepare shapes used by Meta without a live connection.
	cases := []struct {
		q    string
		args []any
	}{
		{"SELECT COUNT(*) FROM wp_postmeta WHERE meta_key = %s AND post_id = %d", []any{"k", int64(1)}},
		{"INSERT INTO wp_postmeta (post_id, meta_key, meta_value) VALUES (%d, %s, %s)", []any{int64(1), "k", "v"}},
		{"UPDATE wp_postmeta SET meta_value = %s WHERE meta_key = %s AND post_id = %d", []any{"v", "k", int64(1)}},
		{"DELETE FROM wp_postmeta WHERE meta_key = %s AND post_id = %d", []any{"k", int64(1)}},
		{"SELECT umeta_id FROM wp_usermeta WHERE meta_key = %s AND user_id = %d", []any{"k", int64(1)}},
	}
	for _, tc := range cases {
		q, args, err := db.Prepare(tc.q, tc.args...)
		if err != nil {
			t.Fatalf("Prepare %q: %v", tc.q, err)
		}
		if q == "" || len(args) != len(tc.args) {
			t.Fatalf("unexpected Prepare result: %q %#v", q, args)
		}
	}
}

func TestAddUpdateDeleteMetadataIntegration(t *testing.T) {
	handle := openTestDB(t)
	defer handle.Close()

	cache := wp.NewObjectCache()
	m := wp.NewMeta(handle, cache)

	// Use post ID 1 if present; otherwise skip.
	q, args, err := handle.Prepare("SELECT ID FROM " + handle.Posts() + " ORDER BY ID ASC LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	v, err := handle.GetVar(q, args...)
	if err != nil || v == nil {
		t.Skip("no posts in DB")
	}
	postID := asTestInt64(v)
	if postID <= 0 {
		t.Fatalf("could not parse post ID from %#v (%T)", v, v)
	}
	key := "gopress_p18_test_meta"

	_, _ = m.DeleteMetadata(wp.MetaPost, postID, key, "")

	mid, ok, err := m.AddMetadata(wp.MetaPost, postID, key, "one", true)
	if err != nil || !ok || mid == 0 {
		t.Fatalf("AddMetadata: mid=%d ok=%v err=%v postID=%d m.DB=%v", mid, ok, err, postID, m.DB != nil)
	}
	if got := m.GetMetadata(wp.MetaPost, postID, key, true); got != "one" {
		t.Fatalf("after add: %#v", got)
	}

	// Unique add should fail.
	_, ok, err = m.AddMetadata(wp.MetaPost, postID, key, "dup", true)
	if err != nil || ok {
		t.Fatalf("unique AddMetadata should fail: ok=%v err=%v", ok, err)
	}

	_, added, updated, err := m.UpdateMetadata(wp.MetaPost, postID, key, "two", "")
	if err != nil || added || !updated {
		t.Fatalf("UpdateMetadata: added=%v updated=%v err=%v", added, updated, err)
	}
	if got := m.GetMetadata(wp.MetaPost, postID, key, true); got != "two" {
		t.Fatalf("after update: %#v", got)
	}

	ok, err = m.DeleteMetadata(wp.MetaPost, postID, key, "")
	if err != nil || !ok {
		t.Fatalf("DeleteMetadata: ok=%v err=%v", ok, err)
	}
	if got := m.GetMetadata(wp.MetaPost, postID, key, true); got != "" {
		t.Fatalf("after delete: %#v", got)
	}
}

func asTestInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case uint64:
		return int64(n)
	case uint32:
		return int64(n)
	case float64:
		return int64(n)
	case []byte:
		var x int64
		for _, b := range n {
			if b < '0' || b > '9' {
				break
			}
			x = x*10 + int64(b-'0')
		}
		return x
	case string:
		var x int64
		for _, b := range n {
			if b < '0' || b > '9' {
				break
			}
			x = x*10 + int64(b-'0')
		}
		return x
	default:
		return 0
	}
}
