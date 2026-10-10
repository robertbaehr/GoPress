package wp

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
)

// SwitchTheme activates stylesheet (switch_theme subset, P7.62).
// Updates template, stylesheet, and current_theme options.
func SwitchTheme(c *Context, absPath, stylesheet string) error {
	stylesheet = strings.TrimSpace(stylesheet)
	if c == nil || c.Options == nil {
		return fmt.Errorf("theme: invalid context")
	}
	if stylesheet == "" {
		return fmt.Errorf("theme: empty stylesheet")
	}
	if strings.Contains(stylesheet, "..") || strings.ContainsAny(stylesheet, `/\`) {
		return fmt.Errorf("theme: invalid stylesheet")
	}
	themes, err := theme.ListInstalledThemes(absPath)
	if err != nil {
		return err
	}
	var found *theme.ThemeData
	for i := range themes {
		if themes[i].Slug == stylesheet {
			found = &themes[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("theme: %q does not exist", stylesheet)
	}
	template := found.Template
	if template == "" {
		template = stylesheet
	}
	if _, err := c.Options.UpdateOption("template", template, "yes"); err != nil {
		return err
	}
	if _, err := c.Options.UpdateOption("stylesheet", stylesheet, "yes"); err != nil {
		return err
	}
	name := found.Name
	if name == "" {
		name = stylesheet
	}
	if _, err := c.Options.UpdateOption("current_theme", name, "yes"); err != nil {
		return err
	}
	if c.Hooks != nil {
		c.Hooks.DoAction("switch_theme", name, *found)
		c.Hooks.DoAction("after_switch_theme", name, *found)
	}
	return nil
}
