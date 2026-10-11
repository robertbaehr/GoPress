package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestBlockTypePageListItemP5134(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types/core/page-list-item", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var bt rest.BlockType
	if err := json.Unmarshal(rec.Body.Bytes(), &bt); err != nil {
		t.Fatal(err)
	}
	if bt.Name != "core/page-list-item" || bt.Title != "Page List Item" {
		t.Fatalf("unexpected block type: %#v", bt)
	}
	if len(bt.Parent) != 1 || bt.Parent[0] != "core/page-list" {
		t.Fatalf("parent=%v want [core/page-list]", bt.Parent)
	}
	for _, key := range []string{"id", "label", "title", "link", "hasChildren"} {
		if _, ok := bt.Attributes[key]; !ok {
			t.Fatalf("missing attribute %q in %#v", key, bt.Attributes)
		}
	}
}
