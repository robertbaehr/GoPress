package depscheck

import "testing"

func TestLicenseForbidden(t *testing.T) {
	needles := []string{"GPL", "AGPL", "SSPL", "BUSL", "Commons Clause"}
	cases := []struct {
		lic  string
		want bool
	}{
		{"MIT", false},
		{"BSD-3-Clause", false},
		{"Apache-2.0", false},
		{"MIT AND Apache-2.0", false},
		{"GPL-2.0", true},
		{"AGPL-3.0", true},
		{"BUSL-1.1", true},
		{"SSPL-1.0", true},
		{"MIT WITH Commons Clause", true},
	}
	for _, tc := range cases {
		if got := licenseForbidden(tc.lic, needles); got != tc.want {
			t.Errorf("licenseForbidden(%q)=%v want %v", tc.lic, got, tc.want)
		}
	}
}
