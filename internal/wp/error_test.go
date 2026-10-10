package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNewWPErrorEmptyCode(t *testing.T) {
	e := wp.NewWPError("", "ignored", "data")
	if e.HasErrors() {
		t.Fatal("empty code should not add errors")
	}
	if wp.IsWPError(e) != true {
		t.Fatal("empty WP_Error is still a WP_Error instance")
	}
}

func TestWPErrorAddGetRemove(t *testing.T) {
	e := wp.NewWPError("rest_forbidden", "Nope", map[string]any{"status": 403})
	if !e.HasErrors() {
		t.Fatal("expected errors")
	}
	if e.GetErrorCode() != "rest_forbidden" {
		t.Fatalf("code=%q", e.GetErrorCode())
	}
	if e.GetErrorMessage("") != "Nope" {
		t.Fatalf("message=%q", e.GetErrorMessage(""))
	}
	data, _ := e.GetErrorData("").(map[string]any)
	if data["status"] != 403 {
		t.Fatalf("data=%v", e.GetErrorData(""))
	}
	e.Add("rest_forbidden", "Also no", nil)
	if got := e.GetErrorMessages("rest_forbidden"); len(got) != 2 {
		t.Fatalf("messages=%v", got)
	}
	e.Remove("rest_forbidden")
	if e.HasErrors() {
		t.Fatal("expected empty after remove")
	}
}

func TestIsWPError(t *testing.T) {
	if wp.IsWPError(nil) {
		t.Fatal("nil")
	}
	if wp.IsWPError("string") {
		t.Fatal("string")
	}
	if wp.IsWPError(error(nil)) {
		t.Fatal("typed nil error")
	}
	var err error = wp.NewWPError("x", "y", nil)
	if !wp.IsWPError(err) {
		t.Fatal("error interface holding *WPError")
	}
}

func TestIsWPErrorInstanceAction(t *testing.T) {
	h := wp.NewHooks()
	called := 0
	h.AddAction("is_wp_error_instance", func(args ...any) {
		called++
		if len(args) != 1 {
			t.Fatalf("args=%v", args)
		}
		if !wp.IsWPError(args[0]) {
			t.Fatalf("arg=%T", args[0])
		}
	}, 10, 1)
	e := wp.NewWPError("code", "msg", nil)
	if !wp.IsWPErrorWithHooks(h, e) {
		t.Fatal("expected true")
	}
	if called != 1 {
		t.Fatalf("called=%d", called)
	}
	if wp.IsWPErrorWithHooks(h, "nope") {
		t.Fatal("non-error")
	}
	if called != 1 {
		t.Fatalf("action should not fire for non-error: %d", called)
	}
}

func TestWPErrorImplementsError(t *testing.T) {
	var err error = wp.NewWPError("oops", "broken", nil)
	if err.Error() != "broken" {
		t.Fatalf("Error()=%q", err.Error())
	}
}

func TestWPErrorGetAllErrorDataMergeExport(t *testing.T) {
	e := wp.NewWPError("a", "one", "d1")
	e.AddData("d2", "a")
	all := e.GetAllErrorData("a")
	if len(all) != 2 || all[0] != "d1" || all[1] != "d2" {
		t.Fatalf("GetAllErrorData=%v", all)
	}

	dst := wp.NewWPError("", "", nil)
	e.ExportTo(dst)
	if dst.GetErrorCode() != "a" || dst.GetErrorMessage("a") != "one" {
		t.Fatalf("ExportTo dst=%+v", dst)
	}
	if got := dst.GetAllErrorData("a"); len(got) != 2 {
		t.Fatalf("exported data=%v", got)
	}

	other := wp.NewWPError("b", "two", map[string]any{"status": 400})
	dst.MergeFrom(other)
	codes := dst.GetErrorCodes()
	if len(codes) != 2 || codes[0] != "a" || codes[1] != "b" {
		t.Fatalf("codes after merge=%v", codes)
	}
}

func TestWPErrorAddedAction(t *testing.T) {
	h := wp.NewHooks()
	var gotCode, gotMsg string
	var gotData any
	h.AddAction("wp_error_added", func(args ...any) {
		if len(args) < 4 {
			t.Fatalf("args=%v", args)
		}
		gotCode, _ = args[0].(string)
		gotMsg, _ = args[1].(string)
		gotData = args[2]
		if !wp.IsWPError(args[3]) {
			t.Fatalf("arg3=%T", args[3])
		}
	}, 10, 4)
	e := wp.NewWPError("", "", nil)
	e.AddWithHooks(h, "rest_forbidden", "Nope", map[string]any{"status": 403})
	if gotCode != "rest_forbidden" || gotMsg != "Nope" {
		t.Fatalf("code=%q msg=%q", gotCode, gotMsg)
	}
	data, _ := gotData.(map[string]any)
	if data["status"] != 403 {
		t.Fatalf("data=%v", gotData)
	}
}
