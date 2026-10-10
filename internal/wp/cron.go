// Cron (P3.8 / P2.16) — schedule events into the cron option; spawn_cron locks
// and ServeCron runs due hooks (rescheduling recurring ones).
package wp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// WP_CRON_LOCK_TIMEOUT — seconds before a doing_cron lock is considered stale.
const CronLockTimeout = 60

// Known recurrence intervals (wp_get_schedules subset).
var cronIntervals = map[string]int64{
	"hourly":     3600,
	"twicedaily": 43200,
	"daily":      86400,
	"weekly":     604800,
}

// GetSchedules returns recurrence name → interval seconds (wp_get_schedules stub).
func GetSchedules() map[string]int64 {
	out := make(map[string]int64, len(cronIntervals))
	for k, v := range cronIntervals {
		out[k] = v
	}
	return out
}

// ScheduleEvent adds a recurring cron hook to the cron option (wp_schedule_event).
func ScheduleEvent(c *Context, timestamp int64, recurrence, hook string, args []any) error {
	if recurrence == "" {
		return fmt.Errorf("wp: ScheduleEvent: empty recurrence")
	}
	interval, ok := cronIntervals[recurrence]
	if !ok {
		return fmt.Errorf("wp: ScheduleEvent: unknown recurrence %q", recurrence)
	}
	return putCronEvent(c, timestamp, hook, args, recurrence, interval)
}

// ScheduleSingleEvent adds a one-shot cron hook (wp_schedule_single_event).
func ScheduleSingleEvent(c *Context, timestamp int64, hook string, args []any) error {
	return putCronEvent(c, timestamp, hook, args, "", 0)
}

// ClearScheduledHook removes all cron events for hook+args (wp_clear_scheduled_hook stub).
func ClearScheduledHook(c *Context, hook string, args []any) error {
	if c == nil || c.Options == nil || hook == "" {
		return fmt.Errorf("wp: ClearScheduledHook: invalid args")
	}
	cron, err := loadCronOption(c)
	if err != nil {
		return err
	}
	if cron == nil {
		return nil
	}
	key := cronArgsKey(args)
	for k, v := range cron {
		ts := cronTimestamp(k)
		if ts <= 0 {
			continue
		}
		bucket, _ := v.(map[any]any)
		if bucket == nil {
			continue
		}
		hookBucket, _ := bucket[hook].(map[any]any)
		if hookBucket == nil {
			continue
		}
		delete(hookBucket, key)
		if len(hookBucket) == 0 {
			delete(bucket, hook)
		} else {
			bucket[hook] = hookBucket
		}
		if len(bucket) == 0 {
			deleteCronTimestamp(cron, ts)
		} else {
			cron[int64(ts)] = bucket
		}
	}
	serialized, err := phpSerialize(cron)
	if err != nil {
		return err
	}
	_, err = c.Options.UpdateOption("cron", serialized, "")
	return err
}

// NextScheduled returns the next Unix timestamp for hook+args, or 0 (wp_next_scheduled).
func NextScheduled(c *Context, hook string, args []any) int64 {
	if c == nil || hook == "" {
		return 0
	}
	cron, err := loadCronOption(c)
	if err != nil || cron == nil {
		return 0
	}
	key := cronArgsKey(args)
	var next int64
	for k, v := range cron {
		ts := cronTimestamp(k)
		if ts <= 0 {
			continue
		}
		bucket, _ := v.(map[any]any)
		if bucket == nil {
			continue
		}
		hookBucket, _ := bucket[hook].(map[any]any)
		if hookBucket == nil {
			continue
		}
		if _, ok := hookBucket[key]; !ok {
			continue
		}
		if next == 0 || ts < next {
			next = ts
		}
	}
	return next
}

