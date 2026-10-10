package wp_test

import (
	"strconv"
	"testing"
	"time"
)

func TestTransientSetGetDelete(t *testing.T) {
	c := testBootQuery(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	name := "gopress_test_transient_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if !c.SetTransient(name, "hello", 0) {
		t.Fatal("SetTransient failed")
	}
	v, ok := c.GetTransient(name)
	if !ok || v != "hello" {
		t.Fatalf("GetTransient=%v ok=%v", v, ok)
	}
	// Options persistence.
	if got := c.Options.GetOption("_transient_"+name, nil); got != "hello" {
		t.Fatalf("option=%v", got)
	}
	// Cache mirror.
	if cv, cok := c.Cache.Get(name, "transient"); !cok || cv != "hello" {
		t.Fatalf("cache=%v ok=%v", cv, cok)
	}
	if !c.DeleteTransient(name) {
		t.Fatal("DeleteTransient failed")
	}
	if _, ok := c.GetTransient(name); ok {
		t.Fatal("expected miss after delete")
	}
	if c.Options.GetOption("_transient_"+name, nil) != nil {
		t.Fatal("option still present")
	}
}

func TestTransientExpiration(t *testing.T) {
	c := testBootQuery(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	name := "gopress_test_transient_exp_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if !c.SetTransient(name, "soon", 1) {
		t.Fatal("SetTransient failed")
	}
	if _, ok := c.GetTransient(name); !ok {
		t.Fatal("expected hit before expiry")
	}
	// Force timeout into the past via options.
	past := strconv.FormatInt(time.Now().Unix()-10, 10)
	if _, err := c.Options.UpdateOption("_transient_timeout_"+name, past, ""); err != nil {
		t.Fatal(err)
	}
	c.Cache.Delete(name, "transient")
	c.Cache.Delete(name+"_timeout", "transient")
	if _, ok := c.GetTransient(name); ok {
		t.Fatal("expected miss after expiry")
	}
	if c.Options.GetOption("_transient_"+name, nil) != nil {
		t.Fatal("expired transient should be deleted")
	}
}

func TestTransientUpdate(t *testing.T) {
	c := testBootQuery(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	name := "gopress_test_transient_upd_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if !c.SetTransient(name, "one", 60) {
		t.Fatal("set1")
	}
	if !c.SetTransient(name, "two", 60) {
		t.Fatal("set2")
	}
	v, ok := c.GetTransient(name)
	if !ok || v != "two" {
		t.Fatalf("got %v ok=%v", v, ok)
	}
	_ = c.DeleteTransient(name)
}
