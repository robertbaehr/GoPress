# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, …), shortcodes, transients, cron stub |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies, usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass, `confirm_admin_email` stub, application passwords |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom), robots/sitemap, comments, oEmbed discovery + endpoint, 404/embed/search/date/author/tag/category archives, pagination, admin bar; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403) |
| Blocks / themes | Parser/render, core block subset, supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts, bindings, block styles/variations registries |
| REST (`/wp-json`) | Index, posts/pages/media/comments/taxonomies/users/settings/search, revisions/autosaves, menus/navigation, templates/global styles (incl. write), block-renderer, block-patterns patterns/categories, url-details (same-site), batch, site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard + list tables, options screens (general/writing/reading/discussion/media/permalink/privacy save), media library + `media-new.php` + async upload, plugins/themes/users stubs, nav menus, site editor shell, tools/import/export/privacy (export UI + confirm step), admin-ajax heartbeat |
| Plugins | MU-plugin scan, header parse, Hello Dolly shim, activate/deactivate, uninstall hook stub; PHP drop-ins detected as unsupported; Akismet not bundled (P9.8) |

## Explicitly out of scope / gone

- Multisite, XML-RPC, trackbacks, Customizer, Links Manager
- Theme/plugin install from wordpress.org, core updates, widgets REST
- Remote block/pattern directory fetches (`api.wordpress.org`)
- PHP drop-ins (`advanced-cache.php`, `object-cache.php`, `db.php`, …) — detected, never loaded
- Akismet (not shipped; activation rejected if files are dropped in)

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

## Differential harness notes

- `make diff` / `gopress-tools diff` reloads `tests/fixtures/site.sql` into `gopress_ref` and `gopress_go` unless `--skip-db`.
- After e2e uploads or other writes, restart `gopress serve` (or omit `--skip-db`) so object-cache/options match the reloaded fixture. Stale process memory can make a green corpus look red.

## Recent stack (through P10.32)

- P5.40 Block style variations from theme.json `/styles`
- P4.56 Richer REST url-details (same-site enrichment)
- P7.41 Privacy policy page setting save
- P2.36 `wp_is_numeric_array`
- P3.34 Atom comments feed
- P6.15 Password-protected post form + `postpass`
- P10.32 E2E for `/feed/rdf/`