// SpawnCron mirrors spawn_cron: lock + ready-job check. It does not open an
// HTTP connection; callers hit /wp-cron.php (or ServeCron) to run jobs.
func SpawnCron(c *Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	if c.Config != nil && c.Config.DisableWPCron {
		return false
	}
	now := float64(time.Now().Unix())
	lock := doingCronLock(c)
	if lock+float64(CronLockTimeout) > now {
		return false
	}
	ready := readyCronTimestamps(c, int64(now))
	if len(ready) == 0 {
		return false
	}
	lockVal := fmt.Sprintf("%.22f", now)
	_ = c.SetTransient("doing_cron", lockVal, CronLockTimeout)
	_, _ = c.Options.UpdateOption("doing_cron", lockVal, "no")
	return true
}

// MaybeSpawnCron is a lightweight front-request hook: spawn when due work exists.
func MaybeSpawnCron(c *Context) {
	_ = SpawnCron(c)
}

// ServeCron handles GET/POST /wp-cron.php — runs due events then returns 200.
func ServeCron(w http.ResponseWriter, r *http.Request, c *Context) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Expires", "Wed, 11 Jan 1984 05:00:00 GMT")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate, max-age=0")
	// P2.67: DOING_CRON / wp_doing_cron for the duration of wp-cron.php.
	if c != nil && c.Scope != nil {
		c.Scope.SetCron(true)
	}
	if c != nil {
		_ = RunCron(c)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte{})
	_ = r
}

// RunCron executes due cron hooks and rewrites the cron option (wp-cron.php body).
func RunCron(c *Context) int {
	if c == nil || c.Options == nil {
		return 0
	}
	now := time.Now().Unix()
	ready := readyCronTimestamps(c, now)
	if len(ready) == 0 {
		return 0
	}
	cron, err := loadCronOption(c)
	if err != nil || cron == nil {
		return 0
	}
	ran := 0
	for _, ts := range ready {
		bucket := cronBucket(cron, ts)
		if bucket == nil {
			continue
		}
		for hookKey, hooksRaw := range bucket {
			hook, _ := hookKey.(string)
			if hook == "" {
				hook = fmt.Sprint(hookKey)
			}
			hookBucket, _ := hooksRaw.(map[any]any)
			if hookBucket == nil {
				continue
			}
			for argsKey, entryRaw := range hookBucket {
				entry, _ := entryRaw.(map[any]any)
				if entry == nil {
					continue
				}
				args := phpArrayToSlice(entry["args"])
				if c.Hooks != nil {
					c.Hooks.DoAction(hook, args...)
				}
				ran++
				schedule, _ := entry["schedule"].(string)
				if schedule == "" {
					if s, ok := entry["schedule"].(bool); ok && !s {
						schedule = ""
					}
				}
				delete(hookBucket, argsKey)
				if schedule != "" {
					interval := cronIntervals[schedule]
					if v, ok := entry["interval"].(int64); ok && v > 0 {
						interval = v
					}
					if interval <= 0 {
						interval = cronIntervals[schedule]
					}
					nextTS := ts + interval
					if nextTS <= now {
						nextTS = now + interval
					}
					_ = putCronEventInto(cron, nextTS, hook, args, schedule, interval)
				}
			}
			if len(hookBucket) == 0 {
				delete(bucket, hookKey)
			} else {
				bucket[hookKey] = hookBucket
			}
		}
		if len(bucket) == 0 {
			deleteCronTimestamp(cron, ts)
		} else {
			cron[int64(ts)] = bucket
		}
	}
	serialized, err := phpSerialize(cron)
	if err == nil {
		_, _ = c.Options.UpdateOption("cron", serialized, "")
	}
	_ = c.DeleteTransient("doing_cron")
	_, _ = c.Options.UpdateOption("doing_cron", "", "no")
	return ran
}

