package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDoingAjaxDefaultFalse(t *testing.T) {
	if wp.DoingAjax(nil) {
		t.Fatal("nil context should be false")
	}
	c := wp.NewContext(context.TODO())
	if wp.DoingAjax(c) {
		t.Fatal("expected DoingAjax false by default")
	}
}

func TestDoingAjaxScopeFlag(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Scope.SetAjax(true)
	if !wp.DoingAjax(c) {
		t.Fatal("expected DoingAjax true when Scope.IsAjax set")
	}
}

func TestDoingAjaxFilter(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Hooks = wp.NewHooks()
	c.Hooks.AddFilter("wp_doing_ajax", func(value any, _ ...any) any {
		return true
	}, 10, 1)
	if !wp.DoingAjax(c) {
		t.Fatal("expected filter to force DoingAjax true")
	}
}
