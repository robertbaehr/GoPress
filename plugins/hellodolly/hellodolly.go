// Package hellodolly is a Go shim of wp-content/plugins/hello.php (P9.7 / P9.12).
package hellodolly

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// lyrics are the Hello Dolly lines (same source as hello.php).
const lyrics = `Hello, Dolly
Well, hello, Dolly
It's so nice to have you back where you belong
You're lookin' swell, Dolly
I can tell, Dolly
You're still glowin', you're still crowin'
You're still goin' strong
I feel the room swayin'
While the band's playin'
One of our old favorite songs from way back when
So, take her wrap, fellas
Dolly, never go away again
Hello, Dolly
Well, hello, Dolly
It's so nice to have you back where you belong
You're lookin' swell, Dolly
I can tell, Dolly
You're still glowin', you're still crowin'
You're still goin' strong
I feel the room swayin'
While the band's playin'
One of our old favorite songs from way back when
So, golly, gee, fellas
Have a little faith in me, fellas
Dolly, never go away
Promise, you'll never go away
Dolly'll never go away again`

// lyricCursor advances so successive admin screens rotate through lines (P9.12).
var lyricCursor atomic.Uint64

// Plugin implements plugins.Plugin for Hello Dolly.
type Plugin struct{}

// File returns the classic WordPress plugin basename.
func (Plugin) File() string { return "hello.php" }

// Activate hooks lyric rotation onto admin_notices and CSS onto admin_head (P9.12).
func (Plugin) Activate(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		line := ChosenLyric(c)
		lang := ""
		if !localeIsEnglish(c) {
			lang = ` lang="en"`
		}
		htmlOut := fmt.Sprintf(
			`<p id="dolly"><span class="screen-reader-text">Quote from Hello Dolly song, by Jerry Herman: </span><span dir="ltr"%s>%s</span></p>`,
			lang,
			line,
		)
		if c != nil {
			c.AdminNotice(htmlOut)
		}
	}, 10, 0)
	c.Hooks.AddAction("admin_head", func(args ...any) {
		if c != nil && c.Hooks != nil {
			c.Hooks.WriteNotice(dollyCSS)
		}
	}, 10, 0)
}

const dollyCSS = `<style type="text/css" id="dolly-css">
#dolly{float:right;padding:5px 10px;margin:0;font-size:12px;line-height:1.6666}
.rtl #dolly{float:left}
.block-editor-page #dolly{display:none}
@media screen and (max-width:782px){#dolly,.rtl #dolly{float:none;padding-left:0;padding-right:0}}
</style>`

// ChosenLyric returns the next Hello Dolly line, texturized like WP (P9.12).
func ChosenLyric(c *wp.Context) string {
	_ = c
	return wp.Texturize(nextLyric())
}

// RandomLyric returns one Hello Dolly line (texturized). Kept for tests/callers.
func RandomLyric() string {
	return wp.Texturize(nextLyric())
}

func nextLyric() string {
	lines := lyricLines()
	if len(lines) == 0 {
		return "Hello, Dolly"
	}
	// Rotate through the lyric list (with a random offset) so each admin_notices
	// call surfaces a different line — WP uses mt_rand each request.
	n := lyricCursor.Add(1)
	off := uint64(rand.Intn(len(lines)))
	return lines[int((n+off)%uint64(len(lines)))]
}

func lyricLines() []string {
	raw := strings.Split(lyrics, "\n")
	var cleaned []string
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return cleaned
}

func localeIsEnglish(c *wp.Context) bool {
	if c == nil || c.Options == nil {
		return true
	}
	loc := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("WPLANG", "")))
	if loc == "" || loc == "<nil>" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(loc), "en_") || strings.EqualFold(loc, "en")
}

// LyricCount reports how many lyric lines are available (tests).
func LyricCount() int { return len(lyricLines()) }
