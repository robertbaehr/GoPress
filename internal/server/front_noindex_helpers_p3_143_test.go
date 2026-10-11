package server

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFrontRobotsNeedsNoRobotsHelperP3143(t *testing.T) {
	c := &wp.Context{Req: httptest.NewRequest("GET", "/?p=1&replytocom=1", nil)}
	if !frontRobotsNeedsNoRobots(c) {
		t.Fatal("replytocom")
	}
	c.Req = httptest.NewRequest("GET", "/?unapproved=1&moderation-hash=x", nil)
	if !frontRobotsNeedsNoRobots(c) {
		t.Fatal("unapproved+hash")
	}
	c.Req = httptest.NewRequest("GET", "/?unapproved=1", nil)
	if frontRobotsNeedsNoRobots(c) {
		t.Fatal("unapproved alone")
	}
	c.Req = httptest.NewRequest("GET", "/?p=1", nil)
	if frontRobotsNeedsNoRobots(c) {
		t.Fatal("plain")
	}
}
