package wp

import "testing"

// P4.171: splitMetaInValues comma / space / empty.
func TestSplitMetaInValuesP4171(t *testing.T) {
	parts := splitMetaInValues("a, b, c")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Fatalf("comma=%v", parts)
	}
	parts = splitMetaInValues("a b  c")
	if len(parts) != 3 || parts[0] != "a" || parts[2] != "c" {
		t.Fatalf("space=%v", parts)
	}
	if got := splitMetaInValues("  "); len(got) != 0 {
		t.Fatalf("empty=%v", got)
	}
	low, high, ok := splitMetaBetweenValue("10,20")
	if !ok || low != "10" || high != "20" {
		t.Fatalf("between via shared split: %q %q %v", low, high, ok)
	}
}
