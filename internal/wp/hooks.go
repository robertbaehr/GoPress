// Hooks implement the WordPress Plugin API actions and filters (P1.4).
//
// Semantics mirror wp-includes/plugin.php / class-wp-hook.php:
// lower priority runs first; equal priorities keep registration order;
// nested DoAction/ApplyFilters are supported via a current-filter stack.
package wp

import (
	"reflect"
	"sort"
	"sync"
)

// ActionFunc is a do_action callback. Args are the values passed to DoAction,
// truncated to the callback's acceptedArgs count.
type ActionFunc func(args ...any)

// FilterFunc is an apply_filters callback. value is the filtered value; args
// are any extra arguments from ApplyFilters (after the value).
type FilterFunc func(value any, args ...any) any

// Hooks is the action/filter registry. WordPress stores actions and filters in
// one map; this type does the same so a tag can be used with either API.
//
// A process typically has one Hooks instance shared across requests via
// Context.Hooks.
type Hooks struct {
	mu sync.Mutex

	// byTag maps hook name → callbacks sorted by (priority, order).
	byTag map[string][]callback

	// nextOrder is a monotonic insertion counter for stable equal-priority order.
	nextOrder int

	// current is the nested current_filter / current_action stack (top at end).
	current []string
}

type callback struct {
	priority     int
	acceptedArgs int
	order        int
	id           uintptr // reflect code pointer for Remove*
	action       ActionFunc
	filter       FilterFunc
}

// NewHooks returns an empty registry.
func NewHooks() *Hooks {
	return &Hooks{
		byTag: make(map[string][]callback),
	}
}

// AddAction registers fn on tag. Lower priority runs first (WordPress default is 10).
// acceptedArgs limits how many DoAction args are passed (0 means none).
func (h *Hooks) AddAction(tag string, fn ActionFunc, priority int, acceptedArgs int) {
	if h == nil || fn == nil {
		return
	}
	h.add(tag, priority, acceptedArgs, funcID(fn), fn, nil)
}

// AddFilter registers fn on tag. Lower priority runs first.
// acceptedArgs counts the filtered value plus extra ApplyFilters args.
func (h *Hooks) AddFilter(tag string, fn FilterFunc, priority int, acceptedArgs int) {
	if h == nil || fn == nil {
		return
	}
	h.add(tag, priority, acceptedArgs, funcID(fn), nil, fn)
}

func (h *Hooks) add(tag string, priority, acceptedArgs int, id uintptr, action ActionFunc, filter FilterFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	list := h.byTag[tag]
	// Same callback + priority replaces (WP unique-id behavior).
	for i := range list {
		if list[i].id == id && list[i].priority == priority {
			list[i].acceptedArgs = acceptedArgs
			list[i].action = action
			list[i].filter = filter
			h.byTag[tag] = list
			return
		}
	}

	h.nextOrder++
	list = append(list, callback{
		priority:     priority,
		acceptedArgs: acceptedArgs,
		order:        h.nextOrder,
		id:           id,
		action:       action,
		filter:       filter,
	})
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].priority != list[j].priority {
			return list[i].priority < list[j].priority
		}
		return list[i].order < list[j].order
	})
	h.byTag[tag] = list
}

// DoAction runs all callbacks for tag (actions and filters; filter returns discarded).
func (h *Hooks) DoAction(tag string, args ...any) {
	if h == nil {
		return
	}
	cbs, _ := h.begin(tag)
	defer h.end()
	for _, cb := range cbs {
		callAction(cb, args)
	}
}

// ApplyFilters runs filter (and action) callbacks for tag, threading value through.
func (h *Hooks) ApplyFilters(tag string, value any, args ...any) any {
	if h == nil {
		return value
	}
	cbs, _ := h.begin(tag)
	defer h.end()
	for _, cb := range cbs {
		value = callFilter(cb, value, args)
	}
	return value
}

// RemoveAction removes an action previously added with the same fn and priority.
func (h *Hooks) RemoveAction(tag string, fn ActionFunc, priority int) bool {
	if h == nil || fn == nil {
		return false
	}
	return h.remove(tag, funcID(fn), priority)
}

// RemoveFilter removes a filter previously added with the same fn and priority.
func (h *Hooks) RemoveFilter(tag string, fn FilterFunc, priority int) bool {
	if h == nil || fn == nil {
		return false
	}
	return h.remove(tag, funcID(fn), priority)
}

func (h *Hooks) remove(tag string, id uintptr, priority int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.byTag[tag]
	for i := range list {
		if list[i].id == id && list[i].priority == priority {
			h.byTag[tag] = append(list[:i], list[i+1:]...)
			if len(h.byTag[tag]) == 0 {
				delete(h.byTag, tag)
			}
			return true
		}
	}
	return false
}

// HasAction reports whether tag has at least one registered callback.
// Like WordPress, actions and filters share the registry.
func (h *Hooks) HasAction(tag string) bool {
	return h.has(tag)
}

// HasFilter reports whether tag has at least one registered callback.
func (h *Hooks) HasFilter(tag string) bool {
	return h.has(tag)
}

func (h *Hooks) has(tag string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byTag[tag]) > 0
}

// CurrentFilter returns the innermost tag being run, or "".
func (h *Hooks) CurrentFilter() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.current) == 0 {
		return ""
	}
	return h.current[len(h.current)-1]
}

// CurrentAction is an alias of CurrentFilter (WordPress shares one stack).
func (h *Hooks) CurrentAction() string {
	return h.CurrentFilter()
}

// DoingFilter reports whether tag is on the current-filter stack.
// If tag is empty, reports whether any filter/action is running.
func (h *Hooks) DoingFilter(tag string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if tag == "" {
		return len(h.current) > 0
	}
	for _, t := range h.current {
		if t == tag {
			return true
		}
	}
	return false
}

// DoingAction is an alias of DoingFilter.
func (h *Hooks) DoingAction(tag string) bool {
	return h.DoingFilter(tag)
}

func (h *Hooks) begin(tag string) ([]callback, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = append(h.current, tag)
	list := h.byTag[tag]
	if len(list) == 0 {
		return nil, false
	}
	// Copy so nested Add/Remove during DoAction/ApplyFilters is safe.
	out := make([]callback, len(list))
	copy(out, list)
	return out, true
}

func (h *Hooks) end() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.current) > 0 {
		h.current = h.current[:len(h.current)-1]
	}
}

func callAction(cb callback, args []any) {
	trimmed := trimArgs(args, cb.acceptedArgs)
	if cb.action != nil {
		cb.action(trimmed...)
		return
	}
	if cb.filter != nil {
		// Action/filter registry is unified; ignore filter return values.
		if len(trimmed) == 0 {
			_ = cb.filter(nil)
			return
		}
		_ = cb.filter(trimmed[0], trimmed[1:]...)
	}
}

func callFilter(cb callback, value any, args []any) any {
	all := make([]any, 0, 1+len(args))
	all = append(all, value)
	all = append(all, args...)
	trimmed := trimArgs(all, cb.acceptedArgs)
	if cb.filter != nil {
		if len(trimmed) == 0 {
			return cb.filter(nil)
		}
		return cb.filter(trimmed[0], trimmed[1:]...)
	}
	if cb.action != nil {
		cb.action(trimmed...)
	}
	return value
}

func trimArgs(args []any, accepted int) []any {
	if accepted <= 0 {
		return nil
	}
	if accepted >= len(args) {
		return args
	}
	return args[:accepted]
}

func funcID(fn any) uintptr {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func || v.IsNil() {
		return 0
	}
	return v.Pointer()
}
