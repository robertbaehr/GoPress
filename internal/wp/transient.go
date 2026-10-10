// Transients API (P2.12) — set/get/delete_transient via options + object cache.
package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const transientCacheGroup = "transient"

func transientOption(name string) string {
	return "_transient_" + name
}

func transientTimeoutOption(name string) string {
	return "_transient_timeout_" + name
}

// SetTransient stores a transient value for expiration seconds (0 = no expiry).
// Persists to the options table and mirrors into the object cache group "transient".
func (c *Context) SetTransient(name string, value any, expiration int) bool {
	name = strings.TrimSpace(name)
	if c == nil || c.Options == nil || name == "" {
		return false
	}
	if expiration < 0 {
		expiration = 0
	}

	opt := transientOption(name)
	timeoutOpt := transientTimeoutOption(name)
	serialized := optionValueString(value)

	var ok bool
	if expiration > 0 {
		deadline := strconv.FormatInt(time.Now().Unix()+int64(expiration), 10)
		if cur := c.Options.GetOption(opt, nil); cur == nil {
			_, _ = c.Options.AddOption(timeoutOpt, deadline, "no")
			added, err := c.Options.AddOption(opt, serialized, "no")
			ok = err == nil && added
		} else if c.Options.GetOption(timeoutOpt, nil) == nil {
			_, _ = c.Options.DeleteOption(opt)
			_, _ = c.Options.AddOption(timeoutOpt, deadline, "no")
			added, err := c.Options.AddOption(opt, serialized, "no")
			ok = err == nil && added
		} else {
			_, _ = c.Options.UpdateOption(timeoutOpt, deadline, "")
			updated, err := c.Options.UpdateOption(opt, serialized, "")
			ok = err == nil && (updated || optionValueString(cur) == serialized)
			if !ok && err == nil {
				// UpdateOption returns false when value unchanged; still success.
				ok = true
			}
		}
	} else {
		// No expiration: drop any prior timeout and store (autoload ok).
		_, _ = c.Options.DeleteOption(timeoutOpt)
		if cur := c.Options.GetOption(opt, nil); cur == nil {
			added, err := c.Options.AddOption(opt, serialized, "yes")
			ok = err == nil && added
		} else {
			updated, err := c.Options.UpdateOption(opt, serialized, "")
			ok = err == nil && (updated || optionValueString(cur) == serialized)
			if !ok && err == nil {
				ok = true
			}
		}
	}

	if ok && c.Cache != nil {
		c.Cache.Set(name, serialized, transientCacheGroup)
		if expiration > 0 {
			c.Cache.Set(name+"_timeout", time.Now().Unix()+int64(expiration), transientCacheGroup)
		} else {
			c.Cache.Delete(name+"_timeout", transientCacheGroup)
		}
	}
	return ok
}

// GetTransient returns the transient value and true when present and unexpired.
func (c *Context) GetTransient(name string) (any, bool) {
	name = strings.TrimSpace(name)
	if c == nil || name == "" {
		return nil, false
	}

	if c.Cache != nil {
		if v, ok := c.Cache.Get(name, transientCacheGroup); ok {
			if to, ok := c.Cache.Get(name+"_timeout", transientCacheGroup); ok {
				deadline, err := cacheInt64(to)
				if err == nil && deadline > 0 && deadline < time.Now().Unix() {
					c.DeleteTransient(name)
					return nil, false
				}
			}
			return v, true
		}
	}

	if c.Options == nil {
		return nil, false
	}

	opt := transientOption(name)
	timeoutOpt := transientTimeoutOption(name)

	// Non-autoloaded transients carry a timeout option.
	if c.Options.GetOption(opt, nil) != nil {
		if to := c.Options.GetOption(timeoutOpt, nil); to != nil {
			deadline, err := parseOptionInt64(to)
			if err == nil && deadline > 0 && deadline < time.Now().Unix() {
				c.DeleteTransient(name)
				return nil, false
			}
		}
	}

	v := c.Options.GetOption(opt, nil)
	if v == nil {
		return nil, false
	}
	if c.Cache != nil {
		c.Cache.Set(name, v, transientCacheGroup)
		if to := c.Options.GetOption(timeoutOpt, nil); to != nil {
			if deadline, err := parseOptionInt64(to); err == nil {
				c.Cache.Set(name+"_timeout", deadline, transientCacheGroup)
			}
		}
	}
	return v, true
}

// DeleteTransient removes a transient from options and the object cache.
func (c *Context) DeleteTransient(name string) bool {
	name = strings.TrimSpace(name)
	if c == nil || name == "" {
		return false
	}
	deleted := false
	if c.Options != nil {
		ok, err := c.Options.DeleteOption(transientOption(name))
		if err == nil && ok {
			deleted = true
		}
		_, _ = c.Options.DeleteOption(transientTimeoutOption(name))
	}
	if c.Cache != nil {
		if c.Cache.Delete(name, transientCacheGroup) {
			deleted = true
		}
		c.Cache.Delete(name+"_timeout", transientCacheGroup)
	}
	return deleted
}

func parseOptionInt64(v any) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	case float64:
		return int64(t), nil
	default:
		return 0, fmt.Errorf("not an int: %T", v)
	}
}

func cacheInt64(v any) (int64, error) {
	return parseOptionInt64(v)
}
