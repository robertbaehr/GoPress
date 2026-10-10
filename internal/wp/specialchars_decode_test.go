package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSpecialcharsDecodeNoQuotes(t *testing.T) {
	in := `Tom &amp; Jerry &lt;3 &gt; 2 &#038; friends`
	got := wp.SpecialcharsDecode(in, wp.ENTNoQuotes)
	want := `Tom & Jerry <3 > 2 & friends`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// Quotes left encoded under ENT_NOQUOTES.
	q := wp.SpecialcharsDecode(`say &quot;hi&quot;`, wp.ENTNoQuotes)
	if q != `say &quot;hi&quot;` {
		t.Fatalf("quotes changed: %q", q)
	}
}

func TestSpecialcharsDecodeQuotes(t *testing.T) {
	in := `say &quot;hi&quot; &#039;there&#039;`
	got := wp.SpecialcharsDecode(in, wp.ENTQuotes)
	want := `say "hi" 'there'`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSpecialcharsDecodeSingle(t *testing.T) {
	got := wp.SpecialcharsDecode(`a&#039;b &quot;c&quot;`, "single")
	if got != `a'b &quot;c&quot;` {
		t.Fatalf("got %q", got)
	}
}

func TestSpecialcharsDecodeZeroPad(t *testing.T) {
	got := wp.SpecialcharsDecode(`&#0038; &#x026;`, wp.ENTNoQuotes)
	if got != `& &` {
		t.Fatalf("got %q", got)
	}
}

func TestSpecialcharsDecodePassthrough(t *testing.T) {
	if got := wp.SpecialcharsDecode("plain", wp.ENTQuotes); got != "plain" {
		t.Fatalf("got %q", got)
	}
	if got := wp.SpecialcharsDecode("", wp.ENTQuotes); got != "" {
		t.Fatalf("empty=%q", got)
	}
}
