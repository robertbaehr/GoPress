package rest

import (
	"testing"
)

func TestLoadIndexRoutesAnon(t *testing.T) {
	routes, err := loadIndexRoutesAnon()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(routes) < 130 {
		t.Fatalf("got %d routes", len(routes))
	}
	if _, ok := routes["/wp/v2/posts"]; !ok {
		t.Fatal("missing posts")
	}
	t.Logf("routes=%d embed=%d", len(routes), len(indexRoutesAnonGZ))
}
