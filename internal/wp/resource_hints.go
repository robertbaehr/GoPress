package wp

import (
	"fmt"
	"net/url"
	"strings"
)

// ResourceHintsHTML mirrors wp_resource_hints() markup (P3.137 / P3.138 / P3.139 / P3.140).
// Default dns-prefetch hosts come from DependenciesUniqueHosts (empty until
// remote script/style enqueue is wired). Filters wp_resource_hints per relation.
// P3.138: bare hostnames, case-insensitive dns-prefetch host dedupe.
// P3.139: case-insensitive scheme+host preconnect dedupe; crossorigin preserved.
// P3.140: prefetch/prerender http(s)-only; case-insensitive href dedupe; as/type/pr.
func ResourceHintsHTML(c *Context) string {
	hints := map[string][]any{
		"dns-prefetch": dependenciesUniqueHostsAny(c),
		"preconnect":   nil,
		"prefetch":     nil,
		"prerender":    nil,
	}
	var b strings.Builder
	for _, relation := range []string{"dns-prefetch", "preconnect", "prefetch", "prerender"} {
		urls := hints[relation]
		if urls == nil {
			urls = []any{}
		}
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("wp_resource_hints", urls, relation); v != nil {
				switch t := v.(type) {
				case []any:
					urls = t
				case []string:
					urls = make([]any, len(t))
					for i, s := range t {
						urls[i] = s
					}
				}
			}
		}
		unique := map[string]map[string]string{}
		order := make([]string, 0)
		seenHosts := map[string]bool{}      // P3.138: case-insensitive host dedupe for dns-prefetch
		seenPreconnect := map[string]bool{} // P3.139: scheme+host (lower) dedupe
		seenPrefetch := map[string]bool{}   // P3.140: case-insensitive href dedupe
		for _, item := range urls {
			atts := map[string]string{}
			href := ""
			switch t := item.(type) {
			case string:
				href = t
			case map[string]any:
				if h, ok := t["href"].(string); ok {
					href = h
					for k, v := range t {
						if s, ok := v.(string); ok {
							atts[k] = s
						} else if s, ok := scalarString(v); ok {
							atts[k] = s
						}
					}
				} else {
					continue
				}
			case map[string]string:
				if h, ok := t["href"]; ok {
					href = h
					for k, v := range t {
						atts[k] = v
					}
				} else {
					continue
				}
			default:
				continue
			}
			href = strings.TrimSpace(href)
			// P3.138: accept bare hostnames before EscURL (which may reject them).
			if (relation == "dns-prefetch" || relation == "preconnect") && isBareHostname(href) {
				href = "//" + href
			}
			href = strings.TrimSpace(EscURL(href))
			if href == "" {
				continue
			}
			if relation == "preconnect" || relation == "dns-prefetch" {
				parsed, err := url.Parse(href)
				if err != nil || parsed.Host == "" {
					// Protocol-relative //host
					if strings.HasPrefix(href, "//") {
						parsed, err = url.Parse("https:" + href)
					}
					if err != nil || parsed == nil || parsed.Host == "" {
						continue
					}
				}
				hostKey := strings.ToLower(parsed.Host)
				if relation == "dns-prefetch" {
					if seenHosts[hostKey] {
						continue
					}
					seenHosts[hostKey] = true
				}
				if relation == "preconnect" && parsed.Scheme != "" {
					href = parsed.Scheme + "://" + parsed.Host
					// P3.139: case-insensitive scheme+host dedupe (keep first casing).
					pcKey := strings.ToLower(parsed.Scheme) + "://" + hostKey
					if seenPreconnect[pcKey] {
						continue
					}
					seenPreconnect[pcKey] = true
				} else {
					href = "//" + parsed.Host
					if relation == "preconnect" {
						pcKey := "//" + hostKey
						if seenPreconnect[pcKey] {
							continue
						}
						seenPreconnect[pcKey] = true
					}
				}
				href = EscURL(href)
				if href == "" {
					continue
				}
			}
			// P3.140: prefetch/prerender — WP esc_url(..., array('http','https')).
			if relation == "prefetch" || relation == "prerender" {
				parsed, err := url.Parse(href)
				if err != nil || parsed == nil {
					continue
				}
				scheme := strings.ToLower(parsed.Scheme)
				if scheme != "http" && scheme != "https" {
					continue
				}
				pfKey := strings.ToLower(href)
				if seenPrefetch[pfKey] {
					continue
				}
				seenPrefetch[pfKey] = true
			}
			if _, seen := unique[href]; seen {
				continue
			}
			atts["rel"] = relation
			atts["href"] = href
			unique[href] = atts
			order = append(order, href)
		}
		for _, href := range order {
			atts := unique[href]
			b.WriteString("<link")
			// Preserve insertion order: for string URLs rel then href; for maps,
			// emit non-rel keys first then rel/href last overwrite order like PHP.
			keys := resourceHintAttrKeys(atts)
			for _, k := range keys {
				v := atts[k]
				if k == "href" {
					b.WriteString(fmt.Sprintf(" %s='%s'", k, EscURL(v)))
				} else {
					b.WriteString(fmt.Sprintf(" %s='%s'", k, EscAttr(v)))
				}
			}
			b.WriteString(" />\n")
		}
	}
	return b.String()
}

// isBareHostname reports a host-only token (no scheme/path/query) for dns-prefetch (P3.138).
func isBareHostname(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "://") || strings.HasPrefix(s, "//") {
		return false
	}
	if strings.ContainsAny(s, "/?# ") {
		return false
	}
	// Require a dot (example.com) or bracketed IPv6; reject lone words.
	if !strings.Contains(s, ".") && !strings.HasPrefix(s, "[") {
		return false
	}
	return true
}

func dependenciesUniqueHostsAny(c *Context) []any {
	hosts := DependenciesUniqueHosts(c)
	out := make([]any, len(hosts))
	for i, h := range hosts {
		out[i] = h
	}
	return out
}

// DependenciesUniqueHosts mirrors wp_dependencies_unique_hosts (P3.137).
// Returns unique remote hosts from enqueued scripts/styles; empty until wired.
func DependenciesUniqueHosts(c *Context) []string {
	_ = c
	return nil
}

func scalarString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case fmt.Stringer:
		return t.String(), true
	case int:
		return fmt.Sprintf("%d", t), true
	case float64:
		return fmt.Sprintf("%v", t), true
	case bool:
		if t {
			return "1", true
		}
		return "0", true
	default:
		return "", false
	}
}

func resourceHintAttrKeys(atts map[string]string) []string {
	// PHP array order: string URLs set rel then href; attr maps keep href first, append rel.
	allowed := map[string]bool{"as": true, "crossorigin": true, "href": true, "pr": true, "rel": true, "type": true}
	extra := 0
	for k := range atts {
		if k != "rel" && k != "href" && allowed[k] {
			extra++
		}
	}
	var keys []string
	if extra == 0 {
		for _, k := range []string{"rel", "href"} {
			if _, ok := atts[k]; ok {
				keys = append(keys, k)
			}
		}
		return keys
	}
	for _, k := range []string{"href", "as", "crossorigin", "pr", "type", "rel"} {
		if _, ok := atts[k]; ok && allowed[k] {
			keys = append(keys, k)
		}
	}
	return keys
}

// RegisterDefaultResourceHintsHook wires wp_resource_hints onto wp_head at priority 2 (P3.137).
// Front rendering still calls ResourceHintsHTML explicitly because Head does not capture output.
func RegisterDefaultResourceHintsHook(c *Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("wp_head", func(args ...any) {
		_ = ResourceHintsHTML(c)
	}, 2, 0)
}
