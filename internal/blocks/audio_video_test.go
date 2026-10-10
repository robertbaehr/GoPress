package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestAudioBlockSavedMarkup(t *testing.T) {
	src := `<!-- wp:audio {"id":12} -->
<figure class="wp-block-audio"><audio controls src="https://example.com/a.mp3"></audio></figure>
<!-- /wp:audio -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-audio"`) || !strings.Contains(out, `<audio`) {
		t.Fatalf("expected saved audio markup: %q", out)
	}
	if !strings.Contains(out, `https://example.com/a.mp3`) {
		t.Fatalf("missing src: %q", out)
	}
}

func TestAudioBlockSynthesizeFromAttrs(t *testing.T) {
	src := `<!-- wp:audio {"src":"https://example.com/track.mp3","autoplay":true,"loop":true,"preload":"none","caption":"Theme song"} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-audio"`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `src="https://example.com/track.mp3"`) {
		t.Fatalf("missing src: %q", out)
	}
	if !strings.Contains(out, " autoplay") || !strings.Contains(out, " loop") {
		t.Fatalf("expected autoplay/loop: %q", out)
	}
	if !strings.Contains(out, `preload="none"`) {
		t.Fatalf("expected preload: %q", out)
	}
	if !strings.Contains(out, "Theme song") {
		t.Fatalf("expected caption: %q", out)
	}
}

func TestAudioBlockFromAttachmentID(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			if id == 99 {
				return "https://example.com/uploads/99.mp3"
			}
			return ""
		},
	}
	src := `<!-- wp:audio {"id":99} /-->`
	out := blocks.RenderWith(env, src)
	if !strings.Contains(out, `src="https://example.com/uploads/99.mp3"`) {
		t.Fatalf("expected attachment URL: %q", out)
	}
}

func TestVideoBlockSynthesizeAndDimensions(t *testing.T) {
	src := `<!-- wp:video {"src":"https://example.com/clip.mp4","width":1280,"height":720,"muted":true,"poster":"https://example.com/p.jpg"} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-video"`) || !strings.Contains(out, `<video`) {
		t.Fatalf("expected video markup: %q", out)
	}
	if !strings.Contains(out, `src="https://example.com/clip.mp4"`) {
		t.Fatalf("missing src: %q", out)
	}
	if !strings.Contains(out, `width="1280"`) || !strings.Contains(out, `height="720"`) {
		t.Fatalf("expected dimensions: %q", out)
	}
	if !strings.Contains(out, `aspect-ratio: 1280 / 720;`) {
		t.Fatalf("expected aspect-ratio style: %q", out)
	}
	if !strings.Contains(out, " muted") || !strings.Contains(out, `poster="https://example.com/p.jpg"`) {
		t.Fatalf("expected muted/poster: %q", out)
	}
}

func TestVideoBlockApplyDimensionsToSaved(t *testing.T) {
	src := `<!-- wp:video {"width":640,"height":360} -->
<figure class="wp-block-video"><video controls src="https://example.com/v.mp4"></video></figure>
<!-- /wp:video -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `width="640"`) || !strings.Contains(out, `height="360"`) {
		t.Fatalf("expected injected dimensions: %q", out)
	}
	if !strings.Contains(out, `aspect-ratio: 640 / 360;`) {
		t.Fatalf("expected aspect-ratio: %q", out)
	}
}

func TestAudioVideoEmptyWithoutSrc(t *testing.T) {
	if out := blocks.Render(`<!-- wp:audio /-->`); strings.TrimSpace(out) != "" {
		t.Fatalf("empty audio should render empty: %q", out)
	}
	if out := blocks.Render(`<!-- wp:video /-->`); strings.TrimSpace(out) != "" {
		t.Fatalf("empty video should render empty: %q", out)
	}
}
