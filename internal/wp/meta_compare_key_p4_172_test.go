package wp

import "testing"

func TestMetaKeyCompareSQLP4172(t *testing.T) {
	sql, val := metaKeyCompareSQL("", "footnotes")
	if sql != "mq.meta_key = %s" || val != "footnotes" {
		t.Fatalf("default=%q %q", sql, val)
	}
	sql, val = metaKeyCompareSQL("LIKE", "note")
	if sql != "mq.meta_key LIKE %s" || val != "%note%" {
		t.Fatalf("like=%q %q", sql, val)
	}
	sql, val = metaKeyCompareSQL("like", "x")
	if sql != "mq.meta_key LIKE %s" || val != "%x%" {
		t.Fatalf("lower like=%q %q", sql, val)
	}
}
