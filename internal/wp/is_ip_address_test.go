package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsIPAddressIPv4(t *testing.T) {
	for _, in := range []string{"127.0.0.1", "192.168.0.1", "0.0.0.0", "255.255.255.255", "999.999.999.999"} {
		if got := wp.IsIPAddress(in); got != 4 {
			t.Fatalf("IsIPAddress(%q)=%d want 4", in, got)
		}
	}
}

func TestIsIPAddressIPv6(t *testing.T) {
	for _, in := range []string{
		"::1",
		"2001:db8::1",
		"[2001:db8::1]",
		"fe80::1",
	} {
		if got := wp.IsIPAddress(in); got != 6 {
			t.Fatalf("IsIPAddress(%q)=%d want 6", in, got)
		}
	}
}

func TestIsIPAddressReject(t *testing.T) {
	for _, in := range []string{"", "localhost", "example.com", "127.0.0", "1.2.3.4.5", "not-an-ip"} {
		if got := wp.IsIPAddress(in); got != 0 {
			t.Fatalf("IsIPAddress(%q)=%d want 0", in, got)
		}
	}
}
