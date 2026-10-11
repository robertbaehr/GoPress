package wp

import "testing"

func TestMetaKeyCompareSQLBinaryP4174(t *testing.T) {
	sql, val, neg := metaKeyCompareSQL("=", "Footnotes", "BINARY")
	if sql != "CAST(mq.meta_key AS BINARY) = %s" || val != "Footnotes" || neg {
		t.Fatalf("binary eq=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("LIKE", "Note", "binary")
	if sql != "CAST(mq.meta_key AS BINARY) LIKE %s" || val != "%Note%" || neg {
		t.Fatalf("binary like=%q %q neg=%v", sql, val, neg)
	}
	sql, val, neg = metaKeyCompareSQL("NOT LIKE", "x", "BINARY")
	if sql != "CAST(mq.meta_key AS BINARY) LIKE %s" || val != "%x%" || !neg {
		t.Fatalf("binary not like=%q %q neg=%v", sql, val, neg)
	}
}
