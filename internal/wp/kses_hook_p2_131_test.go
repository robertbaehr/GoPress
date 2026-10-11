package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.131: KsesHook applies pre_kses; KsesFiltered routes through it.
func TestKsesHookPreKsesP2131(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("pre_kses", func(value any, args ...any) any {
		s, _ := value.(string)
		return "HOOKED:" + s
	}, 10, 3)

	allowed := wp.AllowedHTML{"p": {"class": true}}
	got := wp.KsesHook(c, "<p>hi</p>", allowed, nil)
	if got != "HOOKED:<p>hi</p>" {
		t.Fatalf("KsesHook=%q", got)
	}
	if wp.KsesHook(nil, "x", allowed, nil) != "x" {
		t.Fatal("nil context should pass through")
	}

	out := wp.KsesFiltered(c, "<p>ok</p><script>x</script>", allowed, nil)
	if !strings.HasPrefix(out, "HOOKED:") {
		t.Fatalf("KsesFiltered should run KsesHook first: %q", out)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("script should be stripped after hook: %q", out)
	}
}
