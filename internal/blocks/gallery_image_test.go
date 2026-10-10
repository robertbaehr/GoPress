package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderImageRequiresImg(t *testing.T) {
	env := &blocks.Env{}
	out := env.Render(`<!-- wp:image {"id":5} --><figure class="wp-block-image"><figcaption>Only caption</figcaption></figure><!-- /wp:image -->`)
	if out != "" {
		t.Fatalf("expected empty without img, got %q", out)
	}
}

func TestRenderImageSynthesizeFromAttrs(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			if id == 5 {
				return "http://example.test/a.jpg"
			}
			return ""
		},
	}
	out := env.Render(`<!-- wp:image {"id":5,"alt":"Sample","caption":"Cap"} /-->`)
	if !strings.Contains(out, `src="http://example.test/a.jpg"`) {
		t.Fatalf("missing src: %q", out)
	}
	if !strings.Contains(out, `alt="Sample"`) {
		t.Fatalf("missing alt: %q", out)
	}
	if !strings.Contains(out, `wp-image-5`) {
		t.Fatalf("missing wp-image class: %q", out)
	}
	if !strings.Contains(out, `figcaption`) || !strings.Contains(out, "Cap") {
		t.Fatalf("missing caption: %q", out)
	}
}

func TestRenderImageDataIDAndEmptyCaption(t *testing.T) {
	env := &blocks.Env{}
	markup := `<!-- wp:image {"id":5,"data-id":"5","caption":""} -->
<figure class="wp-block-image size-large"><img src="http://example.test/a.jpg" alt=""/><figcaption></figcaption></figure>
<!-- /wp:image -->`
	out := env.Render(markup)
	if !strings.Contains(out, `data-id="5"`) {
		t.Fatalf("expected data-id: %q", out)
	}
	if strings.Contains(out, "<figcaption") {
		t.Fatalf("empty figcaption should be removed: %q", out)
	}
}

func TestRenderGalleryColumnsAndNestedDataID(t *testing.T) {
	env := &blocks.Env{}
	markup := `<!-- wp:gallery {"columns":3,"imageCrop":true} -->
<figure class="wp-block-gallery">
<!-- wp:image {"id":5} -->
<figure class="wp-block-image"><img src="http://example.test/a.jpg" alt=""/></figure>
<!-- /wp:image -->
</figure>
<!-- /wp:gallery -->`
	out := env.Render(markup)
	if !strings.Contains(out, "wp-block-gallery") || !strings.Contains(out, "has-nested-images") {
		t.Fatalf("gallery classes: %q", out)
	}
	if !strings.Contains(out, "columns-3") || !strings.Contains(out, "is-cropped") {
		t.Fatalf("columns/crop classes: %q", out)
	}
	if !strings.Contains(out, `data-id="5"`) {
		t.Fatalf("expected nested data-id from gallery backcompat: %q", out)
	}
}

func TestRenderGalleryFromIDs(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			switch id {
			case 5:
				return "http://example.test/a.jpg"
			case 6:
				return "http://example.test/b.jpg"
			default:
				return ""
			}
		},
	}
	out := env.Render(`<!-- wp:gallery {"ids":[5,6],"columns":2} /-->`)
	if !strings.Contains(out, "columns-2") || !strings.Contains(out, "is-cropped") {
		t.Fatalf("gallery wrap: %q", out)
	}
	if !strings.Contains(out, "a.jpg") || !strings.Contains(out, "b.jpg") {
		t.Fatalf("missing images: %q", out)
	}
	if !strings.Contains(out, `data-id="5"`) || !strings.Contains(out, `data-id="6"`) {
		t.Fatalf("missing data-id: %q", out)
	}
}

func TestRenderImagePolishAlignWidthTitleDecorative(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			if id == 7 {
				return "http://example.test/x.jpg"
			}
			return ""
		},
	}
	out := env.Render(`<!-- wp:image {"id":7,"align":"center","width":"320","height":"200","aspectRatio":"16/9","scale":"cover","title":"Hero","isDecorative":true,"linkDestination":"media","linkClass":"custom-link"} /-->`)
	for _, want := range []string{
		`wp-block-image`,
		`aligncenter`,
		`is-resized`,
		`src="http://example.test/x.jpg"`,
		`alt=""`,
		`role="presentation"`,
		`style="`,
		`width:320px`,
		`height:200px`,
		`aspect-ratio:16/9`,
		`object-fit:cover`,
		`class="custom-link"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, `title="Hero"`) {
		t.Fatalf("decorative image should omit title: %q", out)
	}
}

func TestRenderImageEnsureClassOnSaved(t *testing.T) {
	env := &blocks.Env{}
	markup := `<!-- wp:image {"align":"wide"} -->
<figure class="size-large"><img src="http://example.test/a.jpg" alt="A"/></figure>
<!-- /wp:image -->`
	out := env.Render(markup)
	if !strings.Contains(out, "wp-block-image") {
		t.Fatalf("expected wp-block-image: %q", out)
	}
	if !strings.Contains(out, "alignwide") {
		t.Fatalf("expected alignwide: %q", out)
	}
}
