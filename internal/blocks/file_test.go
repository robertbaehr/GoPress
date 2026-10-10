package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestFileBlockSavedMarkup(t *testing.T) {
	src := `<!-- wp:file {"href":"https://example.com/doc.pdf"} -->
<div class="wp-block-file"><a href="https://example.com/doc.pdf">doc.pdf</a><a href="https://example.com/doc.pdf" class="wp-block-file__button" download>Download</a></div>
<!-- /wp:file -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-file"`) || !strings.Contains(out, "doc.pdf") {
		t.Fatalf("expected saved file markup: %q", out)
	}
}

func TestFileBlockSynthesizeFromAttrs(t *testing.T) {
	src := `<!-- wp:file {"href":"https://example.com/report.pdf","fileName":"Quarterly Report","showDownloadButton":true} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `href="https://example.com/report.pdf"`) {
		t.Fatalf("missing href: %q", out)
	}
	if !strings.Contains(out, "Quarterly Report") {
		t.Fatalf("missing fileName: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-file__button"`) || !strings.Contains(out, "download") {
		t.Fatalf("expected download button: %q", out)
	}
}

func TestFileBlockHideDownloadButton(t *testing.T) {
	src := `<!-- wp:file {"href":"https://example.com/a.pdf","fileName":"A","showDownloadButton":false} /-->`
	out := blocks.Render(src)
	if strings.Contains(out, "wp-block-file__button") {
		t.Fatalf("download button should be omitted: %q", out)
	}
	if !strings.Contains(out, ">A</a>") {
		t.Fatalf("expected text link: %q", out)
	}
}

func TestFileBlockDisplayPreviewAttrs(t *testing.T) {
	src := `<!-- wp:file {"href":"https://example.com/a.pdf","displayPreview":true,"previewHeight":480} -->
<div class="wp-block-file"><object class="wp-block-file__embed" data="https://example.com/a.pdf" type="application/pdf" style="height:480px" aria-label="notes.pdf"></object><a id="wp-block-file--media-1" href="https://example.com/a.pdf">notes.pdf</a></div>
<!-- /wp:file -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `data-wp-interactive="core/file"`) {
		t.Fatalf("expected interactive directive: %q", out)
	}
	if !strings.Contains(out, `data-wp-bind--hidden="!state.hasPdfPreview"`) {
		t.Fatalf("expected bind hidden: %q", out)
	}
	if !strings.Contains(out, `hidden="true"`) {
		t.Fatalf("expected hidden attr: %q", out)
	}
	if !strings.Contains(out, `aria-label="Embed of notes.pdf."`) {
		t.Fatalf("expected embed aria-label: %q", out)
	}
}

func TestFileBlockSynthesizeWithPreview(t *testing.T) {
	src := `<!-- wp:file {"href":"https://example.com/a.pdf","fileName":"a.pdf","displayPreview":true} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `<object`) || !strings.Contains(out, `data-wp-interactive="core/file"`) {
		t.Fatalf("expected preview object + interactive: %q", out)
	}
}
