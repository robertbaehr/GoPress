package wp

import "testing"

func TestMetaKeyCompareSQLP4172(t *testing.T) {
	sql, val, neg := metaKeyCompareSQL("", "footnotes", "")
	if sql != "mq.meta_key = %s" || val != "footnotes" || neg {
		t.Fatalf("default=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("LIKE", "note", "")
	if sql != "mq.meta_key LIKE %s" || val != "%note%" || neg {
		t.Fatalf("like=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("like", "x", "")
	if sql != "mq.meta_key LIKE %s" || val != "%x%" || neg {
		t.Fatalf("lower like=%q %q neg=%v", sql, val, neg)
	}
}
