package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestSerializeBlockVoidAndWrapped(t *testing.T) {
	voidName := "core/spacer"
	void := blocks.Block{BlockName: &voidName, Attrs: map[string]any{"height": "100px"}}
	got := blocks.SerializeBlock(void)
	if !strings.Contains(got, "<!-- wp:core/spacer") || !strings.HasSuffix(got, " /-->") {
		t.Fatalf("void serialize: %q", got)
	}

	p := "core/paragraph"
	para := blocks.Block{
		BlockName:    &p,
		Attrs:        map[string]any{},
		InnerHTML:    "<p>Hi</p>",
		InnerContent: []any{"<p>Hi</p>"},
	}
	got = blocks.SerializeBlock(para)
	if got != "<!-- wp:core/paragraph --><p>Hi</p><!-- /wp:core/paragraph -->" {
		t.Fatalf("paragraph serialize: %q", got)
	}
}

func TestApplyBlockHooksBeforeAfter(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/paragraph", "core/post-content", "before")
	blocks.RegisterHookedBlock("core/spacer", "core/post-content", "after")

	in := `<!-- wp:core/post-content --><div class="entry">x</div><!-- /wp:core/post-content -->`
	out := blocks.ApplyBlockHooksToContent(in)
	if !strings.HasPrefix(out, "<!-- wp:core/paragraph") {
		t.Fatalf("expected hooked paragraph before, got %q", out)
	}
	if !strings.Contains(out, "<!-- wp:core/post-content") {
		t.Fatalf("expected anchor preserved, got %q", out)
	}
	if !strings.Contains(out, "<!-- wp:core/spacer") {
		t.Fatalf("expected hooked spacer after, got %q", out)
	}
}

func TestApplyBlockHooksFirstLastChild(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/loginout", "core/navigation", "first_child")
	blocks.RegisterHookedBlock("core/spacer", "core/navigation", "last_child")

	in := `<!-- wp:core/navigation --><!-- wp:core/navigation-link {"label":"Home"} /--><!-- /wp:core/navigation -->`
	out := blocks.ApplyBlockHooksToContent(in)
	if !strings.Contains(out, "<!-- wp:core/loginout") {
		t.Fatalf("expected first_child loginout, got %q", out)
	}
	if !strings.Contains(out, "<!-- wp:core/spacer") {
		t.Fatalf("expected last_child spacer, got %q", out)
	}
	// loginout should appear before navigation-link
	li := strings.Index(out, "<!-- wp:core/loginout")
	nav := strings.Index(out, "<!-- wp:core/navigation-link")
	sp := strings.Index(out, "<!-- wp:core/spacer")
	if li < 0 || nav < 0 || sp < 0 || !(li < nav && nav < sp) {
		t.Fatalf("order wrong: loginout=%d navlink=%d spacer=%d in %q", li, nav, sp, out)
	}
}

func TestInsertHookedBlocksRespectsIgnored(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/paragraph", "core/group", "before")
	hooked := blocks.GetHookedBlocks()
	g := "core/group"
	anchor := blocks.Block{
		BlockName: &g,
		Attrs: map[string]any{
			"metadata": map[string]any{
				"ignoredHookedBlocks": []any{"core/paragraph"},
			},
		},
	}
	if got := blocks.InsertHookedBlocks(anchor, "before", hooked); got != "" {
		t.Fatalf("expected ignored hooked block suppressed, got %q", got)
	}
}

func TestGetHookedBlocksGrouping(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("a/one", "core/group", "before")
	blocks.RegisterHookedBlock("a/two", "core/group", "after")
	got := blocks.GetHookedBlocks()
	if len(got["core/group"]["before"]) != 1 || got["core/group"]["before"][0] != "a/one" {
		t.Fatalf("before: %#v", got)
	}
	if len(got["core/group"]["after"]) != 1 || got["core/group"]["after"][0] != "a/two" {
		t.Fatalf("after: %#v", got)
	}
}

func TestApplyBlockHooksNestedAnchors(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/spacer", "core/group", "before")
	in := `<!-- wp:core/columns --><!-- wp:core/group --><div class="g">x</div><!-- /wp:core/group --><!-- /wp:core/columns -->`
	out := blocks.ApplyBlockHooksToContent(in)
	if !strings.Contains(out, "<!-- wp:core/spacer") {
		t.Fatalf("expected nested hooked spacer, got %q", out)
	}
	sp := strings.Index(out, "<!-- wp:core/spacer")
	grp := strings.Index(out, "<!-- wp:core/group")
	if sp < 0 || grp < 0 || sp > grp {
		t.Fatalf("spacer should precede nested group: spacer=%d group=%d in %q", sp, grp, out)
	}
}

func TestSetIgnoredHookedBlocksMetadata(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/paragraph", "core/group", "before")
	hooked := blocks.GetHookedBlocks()
	g := "core/group"
	anchor := blocks.Block{BlockName: &g, Attrs: map[string]any{}}
	_ = blocks.SetIgnoredHookedBlocksMetadata(&anchor, "before", hooked)
	meta, _ := anchor.Attrs["metadata"].(map[string]any)
	raw, _ := meta["ignoredHookedBlocks"].([]any)
	if len(raw) != 1 || raw[0] != "core/paragraph" {
		t.Fatalf("ignored metadata=%#v attrs=%#v", raw, anchor.Attrs)
	}
	if got := blocks.InsertHookedBlocks(anchor, "before", hooked); got != "" {
		t.Fatalf("after set_ignored, insert should be empty, got %q", got)
	}
}

func TestInsertHookedBlocksAndSetIgnored(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/loginout", "core/navigation", "first_child")
	hooked := blocks.GetHookedBlocks()
	n := "core/navigation"
	anchor := blocks.Block{BlockName: &n, Attrs: map[string]any{}}
	markup := blocks.InsertHookedBlocksAndSetIgnored(&anchor, "first_child", hooked)
	if !strings.Contains(markup, "<!-- wp:core/loginout") {
		t.Fatalf("expected loginout markup, got %q", markup)
	}
	meta, _ := anchor.Attrs["metadata"].(map[string]any)
	raw, _ := meta["ignoredHookedBlocks"].([]any)
	if len(raw) != 1 || raw[0] != "core/loginout" {
		t.Fatalf("ignored=%#v", raw)
	}
}

func TestApplyBlockHooksWithIgnoredCallback(t *testing.T) {
	blocks.ResetHookedBlocks()
	defer blocks.ResetHookedBlocks()

	blocks.RegisterHookedBlock("core/paragraph", "core/post-content", "before")
	in := `<!-- wp:core/post-content --><div>x</div><!-- /wp:core/post-content -->`
	out := blocks.ApplyBlockHooksToContentWith(in, blocks.InsertHookedBlocksAndSetIgnored)
	if !strings.HasPrefix(out, "<!-- wp:core/paragraph") {
		t.Fatalf("expected hooked paragraph, got %q", out)
	}
	if !strings.Contains(out, `"ignoredHookedBlocks"`) && !strings.Contains(out, "ignoredHookedBlocks") {
		t.Fatalf("expected ignoredHookedBlocks in serialized attrs, got %q", out)
	}
}
