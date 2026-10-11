package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestCreateUserRequestErase(t *testing.T) {
	cfg := testBootConfig()
	cfg.DBName = "gopress_go"
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })

	req, err := wp.CreateUserRequest(c, "erase-req@example.com", "remove_personal_data", "pending")
	if err != nil {
		t.Fatalf("CreateUserRequest: %v", err)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", req.ID)
		_, _ = c.DB.Exec(q, args...)
	})
	if req.ID <= 0 || req.Status != "request-pending" || req.ActionName != "remove_personal_data" {
		t.Fatalf("request: %+v", req)
	}
	if _, err := wp.CreateUserRequest(c, "erase-req@example.com", "remove_personal_data", "pending"); err == nil {
		t.Fatal("expected duplicate_request")
	}
	list, err := wp.ListUserRequests(c, "remove_personal_data", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range list {
		if r.ID == req.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected request in list")
	}
	if err := wp.CompleteUserRequest(c, req.ID); err != nil {
		t.Fatalf("CompleteUserRequest: %v", err)
	}
}

func TestGenerateAndValidateUserRequestKey(t *testing.T) {
	cfg := testBootConfig()
	cfg.DBName = "gopress_go"
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	req, err := wp.CreateUserRequest(c, "key-validate@example.com", "remove_personal_data", "pending")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", req.ID)
		_, _ = c.DB.Exec(q, args...)
	})
	key, err := wp.GenerateUserRequestKey(c, req.ID)
	if err != nil || len(key) < 10 {
		t.Fatalf("key: %v %q", err, key)
	}
	if err := wp.ValidateUserRequestKey(c, req.ID, key); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := wp.ValidateUserRequestKey(c, req.ID, "wrong-key"); err == nil {
		t.Fatal("expected invalid_key")
	}
	msg := wp.PrivacyAccountRequestConfirmedMessage(c, req.ID)
	if !strings.Contains(msg, "erasure request") {
		t.Fatalf("message=%q", msg)
	}
}
