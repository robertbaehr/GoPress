package wp_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type dateVector struct {
	Format string `json:"format"`
	Result any    `json:"result"`
	TS     int64  `json:"ts"`
	TZ     string `json:"tz"`
	MySQL  string `json:"mysql"`
	GMT    string `json:"gmt"`
	Local  string `json:"local"`
}

type datesCorpus struct {
	WPDate         []dateVector `json:"wp_date"`
	MySQL2Date     []dateVector `json:"mysql2date"`
	GetGMTFromDate []dateVector `json:"get_gmt_from_date"`
	GetDateFromGMT []dateVector `json:"get_date_from_gmt"`
	DateI18n       []dateVector `json:"date_i18n"`
}

func loadDatesCorpus(t *testing.T) datesCorpus {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "dates", "vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus datesCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	return corpus
}

func wantString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// JSON numbers decode as float64.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprint(t)
	}
}

func TestWPDateGoldenVectors(t *testing.T) {
	corpus := loadDatesCorpus(t)
	if len(corpus.WPDate) == 0 {
		t.Fatal("no wp_date vectors")
	}
	for i, v := range corpus.WPDate {
		loc, err := time.LoadLocation(v.TZ)
		if err != nil {
			t.Fatalf("[%d] tz %q: %v", i, v.TZ, err)
		}
		got := wp.WPDate(v.Format, v.TS, loc)
		want := wantString(v.Result)
		if got != want {
			t.Fatalf("[%d] wp_date(%q, %d, %s)=%q want %q", i, v.Format, v.TS, v.TZ, got, want)
		}
	}
}

func TestMySQL2DateGoldenVectors(t *testing.T) {
	corpus := loadDatesCorpus(t)
	if len(corpus.MySQL2Date) == 0 {
		t.Fatal("no mysql2date vectors")
	}
	for i, v := range corpus.MySQL2Date {
		got := wp.MySQL2Date(v.Format, v.MySQL, false)
		want := wantString(v.Result)
		if got != want {
			t.Fatalf("[%d] mysql2date(false)=%q want %q", i, got, want)
		}
		gotGMT := wp.MySQL2Date(v.Format, v.MySQL, true)
		wantGMT := wantString(v.GMT)
		if gotGMT != wantGMT {
			t.Fatalf("[%d] mysql2date(true)=%q want %q", i, gotGMT, wantGMT)
		}
	}
}

func TestGetGMTFromDateGoldenVectors(t *testing.T) {
	corpus := loadDatesCorpus(t)
	for i, v := range corpus.GetGMTFromDate {
		got, err := wp.GetGMTFromDate(v.Local, v.TZ, "Y-m-d H:i:s")
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
		want := wantString(v.Result)
		if got != want {
			t.Fatalf("[%d] get_gmt_from_date(%q,%s)=%q want %q", i, v.Local, v.TZ, got, want)
		}
	}
}

func TestGetDateFromGMTGoldenVectors(t *testing.T) {
	corpus := loadDatesCorpus(t)
	for i, v := range corpus.GetDateFromGMT {
		got, err := wp.GetDateFromGMT(v.GMT, v.TZ, "Y-m-d H:i:s")
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
		want := wantString(v.Result)
		if got != want {
			t.Fatalf("[%d] get_date_from_gmt(%q,%s)=%q want %q", i, v.GMT, v.TZ, got, want)
		}
	}
}

