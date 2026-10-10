// Boot (P1.5) — minimal runtime bootstrap (subset of wp-settings.php).
package wp

import (
	"context"
	"fmt"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/theme"
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
	registerDefaultCacheGroups(c.Cache)
	c.Options = NewOptions(handle, c.Cache)
	c.Meta = NewMeta(handle, c.Cache)
	c.Locale = NewLocale("")
	c.Auth = NewAuth(cfg)
	c.Users = NewUsers(handle, c.Cache)
	c.Roles = NewRoles(c.Options)
	c.Mailer = NewMailer()

	if err := c.Options.LoadAutoload(); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("wp: Boot: load autoload options: %w", err)
	}
	if err := c.Roles.Load(); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("wp: Boot: load roles: %w", err)
	}

	// Active theme.json (P5.4). Missing theme.json is non-fatal (classic themes).
	stylesheet, _ := c.Options.GetOption("stylesheet", "").(string)
	if stylesheet == "" {
		stylesheet = "twentytwentytwo"
	}
	if tj, err := theme.LoadActive(cfg.ABSPATH, stylesheet); err == nil {
		c.Theme = tj
	}

	// Block patterns from theme + wp_block posts (P5.9). Non-fatal on load errors.
	c.Patterns = NewPatternRegistry()
	_ = LoadPatterns(c)

	// Block pattern categories (P5.38).
	c.PatternCategories = NewPatternCategoryRegistry()
	RegisterDefaultPatternCategories(c.PatternCategories, stylesheet)

	// Block styles registry (P5.26).
	c.BlockStyles = NewBlockStyleRegistry()
	RegisterDefaultBlockStyles(c.BlockStyles)

	// Block variations registry (P5.37).
	c.BlockVariations = NewBlockVariationRegistry()
	RegisterDefaultBlockVariations(c.BlockVariations)

	// Block bindings sources registry (P5.39).
	c.BlockBindings = NewBlockBindingsRegistry()
	RegisterDefaultBlockBindingsSources(c.BlockBindings, stylesheet)

	// Font collections library (P4.55) — metadata only; remote families skipped.
	c.FontLibrary = NewFontLibrary()
	RegisterDefaultFontCollections(c.FontLibrary)

	// Shortcodes: gallery, caption, embed (P2.11).
	c.Shortcodes = NewShortcodes()
	RegisterDefaultShortcodes(c.Shortcodes)

	return c, nil
}
