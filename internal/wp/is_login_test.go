package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsLoginDefaultFalse(t *testing.T) {
	if wp.IsLogin(nil) {
		t.Fatal("nil context should be false")
	}
	c := wp.NewContext(context.TODO())
	if wp.IsLogin(c) {
		t.Fatal("expected IsLogin false by default")
	}
}

func TestIsLoginScopeFlag(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Scope.SetLogin(true)
	if !wp.IsLogin(c) {
		t.Fatal("expected IsLogin true when Scope.IsLogin set")
	}
}