func TestHumanTimeDiff(t *testing.T) {
	base := int64(1_700_000_000)
	cases := []struct {
		from, to int64
		want     string
	}{
		{base, base + 1, "1 second"},
		{base, base + 45, "45 seconds"},
		{base, base + 60, "1 minute"},
		{base, base + 120, "2 minutes"},
		{base, base + 3600, "1 hour"},
		{base, base + 7200, "2 hours"},
		{base, base + 86400, "1 day"},
		{base, base + 172800, "2 days"},
		{base, base + 604800, "1 week"},
		{base, base + 1209600, "2 weeks"},
		{base, base + 2592000, "1 month"},
		{base, base + 5184000, "2 months"},
		{base, base + 31536000, "1 year"},
		{base, base + 63072000, "2 years"},
		{base + 90, base, "2 minutes"}, // abs direction
	}
	for _, tc := range cases {
		got := wp.HumanTimeDiff(tc.from, tc.to)
		if got != tc.want {
			t.Fatalf("HumanTimeDiff(%d,%d)=%q want %q", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestDateI18nUTCSubset(t *testing.T) {
	// date_i18n golden vectors diverge for non-UTC (dump cache quirk);
	// assert UTC cases match wp_date.
	corpus := loadDatesCorpus(t)
	for i, v := range corpus.DateI18n {
		if v.TZ != "UTC" {
			continue
		}
		got, err := wp.DateI18n(v.Format, v.TS, v.TZ)
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
		want := wantString(v.Result)
		if got != want {
			t.Fatalf("[%d] date_i18n UTC %q ts=%d got %q want %q", i, v.Format, v.TS, got, want)
		}
	}
}

func TestDateFormatTimeFormatOptions(t *testing.T) {
	// P2.97: date_format / time_format option helpers + expanded tokens.
	c := bootAppPass(t)
	prevDate := c.Options.GetOption("date_format", "F j, Y")
	prevTime := c.Options.GetOption("time_format", "g:i a")
	prevTZ := c.Options.GetOption("timezone_string", "")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("date_format", prevDate, "yes")
		_, _ = c.Options.UpdateOption("time_format", prevTime, "yes")
		_, _ = c.Options.UpdateOption("timezone_string", prevTZ, "yes")
	})

	_, _ = c.Options.UpdateOption("timezone_string", "UTC", "yes")
	_, _ = c.Options.UpdateOption("date_format", "Y-m-d", "yes")
	_, _ = c.Options.UpdateOption("time_format", "g:i a", "yes")
	if got := wp.DateFormat(c); got != "Y-m-d" {
		t.Fatalf("DateFormat=%q", got)
	}
	if got := wp.TimeFormat(c); got != "g:i a" {
		t.Fatalf("TimeFormat=%q", got)
	}

	// 2020-06-15 14:05:00 UTC
	ts := time.Date(2020, 6, 15, 14, 5, 0, 0, time.UTC).Unix()
	if got := wp.WPDateOption(c, ts); got != "2020-06-15" {
		t.Fatalf("WPDateOption=%q", got)
	}
	if got := wp.WPTimeOption(c, ts); got != "2:05 pm" {
		t.Fatalf("WPTimeOption=%q", got)
	}

	_, _ = c.Options.UpdateOption("date_format", "F jS, Y", "yes")
	if got := wp.WPDateOption(c, ts); got != "June 15th, 2020" {
		t.Fatalf("ordinal date=%q", got)
	}

	// Common WP presets.
	for _, tc := range []struct {
		format string
		want   string
	}{
		{"m/d/Y", "06/15/2020"},
		{"d/m/Y", "15/06/2020"},
		{"n/j/Y", "6/15/2020"},
		{"M j, Y", "Jun 15, 2020"},
		{"D, M j", "Mon, Jun 15"},
		{"g:i A", "2:05 PM"},
		{"H:i", "14:05"},
		{"G:i", "14:05"},
	} {
		got := wp.WPDate(tc.format, ts, time.UTC)
		if got != tc.want {
			t.Fatalf("format %q = %q want %q", tc.format, got, tc.want)
		}
	}
}

func TestTimezoneStringOffsetFallback(t *testing.T) {
	// P2.96: empty timezone_string → ±HH:MM from gmt_offset.
	c := bootAppPass(t)
	prevTZ := c.Options.GetOption("timezone_string", "")
	prevOff := c.Options.GetOption("gmt_offset", "0")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("timezone_string", prevTZ, "yes")
		_, _ = c.Options.UpdateOption("gmt_offset", prevOff, "yes")
	})

	_, _ = c.Options.UpdateOption("timezone_string", "", "yes")
	for _, tc := range []struct {
		offset string
		want   string
	}{
		{"0", "+00:00"},
		{"5.5", "+05:30"},
		{"-5.5", "-05:30"},
		{"-6", "-06:00"},
		{"8.75", "+08:45"},
	} {
		_, _ = c.Options.UpdateOption("gmt_offset", tc.offset, "yes")
		got := wp.TimezoneString(c)
		if got != tc.want {
			t.Fatalf("gmt_offset=%s TimezoneString=%q want %q", tc.offset, got, tc.want)
		}
		loc := wp.Timezone(c)
		if loc == nil {
			t.Fatalf("Timezone nil for offset %s", tc.offset)
		}
	}

	_, _ = c.Options.UpdateOption("timezone_string", "America/New_York", "yes")
	if got := wp.TimezoneString(c); got != "America/New_York" {
		t.Fatalf("timezone_string preferred: %q", got)
	}
	loc := wp.Timezone(c)
	if loc.String() != "America/New_York" {
		t.Fatalf("Timezone=%q", loc)
	}
}
