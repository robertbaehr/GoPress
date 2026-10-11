package wp_test

import (
	"fmt"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.143: StrLen / MBStrLen match PHP strlen / _mb_strlen.
func TestStrLenByteLengthP2143(t *testing.T) {
	if wp.StrLen("") != 0 || wp.WPStrLen("") != 0 {
		t.Fatal("empty")
	}
	if wp.StrLen("abc") != 3 {
		t.Fatal("ascii")
	}
	// Pencil ✏ is 3 UTF-8 bytes.
	if wp.StrLen("✏") != 3 {
		t.Fatalf("utf8 bytes: got %d want 3", wp.StrLen("✏"))
	}
	if wp.WPStrLen("Hello") != 5 {
		t.Fatal("WPStrLen")
	}
}

func TestMBStrLenUTF8P2143(t *testing.T) {
	if wp.MBStrLen("Hello", "UTF-8") != 5 {
		t.Fatal("ascii codepoints")
	}
	if wp.MBStrLen("✏", "UTF-8") != 1 {
		t.Fatalf("pencil codepoints: got %d want 1", wp.MBStrLen("✏", "UTF-8"))
	}
	if wp.MBStrLen("café", "utf8") != 4 {
		t.Fatalf("cafe: got %d", wp.MBStrLen("café", "utf8"))
	}
	// Non-UTF-8 charset → byte length.
	if wp.MBStrLen("✏", "ISO-8859-1") != 3 {
		t.Fatalf("latin1 bytes: got %d want 3", wp.MBStrLen("✏", "ISO-8859-1"))
	}
	invalid := "a\xc0b"
	if wp.MBStrLen(invalid, "UTF-8") != 3 {
		t.Fatalf("invalid maximal subparts: got %d want 3", wp.MBStrLen(invalid, "UTF-8"))
	}
}

func TestMBStrLenCtxBlogCharsetP2143(t *testing.T) {
	c := testBootQuery(t)
	prev := c.Options.GetOption("blog_charset", "UTF-8")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("blog_charset", fmt.Sprint(prev), "yes")
	})
	if _, err := c.Options.UpdateOption("blog_charset", "UTF-8", "yes"); err != nil {
		t.Fatal(err)
	}
	if wp.MBStrLenCtx(c, "✏", "") != 1 {
		t.Fatal("default UTF-8 blog_charset")
	}
	if _, err := c.Options.UpdateOption("blog_charset", "ISO-8859-1", "yes"); err != nil {
		t.Fatal(err)
	}
	if wp.MBStrLenCtx(c, "✏", "") != 3 {
		t.Fatal("latin1 blog_charset → byte length")
	}
	if wp.MBStrLenCtx(c, "✏", "UTF-8") != 1 {
		t.Fatal("explicit UTF-8 overrides blog_charset")
	}
}
