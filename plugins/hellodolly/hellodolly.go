// Package hellodolly is a Go shim of wp-content/plugins/hello.php (P9.7).
package hellodolly

import (
	"fmt"
	"html"
	"math/rand"
	"strings"

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

// Plugin implements plugins.Plugin for Hello Dolly.
type Plugin struct{}

// File returns the classic WordPress plugin basename.
func (Plugin) File() string { return "hello.php" }

// Activate hooks a random lyric onto admin_notices.
func (Plugin) Activate(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		line := RandomLyric()
		htmlOut := fmt.Sprintf(
			`<p id="dolly"><span class="screen-reader-text">Quote from Hello Dolly song, by Jerry Herman: </span><span dir="ltr">%s</span></p>`,
			html.EscapeString(line),
		)
		if c != nil {
			c.AdminNotice(htmlOut)
		}
	}, 10, 0)
}

// RandomLyric returns one Hello Dolly line.
func RandomLyric() string {
	lines := strings.Split(lyrics, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	if len(cleaned) == 0 {
		return "Hello, Dolly"
	}
	return cleaned[rand.Intn(len(cleaned))]
}
