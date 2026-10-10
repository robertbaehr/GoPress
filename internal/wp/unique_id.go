package wp

import "sync"

// Process-wide counters for wp_unique_id / wp_unique_prefixed_id (P2.39).
var (
	uniqueIDMu      sync.Mutex
	uniqueIDCounter int
	uniquePrefixed  = map[string]int{}
)

// UniqueID returns prefix concatenated with an incrementing counter (wp_unique_id).
// The counter is process-wide and starts at 1.
func UniqueID(prefix string) string {
	uniqueIDMu.Lock()
	uniqueIDCounter++
	n := uniqueIDCounter
	uniqueIDMu.Unlock()
	return prefix + itoaPositive(n)
}

// UniquePrefixedID returns prefix concatenated with a per-prefix incrementing
// counter (wp_unique_prefixed_id). Each prefix has an independent counter
// starting at 1.
func UniquePrefixedID(prefix string) string {
	uniqueIDMu.Lock()
	uniquePrefixed[prefix]++
	n := uniquePrefixed[prefix]
	uniqueIDMu.Unlock()
	return prefix + itoaPositive(n)
}

func itoaPositive(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// ResetUniqueIDsForTest clears unique ID counters (tests only).
func ResetUniqueIDsForTest() {
	uniqueIDMu.Lock()
	uniqueIDCounter = 0
	uniquePrefixed = map[string]int{}
	uniqueIDMu.Unlock()
}
