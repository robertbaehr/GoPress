# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_html_split`, `wp_basename`, `wp_is_uuid`, `wp_is_hex_color`/`sanitize_hex_color`, `wp_get_image_mime`, `wp_unique_filename`, `wp_ext2type`/`wp_get_ext_types`, `wp_getimagesize`, `wp_check_invalid_utf8`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_filter_object_list`, `wp_list_sort`, `wp_recursive_ksort`, `wp_array_key_first`/`wp_array_key_last`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`, `WP_Http::is_ip_address`, `wp_spaces_regexp`, …), shortcodes, transients, cron stub; `wp-emoji-settings` JSON in head (loader skipped) |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies (Remember Me → 14d + MaxAge), usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass/`reauth=1`/interim session-expired message + POST-body interim + blank-target links, `testcookie` / `wordpress_test_cookie`, `wp_lang` cookie from `?wp_lang=`, external `redirect_to` rejected, `confirm_admin_email` stub, application passwords; REST user create/update; `user-edit.php` stub; users.php bulk delete stub |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom + enclosure meta; RSS `atom:link` self from request URI; RSS channel `<image>` from site icon; feed language from locale; optional `rss_copyright` → RSS `<copyright>` / Atom `<rights>`; no PubSubHubbub hub links), robots/sitemap (`blog_public=0` → `Disallow: /` + meta noindex), comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives (+ `rel=prev/next` on paginated archives), pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header; `rel=canonical`; meta generator via `GetTheGenerator`; no `X-Pingback`; no `trackback_rdf` |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/author-name/biography/avatar/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search, site-logo/title/tagline, read-more/more/nextpage, gallery/image, audio/video synthesize + video dimensions, cover synthesize + featured image, media-text synthesize + featured image, buttons/button synthesize + empty/width polish, columns/column synthesize + stack/width/v-align, group synthesize + tagName/layout, spacer height/width polish, separator opacity/tagName polish, embed provider/type classes, social-links/social-link, footnotes polish, file synthesize + preview attrs), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages/media (incl. DELETE force)/comments/taxonomies (incl. DELETE categories/tags)/users (GET + write)/comments write/settings (richer schema)/search (+ subtype post\|page)/statuses/themes/plugins, revisions (GET + DELETE force)/autosaves (GET + POST), menus/menu-items write/menu-locations (write + DELETE unset)/navigation (incl. DELETE force), templates (incl. DELETE trash/force)/global styles (incl. write + themes/{stylesheet} + /variations), font-families (GET + POST/DELETE force)/font-faces (GET + POST/DELETE force), block-renderer, block-patterns patterns/categories, url-details (same-site), batch (207 + header envelopes + require-all-validate), site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard (+ Quick Draft creates drafts; Activity lists recent comments; Site Health Status widget) + list tables, options screens (general/writing+reading+discussion+media save via options.php/permalink save/privacy save), media library grid/list (+ bulk delete permanently) + `media-new.php` + async upload, edit-comments bulk actions, plugins/themes/users (+ bulk delete stub) + user-edit stubs, edit-tags add/delete category/tag, nav menus, site editor shell, tools/import (WXR upload stub)/export (minimal WXR download)/privacy (requests table + erase confirm UI + suggested-text updated notice + export completion email), site-health status + info stubs, admin-ajax heartbeat |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins`), Hello Dolly shim (lyric rotation + texturize + CSS), activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter), uninstall hook stub; `Requires Plugins` dependency stub (no remote API); PHP drop-ins detected as unsupported; Akismet not bundled (P9.8); Network Activate N/A on single site (P9.11) |

## Explicitly out of scope / gone

- Multisite, XML-RPC, trackbacks (`wp-trackback.php` + `trackback_rdf`), Customizer, Links Manager (`link_manager_enabled` always off)
- Theme/plugin install from wordpress.org, core updates, widgets REST
- Remote block/pattern directory fetches (`api.wordpress.org`)
- PHP drop-ins (`advanced-cache.php`, `object-cache.php`, `db.php`, …) — detected, never loaded
- Akismet (not shipped; activation rejected if files are dropped in)
- Network Activate / Network Deactivate (single-site only)
- `X-Pingback` header (XML-RPC gone)
- PubSubHubbub feed hub links (`rel=hub`)

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

## Recent stack (through P10.52)

- P10.52 E2E for menu-locations DELETE unset + STATUS/PORTING refresh
- P6.24 Remember Me longer cookie (14d + MaxAge; keep checked on error)
- P3.54 Optional feed copyright / Atom rights (`rss_copyright`)
- P2.56 `wp_getimagesize` helper
- P7.61 upload.php list bulk delete permanently
- P4.76 REST POST/DELETE `/wp/v2/font-families/{id}/font-faces`
- P5.60 Polish core/separator (opacity, tagName, align)
- P10.51 Status / PORTING refresh after P5.59–P8.15
- P8.15 Personal data export completion email (`wp_mail` stub)
- P3.53 Feed language from site locale (`BlogInfoLanguage`)
- P2.55 `wp_ext2type` / `wp_get_ext_types` helpers
- P7.60 users.php bulk delete stub (confirm + dodelete)
- P4.75 REST POST/DELETE `/wp/v2/font-families`
- P5.59 Polish core/spacer (height/width, unitless → px, className)
