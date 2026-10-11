package blocks_test

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderLegacyWidgetP5131AttrPaths(t *testing.T) {
	// id path and missing idBase still empty; InnerHTML never leaks.
	cases := []string{
		`<!-- wp:legacy-widget {"id":"search-2"} /-->`,
		`<!-- wp:legacy-widget /-->`,
		`<!-- wp:legacy-widget {"idBase":""} /-->`,
		`<!-- wp:legacy-widget {"idBase":"text"} --><div class="widget">leak</div><!-- /wp:legacy-widget -->`,
	}
	for _, raw := range cases {
		out := blocks.Render(raw)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("expected empty, got %q for %s", out, raw)
		}
	}
}

func TestRenderLegacyWidgetP5131InstanceHash(t *testing.T) {
	payload := `a:1:{s:4:"text";s:4:"hell";}`
	enc := base64.StdEncoding.EncodeToString([]byte(payload))
	goodHash := md5Hex(payload)
	badHash := "deadbeefdeadbeefdeadbeefdeadbeef"

	bad := `<!-- wp:legacy-widget {"idBase":"text","instance":{"encoded":"` + enc + `","hash":"` + badHash + `"}} /-->`
	if out := blocks.Render(bad); strings.TrimSpace(out) != "" {
		t.Fatalf("bad hash without HashNonce: got %q", out)
	}

	env := &blocks.Env{
		HashNonce: func(data string) string { return md5Hex(data) },
	}
	if out := blocks.RenderWith(env, bad); strings.TrimSpace(out) != "" {
		t.Fatalf("bad hash with HashNonce: got %q", out)
	}

	good := `<!-- wp:legacy-widget {"idBase":"text","instance":{"encoded":"` + enc + `","hash":"` + goodHash + `"}} /-->`
	if out := blocks.RenderWith(env, good); strings.TrimSpace(out) != "" {
		// Widget factory still missing — valid instance still renders empty.
		t.Fatalf("valid hash still empty without widgets, got %q", out)
	}

	// No instance object → empty instance path → still empty.
	plain := `<!-- wp:legacy-widget {"idBase":"calendar"} /-->`
	if out := blocks.RenderWith(env, plain); strings.TrimSpace(out) != "" {
		t.Fatalf("plain idBase: got %q", out)
	}
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
