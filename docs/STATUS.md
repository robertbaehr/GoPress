# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_html_split`, `wp_basename`, `wp_is_uuid`, `wp_is_hex_color`/`sanitize_hex_color`, `wp_get_image_mime`, `wp_unique_filename`, `wp_ext2type`/`wp_get_ext_types`, `wp_getimagesize`, `wp_check_filetype`/`wp_get_mime_types`/`get_allowed_mime_types($user)`, `wp_get_upload_dir` path/url/subdir, `wp_mkdir_p`, `wp_check_invalid_utf8`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_filter_object_list`, `wp_list_sort`, `wp_recursive_ksort`, `wp_array_key_first`/`wp_array_key_last`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`, `WP_Http::is_ip_address`, `wp_spaces_regexp`, …), shortcodes, transients, cron stub; `wp-emoji-settings` JSON in head (loader skipped) |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies (Remember Me → 14d + MaxAge; HTTPS → `wordpress_sec_` + Secure flags), usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass/`reauth=1`/interim session-expired message + POST-body interim + blank-target links + interim success (transparent chrome, `login_message`, auth cookies for auth-check), `testcookie` / `wordpress_test_cookie`, `wp_lang` cookie from `?wp_lang=`, external `redirect_to` rejected, `confirm_admin_email` stub, application passwords; REST user create/update/DELETE (force + reassign); `user-new.php` POST create; `user-edit.php` stub; users.php bulk delete stub |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom + enclosure meta; item `<category>` from categories + tags; RSS/Atom `rel=self` from request URI; RSS channel `<image>` from site icon; feed language from locale; optional `rss_copyright` → RSS `<copyright>` / Atom `<rights>`; optional `rss_ttl` → RSS `<ttl>`; `lastBuildDate` from newest post modified time; RDF `admin:generatorAgent`; no PubSubHubbub hub links), robots/sitemap (`blog_public=0` → `Disallow: /` + meta noindex), comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives (+ `rel=prev/next` on paginated archives), pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header; `rel=canonical`; meta generator via `GetTheGenerator`; no `X-Pingback`; no `trackback_rdf` |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/author-name/biography/avatar/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search, site-logo/title/tagline, read-more/more/nextpage, gallery/image (+ align/dimensions/decorative/linkClass polish), audio/video synthesize + video dimensions, cover synthesize + featured image, media-text synthesize + featured image, buttons/button synthesize + empty/width polish, columns/column synthesize + stack/width/v-align, group synthesize + tagName/layout, spacer height/width polish, separator opacity/tagName polish, quote/pullquote synthesize + layout, code/preformatted/verse synthesize + align/textAlign, list/list-item `wp-block-list` + ordered attrs synthesize, heading/paragraph synthesize + class ensure, embed provider/type classes, social-links/social-link, footnotes polish, file synthesize + preview attrs), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations, patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages/media (POST create fields title/alt_text/caption/description/post + DELETE force)/comments (GET + write + DELETE trash/force)/taxonomies (incl. DELETE categories/tags)/users (GET + write + DELETE force/reassign)/settings (richer schema)/search (+ subtype post\|page)/statuses/themes (GET + status activate)/plugins (GET + status activate/deactivate), revisions (GET + DELETE force)/autosaves (GET + POST), menus/menu-items write/menu-locations (write + DELETE unset)/navigation (incl. DELETE force), templates (incl. DELETE trash/force)/global styles (incl. write + themes/{stylesheet} + /variations), font-families (GET + POST/DELETE force)/font-faces (GET + POST/DELETE force), block-renderer, block-patterns patterns/categories, url-details (same-site), batch (207 + header envelopes + require-all-validate), site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard (+ Quick Draft creates drafts; Activity lists recent comments; Site Health Status widget) + list tables (+ edit.php Quick Edit stub), options screens (general/writing+reading+discussion+media save via options.php/permalink save/privacy save), media library grid/list (+ bulk delete permanently) + `media-new.php` + async upload, edit-comments bulk actions + comment.php reply form, plugins (row + bulk activate/deactivate)/themes (Activate + `switch_theme`)/users (+ bulk delete stub) + user-new create + user-edit stubs, edit-tags add/delete category/tag, nav menus, site editor shell, tools/import (WXR upload stub)/export (minimal WXR download)/privacy (requests table + erase confirm UI + suggested-text updated notice + export completion email + erasure fulfillment email), site-health status + info stubs, admin-ajax heartbeat |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins`), Hello Dolly shim (lyric rotation + texturize + CSS), activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter, `plugin_action_links` + `plugin_row_meta` filters), uninstall hook stub; `Requires Plugins` dependency stub (no remote API); PHP drop-ins detected as unsupported; Akismet not bundled (P9.8); Network Activate N/A on single site (P9.11) |

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

## Recent stack (through P10.57)

- P10.57 STATUS/PORTING refresh after P5.65–P9.15
- P9.15 `plugin_row_meta` (+ `after_plugin_row_meta`) on plugins.php
- P3.59 Feed item categories include post tags (`get_the_category_rss`)
- P2.61 `wp_mkdir_p` recursive dir create + traversal guard
- P7.66 `edit.php` inline Quick Edit stub
- P4.81 REST media create fields (`title`/`alt_text`/`caption`/`description`/`post`)
- P5.65 Polish core/image (align, dimensions, decorative, linkClass)
- P10.56 E2E for `user-new.php` createuser + STATUS/PORTING refresh
- P6.26 HTTPS login → `wordpress_sec_` SECURE_AUTH cookie + Secure flags
- P3.58 Atom (+ comments) feed `rel=self` via `feedSelfLink`
- P2.60 `wp_get_upload_dir` Path/URL/Subdir + year/month folders
- P7.65 `comment.php` reply form (`action=replyto`)
- P4.80 REST DELETE `/wp/v2/comments/{id}` trash or force
- P5.64 Polish core/heading + core/paragraph
