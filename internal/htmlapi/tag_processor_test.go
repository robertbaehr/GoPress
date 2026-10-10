package htmlapi_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/htmlapi"
)

func TestNextTagFindsOpeners(t *testing.T) {
	p := htmlapi.NewTagProcessor(`Hello <p class="lead">World</p><br/>`)
	if !p.NextTag(nil) {
		t.Fatal("expected first tag")
	}
	if p.GetTag() != "P" || p.IsTagCloser() {
		t.Fatalf("got tag=%q closer=%v", p.GetTag(), p.IsTagCloser())
	}
	cls, ok := p.GetAttribute("class")
	if !ok || cls != "lead" {
		t.Fatalf("class=%q ok=%v", cls, ok)
	}
	if !p.HasClass("lead") {
		t.Fatal("expected HasClass lead")
	}
	if !p.NextTag(nil) {
		t.Fatal("expected br")
	}
	if p.GetTag() != "BR" {
		t.Fatalf("got %q", p.GetTag())
	}
	if p.NextTag(nil) {
		t.Fatal("no more openers")
	}
}

func TestNextTagByNameAndClosers(t *testing.T) {
	p := htmlapi.NewTagProcessor(`<div><span>x</span></div>`)
	if !p.NextTag(&htmlapi.TagQuery{TagName: "span"}) {
		t.Fatal("expected span opener")
	}
	if p.GetTag() != "SPAN" || p.IsTagCloser() {
		t.Fatalf("got %q closer=%v", p.GetTag(), p.IsTagCloser())
	}
	p2 := htmlapi.NewTagProcessor(`<div><span>x</span></div>`)
	if !p2.NextTag(&htmlapi.TagQuery{TagName: "div", TagClosers: true}) {
		t.Fatal("expected div")
	}
	// First match is opener; seek closer.
	if !p2.NextTag(&htmlapi.TagQuery{TagName: "div", TagClosers: true}) {
		t.Fatal("expected div closer")
	}
	if !p2.IsTagCloser() || p2.GetTag() != "DIV" {
		t.Fatalf("got closer=%v tag=%q", p2.IsTagCloser(), p2.GetTag())
	}
}

func TestNextTagSkipsCommentsAndBooleanAttrs(t *testing.T) {
	p := htmlapi.NewTagProcessor(`<!-- hi --><input disabled type="checkbox">`)
	if !p.NextTag(&htmlapi.TagQuery{TagName: "input"}) {
		t.Fatal("expected input")
	}
	if _, ok := p.GetAttribute("disabled"); !ok {
		t.Fatal("expected boolean disabled")
	}
	typ, ok := p.GetAttribute("type")
	if !ok || typ != "checkbox" {
		t.Fatalf("type=%q ok=%v", typ, ok)
	}
}

func TestNextTagClassQuery(t *testing.T) {
	p := htmlapi.NewTagProcessor(`<p class="a">1</p><p class="b fullwidth">2</p>`)
	if !p.NextTag(&htmlapi.TagQuery{ClassName: "fullwidth"}) {
		t.Fatal("expected class match")
	}
	if p.GetTag() != "P" {
		t.Fatalf("got %q", p.GetTag())
	}
}
