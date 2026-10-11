// Hooks implement the WordPress Plugin API actions and filters (P1.4).
//
// Semantics mirror wp-includes/plugin.php / class-wp-hook.php:
// lower priority runs first; equal priorities keep registration order;
// nested DoAction/ApplyFilters are supported via a current-filter stack.
package wp

import (
	"reflect"
	"sort"
	"strings"
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

	// actionCounts tracks how many times each tag has been fired (did_action, P9.27).
	actionCounts map[string]int

	// filterCounts tracks how many times each tag has been applied (did_filter, P9.28).
	filterCounts map[string]int

	// noticeCapture collects HTML during CaptureAdminNotices (P9.7).
	noticeCapture *strings.Builder
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
		byTag:        make(map[string][]callback),
		actionCounts: make(map[string]int),
		filterCounts: make(map[string]int),
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
// Increments the did_action counter for tag (P9.27).
// P9.32: when 'all' is registered, its callbacks run first while doing_filter(tag).
func (h *Hooks) DoAction(tag string, args ...any) {
	if h == nil {
		return
	}
	h.bumpAction(tag)
	hasAll := h.hasCallbacks("all")
	if hasAll {
		// WP: push tag before _wp_call_all_hook so doing_filter($tag) is true.
		h.pushCurrent(tag)
		allArgs := make([]any, 0, 1+len(args))
		allArgs = append(allArgs, tag)
		allArgs = append(allArgs, args...)
		h.runAllHook(allArgs)
	}
	cbs := h.callbacksCopy(tag)
	if len(cbs) == 0 {
		if hasAll {
			h.end()
		}
		return
	}
	if !hasAll {
		h.pushCurrent(tag)
	}
	defer h.end()
	for _, cb := range cbs {
		callAction(cb, args)
	}
}

// DoActionRefArray mirrors do_action_ref_array — unpacks args and runs DoAction (P9.27).
// Pointer / mutable values in args can be mutated by callbacks (Go ref semantics).
func (h *Hooks) DoActionRefArray(tag string, args []any) {
	if h == nil {
		return
	}
	h.DoAction(tag, args...)
}

// ApplyFilters runs filter (and action) callbacks for tag, threading value through.
// Increments did_filter (P9.28) even when no callbacks are registered.
// P9.32: when 'all' is registered, its callbacks run first while doing_filter(tag).
func (h *Hooks) ApplyFilters(tag string, value any, args ...any) any {
	if h == nil {
		return value
	}
	h.bumpFilter(tag)
	hasAll := h.hasCallbacks("all")
	if hasAll {
		h.pushCurrent(tag)
		allArgs := make([]any, 0, 2+len(args))
		allArgs = append(allArgs, tag, value)
		allArgs = append(allArgs, args...)
		h.runAllHook(allArgs)
	}
	cbs := h.callbacksCopy(tag)
	if len(cbs) == 0 {
		if hasAll {
			h.end()
		}
		return value
	}
	if !hasAll {
		h.pushCurrent(tag)
	}
	defer h.end()
	for _, cb := range cbs {
		value = callFilter(cb, value, args)
	}
	return value
}

// ApplyFiltersRefArray mirrors apply_filters_ref_array (P9.27 / P9.28).
// args[0] is the filtered value; remaining elements are extra filter args.
// Empty args returns nil (does not bump did_filter).
func (h *Hooks) ApplyFiltersRefArray(tag string, args []any) any {
	if h == nil {
		if len(args) == 0 {
			return nil
		}
		return args[0]
	}
	if len(args) == 0 {
		return nil
	}
	return h.ApplyFilters(tag, args[0], args[1:]...)
}

// ApplyFiltersDeprecated mirrors apply_filters_deprecated (P9.33).
// When no callbacks are registered on hookName, returns args[0] without
// firing deprecated_hook_run. Otherwise notifies via DeprecatedHook then
// applies the filter with ApplyFiltersRefArray.
func (h *Hooks) ApplyFiltersDeprecated(hookName string, args []any, version, replacement, message string) any {
	if len(args) == 0 {
		return nil
	}
	if h == nil || !h.HasFilter(hookName) {
		return args[0]
	}
	h.DeprecatedHook(hookName, version, replacement, message)
	return h.ApplyFiltersRefArray(hookName, args)
}

// DoActionDeprecated mirrors do_action_deprecated (P9.33 / P9.34).
// No-op when no callbacks are registered on hookName (has_action).
// Nil/empty args still fire DoActionRefArray when callbacks exist (P9.34).
func (h *Hooks) DoActionDeprecated(hookName string, args []any, version, replacement, message string) {
	if h == nil || !h.HasAction(hookName) {
		return
	}
	h.DeprecatedHook(hookName, version, replacement, message)
	if args == nil {
		args = []any{}
	}
	h.DoActionRefArray(hookName, args)
}

// LastDeprecatedHookNotice is the most recent notice text from DeprecatedHook
// when deprecated_hook_trigger_error is truthy (P9.34). Empty when suppressed.
var LastDeprecatedHookNotice string

// DeprecatedHook mirrors _deprecated_hook (P9.33 / P9.34).
// Fires deprecated_hook_run; when deprecated_hook_trigger_error allows,
// records a WP-style notice in LastDeprecatedHookNotice (no process abort).
func (h *Hooks) DeprecatedHook(hookName, version, replacement, message string) {
	if h == nil {
		return
	}
	h.DoAction("deprecated_hook_run", hookName, replacement, version, message)
	// P9.34: honor deprecated_hook_trigger_error (default true; WP_DEBUG gate omitted).
	trigger := true
	if v := h.ApplyFilters("deprecated_hook_trigger_error", trigger); v != nil {
		switch t := v.(type) {
		case bool:
			trigger = t
		case int:
			trigger = t != 0
		case string:
			trigger = t != "" && t != "0" && !strings.EqualFold(t, "false")
		}
	}
	if !trigger {
		LastDeprecatedHookNotice = ""
		return
	}
	LastDeprecatedHookNotice = DeprecatedHookNotice(hookName, version, replacement, message)
}

// DeprecatedHookNotice builds the _deprecated_hook user-facing message (P9.34).
func DeprecatedHookNotice(hookName, version, replacement, message string) string {
	extra := ""
	if strings.TrimSpace(message) != "" {
		extra = " " + message
	}
	if strings.TrimSpace(replacement) != "" {
		return "Hook " + hookName + " is deprecated since version " + version +
			"! Use " + replacement + " instead." + extra
	}
	return "Hook " + hookName + " is deprecated since version " + version +
		" with no alternative available." + extra
}

// DidAction returns how many times tag has been fired via DoAction /
// DoActionRefArray (did_action, P9.27).
func (h *Hooks) DidAction(tag string) int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.actionCounts[tag]
}

