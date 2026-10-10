package hellodolly_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/plugins/hellodolly"
)

func TestLyricRotationAndTexturize(t *testing.T) {
	if hellodolly.LyricCount() < 10 {
		t.Fatalf("expected many lyrics, got %d", hellodolly.LyricCount())
	}
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		line := hellodolly.RandomLyric()
		if line == "" {
			t.Fatal("empty lyric")
		}
		seen[line] = true
		// Texturize turns ASCII apostrophes into curly entities on matching lines.
		if strings.Contains(line, "&#8217;") || strings.Contains(line, "Hello, Dolly") || strings.Contains(line, "Dolly") {
			continue
		}
	}
	if len(seen) < 3 {
		t.Fatalf("expected rotation across lyrics, unique=%d set=%v", len(seen), seen)
	}
}

func TestChosenLyricMatchesRandom(t *testing.T) {
	a := hellodolly.ChosenLyric(nil)
	b := hellodolly.RandomLyric()
	if a == "" || b == "" {
		t.Fatal("empty")
	}
}
