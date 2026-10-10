package wp

import (
	"reflect"
	"testing"
)

func TestSortMapKeysNumeric(t *testing.T) {
	keys := []string{"10", "2", "1"}
	sortMapKeys(keys)
	want := []string{"1", "2", "10"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("numeric: %#v want %#v", keys, want)
	}
	keys = []string{"z", "a", "m"}
	sortMapKeys(keys)
	want = []string{"a", "m", "z"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("alpha: %#v want %#v", keys, want)
	}
}