// DidFilter returns how many times tag has been applied via ApplyFilters /
// ApplyFiltersRefArray (did_filter, P9.28).
func (h *Hooks) DidFilter(tag string) int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.filterCounts[tag]
}

func (h *Hooks) bumpAction(tag string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.actionCounts == nil {
		h.actionCounts = make(map[string]int)
	}
	h.actionCounts[tag]++
}

func (h *Hooks) bumpFilter(tag string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.filterCounts == nil {
		h.filterCounts = make(map[string]int)
	}
	h.filterCounts[tag]++
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

// RemoveAllFilters removes all callbacks on tag (remove_all_filters, P9.30).
// When priorityExact >= 0, only that priority is cleared; otherwise all
// priorities are removed. Always returns true (WP return value).
func (h *Hooks) RemoveAllFilters(tag string, priorityExact int) bool {
	return h.removeAll(tag, priorityExact)
}

// RemoveAllActions removes all callbacks on tag (remove_all_actions, P9.30).
// Actions and filters share the registry — same as RemoveAllFilters.
func (h *Hooks) RemoveAllActions(tag string, priorityExact int) bool {
	return h.removeAll(tag, priorityExact)
}

func (h *Hooks) removeAll(tag string, priorityExact int) bool {
	if h == nil {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.byTag[tag]
	if len(list) == 0 {
		return true
	}
	if priorityExact < 0 {
		delete(h.byTag, tag)
		return true
	}
	kept := list[:0]
	for _, cb := range list {
		if cb.priority != priorityExact {
			kept = append(kept, cb)
		}
	}
	if len(kept) == 0 {
		delete(h.byTag, tag)
	} else {
		h.byTag[tag] = kept
	}
	return true
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

// HasFilterFunc reports whether fn is hooked on tag (has_filter with callback).
// When found, returns its priority and true — matching WP_Hook::has_filter
// returning the callback priority (P9.29). When priorityExact >= 0, requires
// that exact priority (WP $priority int check → bool).
func (h *Hooks) HasFilterFunc(tag string, fn FilterFunc, priorityExact int) (priority int, ok bool) {
	if h == nil || fn == nil {
		return 0, false
	}
	return h.hasFunc(tag, funcID(fn), priorityExact)
}

// HasActionFunc reports whether fn is hooked on tag (has_action with callback).
// See HasFilterFunc (P9.29).
func (h *Hooks) HasActionFunc(tag string, fn ActionFunc, priorityExact int) (priority int, ok bool) {
	if h == nil || fn == nil {
		return 0, false
	}
	return h.hasFunc(tag, funcID(fn), priorityExact)
}

func (h *Hooks) has(tag string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byTag[tag]) > 0
}

// hasFunc looks up a callback by id. priorityExact < 0 means "any priority"
// (return first match priority); >= 0 requires that priority (bool-style).
func (h *Hooks) hasFunc(tag string, id uintptr, priorityExact int) (int, bool) {
	if id == 0 {
		return 0, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.byTag[tag]
	for i := range list {
		if list[i].id != id {
			continue
		}
		if priorityExact >= 0 {
			if list[i].priority == priorityExact {
				return list[i].priority, true
			}
			continue
		}
		return list[i].priority, true
	}
	return 0, false
}

// CurrentFilter returns the innermost tag being run, or "" (current_filter, P9.31).
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

// CurrentAction is an alias of CurrentFilter (current_action; WP shares one stack).
func (h *Hooks) CurrentAction() string {
	return h.CurrentFilter()
}

// CurrentFilters returns a copy of the nested current_filter stack
// (outermost first), mirroring the $wp_current_filter global (P9.31).
func (h *Hooks) CurrentFilters() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.current) == 0 {
		return nil
	}
	out := make([]string, len(h.current))
	copy(out, h.current)
	return out
}

// DoingFilter reports whether tag is on the current-filter stack (doing_filter, P9.31 / P9.32).
// If tag is empty, reports whether any filter/action is running (PHP null arity).
// P9.32: during 'all' callbacks, the specific tag is on the stack (not "all").
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

// DoingAction is an alias of DoingFilter (doing_action).
func (h *Hooks) DoingAction(tag string) bool {
	return h.DoingFilter(tag)
}

func (h *Hooks) pushCurrent(tag string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = append(h.current, tag)
}

func (h *Hooks) hasCallbacks(tag string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byTag[tag]) > 0
}

