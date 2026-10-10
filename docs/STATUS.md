# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_check_invalid_utf8`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_filter_object_list`, `wp_list_sort`, `wp_recursive_ksort`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`, …), shortcodes, transients, cron stub |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies, usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass/`reauth=1`/interim session-expired message, `confirm_admin_email` stub, application passwords; REST user create/update; `user-edit.php` stub |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom), robots/sitemap, comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives (+ `rel=prev/next` on paginated archives), pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header; no `X-Pingback` |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/author-name/biography/avatar/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search, site-logo/title/tagline, read-more/more/nextpage, gallery/image, embed provider/type classes), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages/media/comments/taxonomies (incl. DELETE categories/tags)/users (GET + write)/comments write/settings (richer schema)/search/statuses/themes/plugins, revisions/autosaves, menus/menu-items write/navigation, templates/global styles (incl. write), block-renderer, block-patterns patterns/categories, url-details (same-site), batch, site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard + list tables, options screens (general/writing+reading+discussion save via options.php/media/permalink/privacy save), media library grid/list + `media-new.php` + async upload, edit-comments bulk actions, plugins/themes/users + user-edit stubs, nav menus, site editor shell, tools/import/export/privacy (requests table + confirm), site-health status + info stubs, admin-ajax heartbeat |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins`), Hello Dolly shim, activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter), uninstall hook stub; PHP drop-ins detected as unsupported; Akismet not bundled (P9.8) |

## Explicitly out of scope / gone

- Multisite, XML-RPC, trackbacks, Customizer, Links Manager
- Theme/plugin install from wordpress.org, core updates, widgets REST
- Remote block/pattern directory fetches (`api.wordpress.org`)
- PHP drop-ins (`advanced-cache.php`, `object-cache.php`, `db.php`, …) — detected, never loaded
- Akismet (not shipped; activation rejected if files are dropped in)
- `X-Pingback` header (XML-RPC gone)

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

## Recent stack (through P10.41)

- P10.41 Status / PORTING refresh
- P6.19 Interim login session-expired `login-message`
- P3.43 Archive `rel=prev` / `rel=next` pagination links
- P2.45 `wp_recursive_ksort`
- P7.50 POST save discussion options via `options.php`
- P4.65 REST DELETE `/wp/v2/categories/{id}` and `/wp/v2/tags/{id}` (`force=true`)
- P5.49 core/embed provider/type/align class improvements
- P10.40 E2E for `wp-login.php?reauth=1` (+ STATUS/PORTING refresh)
- P8.12 Privacy policy guide content registry (`privacy_policy_guide`)
- P3.42 `rel=canonical` / `wp_get_canonical_url`
- P2.44 `wp_list_sort`
- P7.49 POST save reading options via `options.php`
- P4.64 REST PATCH/PUT `/wp/v2/comments/{id}`
- P5.48 core/gallery + core/image render improvements
- P10.39 E2E for POST `/wp/v2/users` (+ STATUS/PORTING refresh)
- P6.18 `reauth=1` forces login form (clears cookies; skips logged-in redirect)
- P3.41 Omit `X-Pingback` on singular front responses
- P2.43 `wp_filter_object_list` / `wp_list_filter`
- P7.48 POST save writing options via `options.php`
- P4.63 Richer REST `/wp/v2/settings` schema
- P5.47 core/read-more + more + nextpage
