package wp

import "testing"

func TestMetaKeyCompareSQLP4173(t *testing.T) {
	sql, val, neg := metaKeyCompareSQL("NOT LIKE", "note", "")
	if sql != "mq.meta_key LIKE %s" || val != "%note%" || !neg {
		t.Fatalf("not like=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("!=", "footnotes", "")
	if sql != "mq.meta_key = %s" || val != "footnotes" || !neg {
		t.Fatalf("neq=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("EXISTS", "x", "")
	if sql != "mq.meta_key = %s" || val != "x" || neg {
		t.Fatalf("exists=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("LIKE", "note", "")
	if sql != "mq.meta_key LIKE %s" || val != "%note%" || neg {
		t.Fatalf("like=%q %q neg=%v", sql, val, neg)
	}
}
