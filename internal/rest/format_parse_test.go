package rest

import (
	"net/http/httptest"
	"testing"
)

func TestParsePostsFormatValid(t *testing.T) {
	req := httptest.NewRequest("GET", "/x?format=status,aside&format[]=standard", nil)
	in, std, errMsg := parsePostsFormat(req)
	if errMsg != "" {
		t.Fatalf("unexpected err %q", errMsg)
	}
	if !std {
		t.Fatal("expected includeStandard")
	}
	if len(in) != 2 {
		t.Fatalf("formatIn=%v", in)
	}
	in, std, errMsg = parsePostsFormat(nil)
	if in != nil || std || errMsg != "" {
		t.Fatal("nil request")
	}
}

// P4.154: invalid enum token returns rest_not_in_enum message with index.
func TestParsePostsFormatInvalidP4154(t *testing.T) {
	req := httptest.NewRequest("GET", "/x?format=bogus", nil)
	in, std, errMsg := parsePostsFormat(req)
	if in != nil || std {
		t.Fatalf("in=%v std=%v", in, std)
	}
	want := "format[0] is not one of standard, aside, chat, gallery, link, image, quote, status, video, and audio."
	if errMsg != want {
		t.Fatalf("errMsg=%q want %q", errMsg, want)
	}

	req = httptest.NewRequest("GET", "/x?format=status,,bogus", nil)
	_, _, errMsg = parsePostsFormat(req)
	want = "format[1] is not one of standard, aside, chat, gallery, link, image, quote, status, video, and audio."
	if errMsg != want {
		t.Fatalf("mixed errMsg=%q want %q", errMsg, want)
	}

	req = httptest.NewRequest("GET", "/x?format=", nil)
	in, std, errMsg = parsePostsFormat(req)
	if errMsg != "" || len(in) != 0 || std {
		t.Fatalf("empty scalar: in=%v std=%v err=%q", in, std, errMsg)
	}

	req = httptest.NewRequest("GET", "/x?format[]=", nil)
	_, _, errMsg = parsePostsFormat(req)
	if errMsg == "" {
		t.Fatal("expected empty array element error")
	}
}
