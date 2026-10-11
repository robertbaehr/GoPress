package server

import "testing"

func TestWPRobotsSensitivePageHelperP3142(t *testing.T) {
	out := wpRobotsSensitivePage(nil)
	if out["noindex"] != true || out["noarchive"] != true {
		t.Fatalf("got %#v", out)
	}
}
