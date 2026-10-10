package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMetadataLazyloaderQueueAndFlush(t *testing.T) {
	cache := wp.NewObjectCache()
	m := wp.NewMeta(nil, cache)
	if m.Lazy == nil {
		t.Fatal("expected Lazyloader on NewMeta")
	}

	m.Lazy.QueueObjects(wp.MetaComment, []int64{10, 11, 0, 11})
	if m.Lazy.PendingCount(wp.MetaComment) != 2 {
		t.Fatalf("pending=%d", m.Lazy.PendingCount(wp.MetaComment))
	}

	m.Lazy.Flush(wp.MetaComment, 12)
	if m.Lazy.PendingCount(wp.MetaComment) != 0 {
		t.Fatal("queue should be empty after flush")
	}

	m.Lazy.QueueObjects(wp.MetaTerm, []int64{1, 2})
	m.Lazy.ResetQueue(wp.MetaTerm)
	if m.Lazy.PendingCount(wp.MetaTerm) != 0 {
		t.Fatal("ResetQueue should clear")
	}
}

func TestGetMetadataFlushesLazyQueue(t *testing.T) {
	cache := wp.NewObjectCache()
	cache.Set("5", map[string][]string{"k": {"v"}}, "comment_meta")
	m := wp.NewMeta(nil, cache)
	m.Lazy.QueueObjects(wp.MetaComment, []int64{6, 7})
	if m.Lazy.PendingCount(wp.MetaComment) != 2 {
		t.Fatalf("pending=%d", m.Lazy.PendingCount(wp.MetaComment))
	}
	got := m.GetMetadata(wp.MetaComment, 5, "k", true)
	if got != "v" {
		t.Fatalf("got %#v", got)
	}
	if m.Lazy.PendingCount(wp.MetaComment) != 0 {
		t.Fatal("GetMetadata should flush lazy queue")
	}
}

func TestUpdateMetaCacheBatchHelper(t *testing.T) {
	cache := wp.NewObjectCache()
	m := wp.NewMeta(nil, cache)
	out, err := m.UpdateMetaCache(wp.MetaPost, []int64{100, 101})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 cache entries, got %d", len(out))
	}
	if out[100] == nil || out[101] == nil {
		t.Fatalf("missing empty buckets: %#v", out)
	}
}
