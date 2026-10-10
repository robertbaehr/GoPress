# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_check_invalid_utf8`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`, …), shortcodes, transients, cron stub |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies, usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass, `confirm_admin_email` stub, application passwords; REST user create/update; `user-edit.php` stub |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom), robots/sitemap, comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives, pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/author-name/biography/avatar/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search, site-logo/title/tagline), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages/media/comments/taxonomies/users (GET + write)/settings/search/statuses/themes/plugins, revisions/autosaves, menus/menu-items write/navigation, templates/global styles (incl. write), block-renderer, block-patterns patterns/categories, url-details (same-site), batch, site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard + list tables, options screens (general/writing/reading/discussion/media/permalink/privacy save), media library grid/list + `media-new.php` + async upload, edit-comments bulk actions, plugins/themes/users + user-edit stubs, nav menus, site editor shell, tools/import/export/privacy (requests table + confirm), site-health status + info stubs, admin-ajax heartbeat |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins`), Hello Dolly shim, activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter), uninstall hook stub; PHP drop-ins detected as unsupported; Akismet not bundled (P9.8) |

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

## Recent stack (through P10.38)

- P10.38 Status / PORTING refresh
- P9.10 `get_mu_plugins` header parse for top-level must-use plugins
- P3.40 `wp_get_shortlink` + head/`Link` `rel=shortlink` (`?p=`)
- P2.42 `wp_check_invalid_utf8` (strip → U+FFFD)
- P7.47 `user-edit.php` stub (edit_user + Users list links)
- P4.62 REST POST/PUT `/wp/v2/users` (`create_users` / `edit_user`)
- P5.46 core/avatar + post-author-name + post-author-biography
- P10.37 E2E for GET `/wp/v2/plugins` (+ Hello Dolly item)
- P8.11 Privacy policy default content (`get_default_content`) on Policy Guide
- P3.39 alternate feed link tags in head (`feed_links_extra` author/search)
- P2.41 `wp_normalize_path` / `wp_is_stream`
- P7.46 `update.php` 403 (plugin/theme upgrades gone)
- P4.61 GET `/wp/v2/block-types/{namespace}/{name}` (+ namespace list)
- P5.45 core/site-logo + site-title + site-tagline renderers
