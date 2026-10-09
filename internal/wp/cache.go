// Object cache (P1.6) — in-memory port of WP_Object_Cache / cache.php.
//
// Groups isolate keys (WordPress default group is "default"). Expiration is
// accepted for API parity with wp_cache_* but ignored in this process-local
// implementation, matching core WP_Object_Cache.
package wp

import (
	"fmt"
	"strconv"
	"sync"
)

// ObjectCache is a thread-safe in-memory object cache keyed by (group, key).
type ObjectCache struct {
	mu     sync.RWMutex
	groups map[string]map[string]any
}

// NewObjectCache returns an empty cache.
func NewObjectCache() *ObjectCache {
	return &ObjectCache{
		groups: make(map[string]map[string]any),
	}
}

func normGroup(group string) string {
	if group == "" {
		return "default"
	}
	return group
}

func (c *ObjectCache) groupLocked(group string) map[string]any {
	g, ok := c.groups[group]
	if !ok {
		g = make(map[string]any)
		c.groups[group] = g
	}
	return g
}

// Get returns the cached value and true if present.
func (c *ObjectCache) Get(key, group string) (any, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	group = normGroup(group)
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.groups[group]
	if !ok {
		return nil, false
	}
	v, ok := g[key]
	if !ok {
		return nil, false
	}
	return v, true
}

// Set stores value under key in group. Always succeeds for non-empty keys.
func (c *ObjectCache) Set(key string, value any, group string) bool {
	if c == nil || key == "" {
		return false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groupLocked(group)[key] = value
	return true
}

// Add stores value only if the key is missing. Returns false if already present.
func (c *ObjectCache) Add(key string, value any, group string) bool {
	if c == nil || key == "" {
		return false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	g := c.groupLocked(group)
	if _, exists := g[key]; exists {
		return false
	}
	g[key] = value
	return true
}

// Replace updates value only if the key already exists.
func (c *ObjectCache) Replace(key string, value any, group string) bool {
	if c == nil || key == "" {
		return false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[group]
	if !ok {
		return false
	}
	if _, exists := g[key]; !exists {
		return false
	}
	g[key] = value
	return true
}

// Delete removes key from group. Returns false if missing.
func (c *ObjectCache) Delete(key, group string) bool {
	if c == nil || key == "" {
		return false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[group]
	if !ok {
		return false
	}
	if _, exists := g[key]; !exists {
		return false
	}
	delete(g, key)
	return true
}

// Flush clears the entire cache.
func (c *ObjectCache) Flush() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groups = make(map[string]map[string]any)
	return true
}

// FlushGroup clears one group.
func (c *ObjectCache) FlushGroup(group string) bool {
	if c == nil {
		return false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.groups, group)
	return true
}

// Incr increments a numeric cache value by offset (default 1). Returns the new
// value, or false if the key is missing. Non-numeric values become 0 before incr.
// Results are clamped at 0 (WordPress behavior).
func (c *ObjectCache) Incr(key string, offset int64, group string) (int64, bool) {
	return c.nudge(key, offset, group)
}

// Decr decrements a numeric cache value by offset (default 1).
func (c *ObjectCache) Decr(key string, offset int64, group string) (int64, bool) {
	return c.nudge(key, -offset, group)
}

func (c *ObjectCache) nudge(key string, delta int64, group string) (int64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	group = normGroup(group)
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[group]
	if !ok {
		return 0, false
	}
	cur, exists := g[key]
	if !exists {
		return 0, false
	}
	n, ok := asInt64(cur)
	if !ok {
		n = 0
	}
	n += delta
	if n < 0 {
		n = 0
	}
	g[key] = n
	return n, true
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float32:
		return int64(n), true
	case float64:
		return int64(n), true
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case fmt.Stringer:
		parsed, err := strconv.ParseInt(n.String(), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
