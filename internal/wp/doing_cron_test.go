package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDoingCronDefaultFalse(t *testing.T) {
	if wp.DoingCron(nil) {
		t.Fatal("nil context should be false")
	}
	c := wp.NewContext(context.TODO())
	if wp.DoingCron(c) {
		t.Fatal("expected DoingCron false by default")
	}
}

func TestDoingCronScopeFlag(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Scope.SetCron(true)
	if !wp.DoingCron(c) {
		t.Fatal("expected DoingCron true when Scope.IsCron set")
	}
}

func TestDoingCronFilter(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Hooks = wp.NewHooks()
	c.Hooks.AddFilter("wp_doing_cron", func(value any, _ ...any) any {
		return true
	}, 10, 1)
	if !wp.DoingCron(c) {
		t.Fatal("expected filter to force DoingCron true")
	}
}
