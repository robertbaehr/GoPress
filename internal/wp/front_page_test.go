package wp_test

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestApplyFrontPageDisplayStaticFront(t *testing.T) {
	c := testBootQuery(t)

	origShow := c.Options.GetOption("show_on_front", "posts")
	origFront := c.Options.GetOption("page_on_front", "0")
	origPosts := c.Options.GetOption("page_for_posts", "0")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("show_on_front", origShow, "yes")
		_, _ = c.Options.UpdateOption("page_on_front", origFront, "yes")
		_, _ = c.Options.UpdateOption("page_for_posts", origPosts, "yes")
	})

	pages, err := wp.ListPublishedPages(c, 10)
	if err != nil || len(pages) == 0 {
		t.Skip("no pages")
	}
	front := pages[0]
	var postsPage wp.NavPage
	if len(pages) > 1 {
		postsPage = pages[1]
	} else {
		postsPage = front
	}
	_, _ = c.Options.UpdateOption("show_on_front", "page", "yes")
	_, _ = c.Options.UpdateOption("page_on_front", front.ID, "yes")
	if postsPage.ID != front.ID {
		_, _ = c.Options.UpdateOption("page_for_posts", postsPage.ID, "yes")
	}

	req := httptest.NewRequest("GET", "/", nil)
	rc := c.ForRequest(httptest.NewRecorder(), req)
	rules, err := rc.LoadRewriteRules()
	if err != nil {
		t.Fatal(err)
	}
	res := rc.ParseRequest(rules)
	if res.IsHome {
		t.Fatal("static front `/` should not be is_home")
	}
	if !rc.Scope.IsFrontPage {
		t.Fatal("expected IsFrontPage")
	}
	if rc.Scope.IsHome {
		t.Fatal("IsHome should be false on static front")
	}
	if rc.Scope.QueriedObjectID != front.ID {
		t.Fatalf("QueriedObjectID=%d want %d", rc.Scope.QueriedObjectID, front.ID)
	}
	if res.QueryVars["page_id"] != strconv.FormatInt(front.ID, 10) {
		t.Fatalf("page_id=%q", res.QueryVars["page_id"])
	}

	if postsPage.ID == front.ID || postsPage.Name == "" {
		return
	}
	req2 := httptest.NewRequest("GET", "/"+postsPage.Name+"/", nil)
	rc2 := c.ForRequest(httptest.NewRecorder(), req2)
	res2 := rc2.ParseRequest(rules)
	if !res2.IsHome || !rc2.Scope.IsPostsPage {
		t.Fatalf("posts page: IsHome=%v IsPostsPage=%v", res2.IsHome, rc2.Scope.IsPostsPage)
	}
	if rc2.Scope.IsFrontPage {
		t.Fatal("posts page must not be front page")
	}
}
