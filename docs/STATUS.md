# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`, …), shortcodes, transients, cron stub |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies, usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass, `confirm_admin_email` stub, application passwords |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom), robots/sitemap, comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives, pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403) |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages/media/comments/taxonomies/users/settings/search/statuses/themes/plugins, revisions/autosaves, menus/menu-items write/navigation, templates/global styles (incl. write), block-renderer, block-patterns patterns/categories, url-details (same-site), batch, site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard + list tables, options screens (general/writing/reading/discussion/media/permalink/privacy save), media library grid/list + `media-new.php` + async upload, edit-comments bulk actions, plugins/themes/users stubs, nav menus, site editor shell, tools/import/export/privacy (requests table + confirm), site-health status + info stubs, admin-ajax heartbeat |
| Plugins | MU-plugin scan, header parse, Hello Dolly shim, activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter), uninstall hook stub; PHP drop-ins detected as unsupported; Akismet not bundled (P9.8) |

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

## Recent stack (through P3.39)

- P3.39 alternate feed link tags in head (`feed_links_extra` author/search)
- P2.41 `wp_normalize_path` / `wp_is_stream`
- P7.46 `update.php` 403 (plugin/theme upgrades gone)
- P4.61 GET `/wp/v2/block-types/{namespace}/{name}` (+ namespace list)
- P5.45 core/site-logo + site-title + site-tagline renderers
- P10.36 E2E for `home_url` links + absolute canonical Location
- P6.17 Interim login success Continue + top-level redirect_to
- P3.38 Strengthen trailing-slash redirects (absolute Location, /page/N)
- P2.40 `wp_parse_url` (protocol-relative + path-only)
- P7.45 Site Health Status/Info tab chrome polish
- P4.60 GET `/wp/v2/plugins` (+ plugin item) for installed plugins
- P5.44 query-pagination / previous / next / numbers (paged + max pages)
- P10.35 Status / PORTING refresh
- P9.9 Plugin activate/deactivate hooks + `active_plugins` filter
- P3.37 `home_url` / `site_url` / `set_url_scheme` / `admin_url`
- P2.39 `wp_unique_id` / `wp_unique_prefixed_id`
- P7.44 Site Health Info tab (`site-health-info.php`); Customizer already gone (P7.6)
- P4.59 GET `/wp/v2/themes` (+ stylesheet item)
- P5.43 post-terms / post-date / post-author / post-title / post-excerpt / post-content renderers
- P10.34 E2E for theme `search.html` (`/?s=`)
- P6.16 Logout honors `redirect_to` (default `loggedout=true`)
- P3.36 Prefer theme `search.html` (+ header/footer, curly query-title)
- P2.38 `wp_array_slice_assoc`
- P7.43 edit-comments.php bulk actions
- P4.58 GET `/wp/v2/statuses`
- P5.42 calendar / archives / latest-posts / latest-comments / tag-cloud / search block renderers