func doingCronLock(c *Context) float64 {
	if v, ok := c.GetTransient("doing_cron"); ok {
		switch t := v.(type) {
		case string:
			f, _ := strconv.ParseFloat(t, 64)
			return f
		case float64:
			return t
		}
	}
	raw := c.Options.GetOption("doing_cron", nil)
	switch t := raw.(type) {
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case float64:
		return t
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func readyCronTimestamps(c *Context, now int64) []int64 {
	cron, err := loadCronOption(c)
	if err != nil || cron == nil {
		return nil
	}
	var out []int64
	for k := range cron {
		ts := cronTimestamp(k)
		if ts > 0 && ts <= now {
			out = append(out, ts)
		}
	}
	// Ascending order so earlier jobs run first.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func cronTimestamp(k any) int64 {
	switch t := k.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		if t == "version" {
			return 0
		}
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

func cronBucket(cron map[any]any, ts int64) map[any]any {
	if b, ok := cron[ts].(map[any]any); ok {
		return b
	}
	if b, ok := cron[int64(ts)].(map[any]any); ok {
		return b
	}
	return nil
}

func deleteCronTimestamp(cron map[any]any, ts int64) {
	delete(cron, ts)
	delete(cron, int64(ts))
}

func putCronEvent(c *Context, timestamp int64, hook string, args []any, schedule string, interval int64) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: cron: nil context/options")
	}
	if hook == "" {
		return fmt.Errorf("wp: cron: empty hook")
	}
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}

	cron, err := loadCronOption(c)
	if err != nil {
		return err
	}
	if cron == nil {
		cron = map[any]any{"version": int64(2)}
	}
	if _, ok := cron["version"]; !ok {
		cron["version"] = int64(2)
	}
	if err := putCronEventInto(cron, timestamp, hook, args, schedule, interval); err != nil {
		return err
	}
	serialized, err := phpSerialize(cron)
	if err != nil {
		return fmt.Errorf("wp: cron serialize: %w", err)
	}
	_, err = c.Options.UpdateOption("cron", serialized, "")
	return err
}

func putCronEventInto(cron map[any]any, timestamp int64, hook string, args []any, schedule string, interval int64) error {
	argsArr := sliceToPHPArray(args)
	key := cronArgsKey(args)
	entry := map[any]any{
		"schedule": false,
		"args":     argsArr,
	}
	if schedule != "" {
		entry["schedule"] = schedule
		entry["interval"] = interval
	}

	tsBucket, _ := cron[timestamp].(map[any]any)
	if tsBucket == nil {
		tsBucket, _ = cron[int64(timestamp)].(map[any]any)
	}
	if tsBucket == nil {
		tsBucket = map[any]any{}
	}
	hookBucket, _ := tsBucket[hook].(map[any]any)
	if hookBucket == nil {
		hookBucket = map[any]any{}
	}
	hookBucket[key] = entry
	tsBucket[hook] = hookBucket
	cron[int64(timestamp)] = tsBucket
	return nil
}

func loadCronOption(c *Context) (map[any]any, error) {
	raw := c.Options.GetOption("cron", nil)
	if raw == nil {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		s = fmt.Sprint(raw)
	}
	if s == "" {
		return nil, nil
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return nil, fmt.Errorf("wp: cron unserialize: %w", err)
	}
	m, ok := v.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("wp: cron: unexpected type %T", v)
	}
	return m, nil
}

func sliceToPHPArray(args []any) map[any]any {
	out := map[any]any{}
	for i, a := range args {
		out[int64(i)] = a
	}
	return out
}

func phpArrayToSlice(v any) []any {
	m, ok := v.(map[any]any)
	if !ok || len(m) == 0 {
		return nil
	}
	max := -1
	for k := range m {
		i := int(cronTimestamp(k))
		if i > max {
			max = i
		}
	}
	if max < 0 {
		return nil
	}
	out := make([]any, max+1)
	for k, val := range m {
		i := int(cronTimestamp(k))
		if i >= 0 && i < len(out) {
			out[i] = val
		}
	}
	return out
}

func cronArgsKey(args []any) string {
	// WordPress uses md5( serialize( args ) ).
	serialized, err := phpSerialize(sliceToPHPArray(args))
	if err != nil {
		sum := md5.Sum([]byte(fmt.Sprintf("%v", args)))
		return hex.EncodeToString(sum[:])
	}
	sum := md5.Sum([]byte(serialized))
	return hex.EncodeToString(sum[:])
}
