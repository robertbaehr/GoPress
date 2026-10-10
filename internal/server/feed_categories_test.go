package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func testBootFeedCats(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH:        "/workspace/",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestFeedCategoryNamesIncludesTags(t *testing.T) {
	c := testBootFeedCats(t)
	// Fixture Post 01 (id=13) has category "Fixture Leaf" and tag "Fixture Tag 01".
	names := feedCategoryNames(c, 13)
	hasCat, hasTag := false, false
	for _, n := range names {
		if n == "Fixture Leaf" {
			hasCat = true
		}
		if n == "Fixture Tag 01" {
			hasTag = true
		}
	}
	if !hasCat || !hasTag {
		t.Fatalf("names=%v want Fixture Leaf + Fixture Tag 01", names)
	}
}

func TestFeedRSSEmitsTagCategoriesWhenPresent(t *testing.T) {
	c := testBootFeedCats(t)
	// Attach Fixture Tag 01 to Hello world (id=1) for the duration of this test.
	q, args, err := c.DB.Prepare(
		"SELECT term_taxonomy_id FROM "+c.DB.TermTaxonomy()+" tt "+
			"INNER JOIN "+c.DB.Terms()+" t ON t.term_id = tt.term_id "+
			"WHERE t.name = %s AND tt.taxonomy = %s LIMIT 1",
		"Fixture Tag 01", "post_tag",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("lookup tag: %v %#v", err, row)
	}
	ttID, err := parseDBInt64(row["term_taxonomy_id"])
	if err != nil || ttID <= 0 {
		t.Fatalf("bad tt id %#v: %v", row["term_taxonomy_id"], err)
	}
	ins, iArgs, err := c.DB.Prepare(
		"INSERT IGNORE INTO "+c.DB.TermRelationships()+" (object_id, term_taxonomy_id, term_order) VALUES (%d, %d, %d)",
		int64(1), ttID, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(ins, iArgs...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		del, dArgs, _ := c.DB.Prepare(
			"DELETE FROM "+c.DB.TermRelationships()+" WHERE object_id = %d AND term_taxonomy_id = %d",
			int64(1), ttID,
		)
		_, _ = c.DB.Exec(del, dArgs...)
	})

	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<category><![CDATA[Fixture Tag 01]]></category>") {
		t.Fatalf("rss missing tag category: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, "<category><![CDATA[Uncategorized]]></category>") {
		t.Fatalf("rss missing category: %q", body[:min(800, len(body))])
	}
}

func parseDBInt64(v any) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case uint64:
		return int64(t), nil
	case []byte:
		return strconv.ParseInt(string(t), 10, 64)
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported %T", v)
	}
}
