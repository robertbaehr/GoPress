// Boot (P1.5) — minimal runtime bootstrap (subset of wp-settings.php).
package wp

import (
	"context"
	"fmt"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
)

// Boot opens the database and wires Hooks, Cache, Options, and Meta on a Context.
// Autoloaded options are loaded into the object cache (alloptions).
func Boot(ctx context.Context, cfg *config.Config) (*Context, error) {
	if cfg == nil {
		return nil, fmt.Errorf("wp: Boot: nil config")
	}
	c := NewContext(ctx)
	c.Config = cfg

	handle, err := db.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("wp: Boot: %w", err)
	}
	c.DB = handle

	c.Hooks = NewHooks()
	c.Cache = NewObjectCache()
	c.Options = NewOptions(handle, c.Cache)
	c.Meta = NewMeta(handle, c.Cache)

	if err := c.Options.LoadAutoload(); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("wp: Boot: load autoload options: %w", err)
	}
	return c, nil
}
