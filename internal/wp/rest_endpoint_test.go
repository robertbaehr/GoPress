package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsServingRESTRequest(t *testing.T) {
	if wp.IsServingRESTRequest(nil) {
		t.Fatal("nil context")
	}
	c := wp.NewContext(context.TODO())
	if wp.IsServingRESTRequest(c) {
		t.Fatal("fresh context should not be REST")
	}
	c.Scope.SetREST(true)
	if !wp.IsServingRESTRequest(c) {
		t.Fatal("SetREST should make IsServingRESTRequest true")
	}
}

func TestIsRESTEndpoint(t *testing.T) {
	c := wp.NewContext(context.TODO())
	if wp.IsRESTEndpoint(c) {
		t.Fatal("fresh context")
	}
	c.Scope.BeginRESTDispatch()
	if !wp.IsRESTEndpoint(c) {
		t.Fatal("BeginRESTDispatch should make IsRESTEndpoint true")
	}
	if !wp.IsServingRESTRequest(c) {
		t.Fatal("dispatch also sets IsREST")
	}
	c.Scope.EndRESTDispatch()
	// IsREST stays true after End (standalone REST_REQUEST semantics).
	if !wp.IsServingRESTRequest(c) {
		t.Fatal("IsREST should remain after EndRESTDispatch")
	}
	if c.Scope.IsRESTDispatching() {
		t.Fatal("dispatch nesting should be zero")
	}
}

func TestIsRESTEndpointFilter(t *testing.T) {
	c := wp.NewContext(context.TODO())
	c.Hooks = wp.NewHooks()
	c.Hooks.AddFilter("wp_is_rest_endpoint", func(value any, _ ...any) any {
		return true
	}, 10, 1)
	if !wp.IsRESTEndpoint(c) {
		t.Fatal("filter should force true")
	}
}
