package wp_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFromContextRoundTrip(t *testing.T) {
	c := wp.NewContext(context.Background())
	ctx := wp.WithContext(context.Background(), c)
	got := wp.FromContext(ctx)
	if got != c {
		t.Fatalf("FromContext got %p want %p", got, c)
	}
	if wp.FromContext(context.Background()) != nil {
		t.Fatal("expected nil for empty context")
	}
}

func TestRequestScopeQueryOnce(t *testing.T) {
	s := wp.NewRequestScope()
	var n atomic.Int32
	fn := func() (any, error) {
		n.Add(1)
		return "query", nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := s.DoQueryParse(fn)
			if err != nil || v != "query" {
				t.Errorf("DoQueryParse: %v %v", v, err)
			}
		}()
	}
	wg.Wait()
	if n.Load() != 1 {
		t.Fatalf("fn ran %d times; want 1", n.Load())
	}
	if s.Query() != "query" {
		t.Fatalf("Query()=%v", s.Query())
	}
}

func TestRequestScopeCurrentPostOnce(t *testing.T) {
	s := wp.NewRequestScope()
	var n atomic.Int32
	_, _ = s.DoCurrentPost(func() (any, error) {
		n.Add(1)
		return map[string]int{"ID": 1}, nil
	})
	_, _ = s.DoCurrentPost(func() (any, error) {
		n.Add(1)
		return map[string]int{"ID": 2}, nil
	})
	if n.Load() != 1 {
		t.Fatalf("fn ran %d times; want 1", n.Load())
	}
	post, _ := s.CurrentPost().(map[string]int)
	if post["ID"] != 1 {
		t.Fatalf("post=%v", post)
	}
}

func TestNewContextHasScope(t *testing.T) {
	c := wp.NewContext(context.TODO())
	if c.Scope == nil {
		t.Fatal("Scope is nil")
	}
	c.Scope.SetAdmin(true)
	c.Scope.SetREST(true)
	if !c.Scope.IsAdmin || !c.Scope.IsREST {
		t.Fatal("flags not set")
	}
}