func (h *Hooks) callbacksCopy(tag string) []callback {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.byTag[tag]
	if len(list) == 0 {
		return nil
	}
	// Copy so nested Add/Remove during DoAction/ApplyFilters is safe.
	out := make([]callback, len(list))
	copy(out, list)
	return out
}

// runAllHook mirrors _wp_call_all_hook / WP_Hook::do_all_hook (P9.32).
// Callbacks receive hookArgs (tag first); return values are discarded.
func (h *Hooks) runAllHook(hookArgs []any) {
	for _, cb := range h.callbacksCopy("all") {
		callAction(cb, hookArgs)
	}
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

// BeginNoticeCapture starts buffering AdminNotice writes.
func (h *Hooks) BeginNoticeCapture(buf *strings.Builder) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.noticeCapture = buf
	h.mu.Unlock()
}

// EndNoticeCapture clears the notice capture buffer pointer.
func (h *Hooks) EndNoticeCapture() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.noticeCapture = nil
	h.mu.Unlock()
}

// WriteNotice appends HTML to the active notice capture buffer.
func (h *Hooks) WriteNotice(html string) {
	if h == nil || html == "" {
		return
	}
	h.mu.Lock()
	buf := h.noticeCapture
	h.mu.Unlock()
	if buf != nil {
		buf.WriteString(html)
		if !strings.HasSuffix(html, "\n") {
			buf.WriteByte('\n')
		}
	}
}
