# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting, shortcodes, transients, cron stub |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies, usermeta `session_tokens`, nonces, login/logout/lost-password, application passwords |
| Front | Home + singular + attachment HTML, rewrite, feeds (RSS/Atom), robots/sitemap, comments, oEmbed discovery + endpoint, 404/embed templates, admin bar |
| Blocks / themes | Parser/render, core block subset, supports, style engine, theme.json, patterns, template hierarchy/parts, bindings, block styles registry |
| REST (`/wp-json`) | Index, posts/pages/media/comments/taxonomies/users/settings/search, revisions/autosaves, menus/navigation, templates/global styles, batch, site-health stubs; remote directory proxies blocked |
| Admin | Dashboard + list tables, options screens, media library + `media-new.php` + async upload, plugins/themes/users stubs, nav menus, site editor shell, tools/import/export/privacy, admin-ajax heartbeat |
| Plugins | MU-plugin scan, header parse, Hello Dolly shim; PHP drop-ins detected as unsupported |

## Explicitly out of scope / gone

- Multisite, XML-RPC, trackbacks, Customizer, Links Manager
- Theme/plugin install from wordpress.org, core updates, widgets REST
- Remote block/pattern directory fetches (`api.wordpress.org`)
- PHP drop-ins (`advanced-cache.php`, `object-cache.php`, `db.php`, …) — detected, never loaded

## Remaining (high level)

- Deeper block editor / site editor parity (full Gutenberg UI, more block types)
- Broader REST write coverage and corpus climb on allowlisted diffs
- Richer admin screens (post editor beyond shell, full list-table actions)
- Plugin runtime beyond Go shims; theme PHP execution
- Mail/HTTP/cron production behavior; image editor edge cases
- Full `PORTING.md` `todo`/`partial` rows → `done`

## How to refresh

```bash
./bin/gopress-tools porting-status   # rewrites PORTING.md
make check
```
