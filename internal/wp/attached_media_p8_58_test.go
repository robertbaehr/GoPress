package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.58: GetAttachedMedia returns parented attachments by mime prefix.
func TestGetAttachedMediaP858(t *testing.T) {
	c := testBootQuery(t)
	// Temporarily parent attachment 5 under post 1.
	q, args, err := c.DB.Prepare("UPDATE "+c.DB.Posts()+" SET post_parent = %d WHERE ID = %d", int64(1), int64(5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("UPDATE "+c.DB.Posts()+" SET post_parent = %d WHERE ID = %d", int64(0), int64(5))
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})
	if c.Cache != nil {
		c.Cache.Flush()
	}

	got := wp.GetAttachedMedia(c, "image", 1)
	if len(got) < 1 {
		t.Fatalf("expected image attachments, got %d", len(got))
	}
	found := false
	for _, p := range got {
		if p.ID == 5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing attachment 5 in %#v", got)
	}

	audio := wp.GetAttachedMedia(c, "audio", 1)
	for _, p := range audio {
		if p.ID == 5 {
			t.Fatal("image should not match audio")
		}
	}

	c.Hooks.AddFilter("get_attached_media", func(value any, _ ...any) any {
		return []wp.Post{{ID: 99, Type: "attachment"}}
	}, 10, 3)
	if got := wp.GetAttachedMedia(c, "image", 1); len(got) != 1 || got[0].ID != 99 {
		t.Fatalf("filter=%#v", got)
	}
}
