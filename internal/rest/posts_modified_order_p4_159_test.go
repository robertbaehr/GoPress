package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.159: orderby[]= / order[]= → rest_invalid_param / rest_invalid_type (string schema).
func TestPostsOrderbyOrderArrayFormInvalidP4159(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, tc := range []struct {
		path string
		key  string
	}{
		{"/wp-json/wp/v2/posts?orderby[]=modified&per_page=1", "orderby"},
		{"/wp-json/wp/v2/posts?order[]=asc&per_page=1", "order"},
		{"/wp-json/wp/v2/pages?orderby[]=modified&order[]=desc&per_page=1", "orderby"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "rest_invalid_type") {
			t.Fatalf("%s: expected rest_invalid_type: %s", tc.path, body)
		}
		if !strings.Contains(body, tc.key+" is not of type string") {
			t.Fatalf("%s: expected string type error for %s: %s", tc.path, tc.key, body)
		}
	}
}

// P4.159: orderby=modified&order=asc|desc sorts by post_modified then ID.
func TestPostsOrderbyModifiedAscDescP4159(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Stamp three fixture posts with distinct modified times.
	stamps := []struct {
		id int64
		ts string
	}{
		{1, "2024-01-01 10:00:00"},
		{13, "2024-06-01 10:00:00"},
		{19, "2024-12-01 10:00:00"},
	}
	for _, s := range stamps {
		uq, uargs, err := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
			s.ts, s.ts, s.id,
		)
		if err != nil {
			t.Fatalf("prepare bump %d: %v", s.id, err)
		}
		if _, err := c.DB.Exec(uq, uargs...); err != nil {
			t.Fatalf("bump %d: %v", s.id, err)
		}
		id := s.id
		t.Cleanup(func() {
			uq, uargs, _ := c.DB.Prepare(
				"UPDATE "+c.DB.Posts()+" SET post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
				"2024-06-15 12:00:00", "2024-06-15 12:00:00", id,
			)
			_, _ = c.DB.Exec(uq, uargs...)
		})
	}

	include := "1,13,19"
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=modified&order=asc&include="+include+"&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("asc status=%d body=%s", rec.Code, rec.Body.String())
	}
	var asc []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &asc); err != nil {
		t.Fatal(err)
	}
	if len(asc) != 3 {
		t.Fatalf("asc len=%d want 3 body=%s", len(asc), rec.Body.String())
	}
	for i := 1; i < len(asc); i++ {
		if asc[i-1].Modified > asc[i].Modified {
			t.Fatalf("asc order broken: %q then %q", asc[i-1].Modified, asc[i].Modified)
		}
	}
	if asc[0].ID != 1 || asc[1].ID != 13 || asc[2].ID != 19 {
		t.Fatalf("asc ids=%v want 1,13,19", []int64{asc[0].ID, asc[1].ID, asc[2].ID})
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=modified&order=desc&include="+include+"&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("desc status=%d body=%s", rec.Code, rec.Body.String())
	}
	var desc []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &desc); err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%d,%d,%d", desc[0].ID, desc[1].ID, desc[2].ID)
	if got != "19,13,1" {
		t.Fatalf("desc ids=%s want 19,13,1", got)
	}
}
