package diff

import (
	"net/http"
	"testing"
)

func TestNormalizeJSONSemantic(t *testing.T) {
	a := Response{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: []byte(`{"b":1,"a":2}`)}
	b := Response{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: []byte(`{"a":2,"b":1}`)}
	na, err := Normalize(a, CompareJSON)
	if err != nil {
		t.Fatal(err)
	}
	nb, err := Normalize(b, CompareJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(na, nb) {
		t.Fatalf("expected semantic JSON match\nA:\n%s\nB:\n%s", na.Body, nb.Body)
	}
}

func TestNormalizeNonceAndCookie(t *testing.T) {
	a := Response{
		Status: 200,
		Headers: http.Header{
			"Set-Cookie": []string{"wordpress_logged_in_abc=session1; path=/; HttpOnly"},
			"Date":       []string{"Fri, 09 Oct 2026 00:00:00 GMT"},
		},
		Body: []byte(`{"nonce":"abcdef1234","x":1}`),
	}
	b := Response{
		Status: 200,
		Headers: http.Header{
			"Set-Cookie": []string{"wordpress_logged_in_abc=session2; path=/; HttpOnly"},
			"Date":       []string{"Fri, 09 Oct 2026 12:00:00 GMT"},
		},
		Body: []byte(`{"nonce":"9999999999","x":1}`),
	}
	na, err := Normalize(a, CompareJSON)
	if err != nil {
		t.Fatal(err)
	}
	nb, err := Normalize(b, CompareJSON)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := na.Headers["Date"]; ok {
		t.Fatal("Date header should be omitted")
	}
	if !Equal(na, nb) {
		t.Fatalf("nonce/cookie scrub failed\nA:%+v\nB:%+v\nA body:%s\nB body:%s", na.Cookies, nb.Cookies, na.Body, nb.Body)
	}
}

func TestNormalizeHTMLAttrOrder(t *testing.T) {
	a := Response{Status: 200, Body: []byte(`<div id="x" class="a"> hi </div>`)}
	b := Response{Status: 200, Body: []byte(`<div class="a" id="x">hi</div>`)}
	na, err := Normalize(a, CompareHTML)
	if err != nil {
		t.Fatal(err)
	}
	nb, err := Normalize(b, CompareHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(na, nb) {
		t.Fatalf("HTML attr/ws mismatch\nA:\n%s\nB:\n%s", na.Body, nb.Body)
	}
}
