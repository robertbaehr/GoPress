// Cron stubs (P3.8) — schedule/spawn events into the cron option; wp-cron.php returns 200.
package wp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// Known recurrence intervals (wp_get_schedules subset).
var cronIntervals = map[string]int64{
	"hourly":     3600,
	"twicedaily": 43200,
	"daily":      86400,
	"weekly":     604800,
}

// ScheduleEvent adds a recurring cron hook to the cron option (wp_schedule_event stub).
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

// ScheduleSingleEvent adds a one-shot cron hook (wp_schedule_single_event stub).
func ScheduleSingleEvent(c *Context, timestamp int64, hook string, args []any) error {
	return putCronEvent(c, timestamp, hook, args, "", 0)
}

// SpawnCron is a stub for spawn_cron — records that a spawn was requested.
// It does not open an HTTP connection; /wp-cron.php is expected to be hit separately.
func SpawnCron(c *Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	if c.Config != nil && c.Config.DisableWPCron {
		return false
	}
	// Touch doing_cron lock option like WP's transient; stub stores a timestamp string.
	_, _ = c.Options.UpdateOption("doing_cron", fmt.Sprintf("%d", time.Now().Unix()), "no")
	return true
}

// ServeCron handles GET/POST /wp-cron.php — always 200 for the stub (jobs not executed).
func ServeCron(w http.ResponseWriter, r *http.Request, c *Context) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Expires", "Wed, 11 Jan 1984 05:00:00 GMT")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate, max-age=0")
	w.WriteHeader(http.StatusOK)
	// Empty body matches a quiet cron runner.
	_, _ = w.Write([]byte{})
	_ = r
	_ = c
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
		// PHP may have stored int keys that unserialize as int64.
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

	serialized, err := phpSerialize(cron)
	if err != nil {
		return fmt.Errorf("wp: cron serialize: %w", err)
	}
	_, err = c.Options.UpdateOption("cron", serialized, "")
	return err
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
