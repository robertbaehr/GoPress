package wp

import "testing"

func TestMetaTypeCastSQLP4170(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"CHAR", ""},
		{"char", ""},
		{"NUMERIC", "SIGNED"},
		{"numeric", "SIGNED"},
		{"SIGNED", "SIGNED"},
		{"UNSIGNED", "UNSIGNED"},
		{"DECIMAL", "DECIMAL"},
		{"BINARY", "BINARY"},
		{"DATE", "DATE"},
		{"DATETIME", "DATETIME"},
		{"TIME", "TIME"},
		{"FLOAT", ""},
		{"bogus", ""},
	}
	for _, tc := range cases {
		got := metaTypeCastSQL(tc.in)
		if got != tc.want {
			t.Fatalf("metaTypeCastSQL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
