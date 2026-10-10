# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_html_split`, `wp_basename`, `wp_is_uuid`, `wp_is_hex_color`/`sanitize_hex_color`, `wp_get_image_mime`, `wp_unique_filename`, `wp_ext2type`/`wp_get_ext_types`, `wp_getimagesize`, `wp_check_filetype`/`wp_get_mime_types`/`get_allowed_mime_types($user)`, `wp_get_upload_dir` path/url/subdir, `wp_mkdir_p`, `wp_tempnam`/`get_temp_dir`, `wp_is_ini_value_changeable`, `wp_is_json_request`/`wp_is_json_media_type`, `wp_is_rest_endpoint`/`wp_is_serving_rest_request`, `wp_doing_ajax`, `wp_doing_cron`, `is_login`, `wp_get_environment_type`, `wp_get_development_mode`/`wp_is_development_mode`, `get_current_user_id`/`wp_get_current_user`, `wp_get_session_token`, `wp_get_raw_referer`, `wp_get_referer`/`wp_validate_redirect`, `get_user_option`/`update_user_option`, `wp_check_invalid_utf8`, `wp_validate_boolean`, `wp_slash`, `wp_is_numeric_array`, `wp_list_pluck`, `wp_filter_object_list`, `wp_list_sort`, `wp_recursive_ksort`, `wp_array_key_first`/`wp_array_key_last`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`/`wp_is_stream`, `WP_Http::is_ip_address`, `wp_spaces_regexp`, …), shortcodes, transients, cron stub (`DOING_CRON` on `/wp-cron.php`); `wp-emoji-settings` JSON in head (loader skipped) |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies (Remember Me → 14d + MaxAge; HTTPS → `wordpress_sec_` + Secure flags), usermeta `session_tokens`, nonces, login/logout/lost-password/resetpass/postpass/`reauth=1`/interim session-expired message (`login_messages`) + `aria-describedby` focus association + POST-body interim + blank-target links + interim success (transparent chrome, `login_message`, auth cookies for auth-check, empty-title `&lsaquo;` document title), `testcookie` / `wordpress_test_cookie`, `wp_lang` cookie from `?wp_lang=` + language switcher preserves POST `redirect_to`/`action`/`interim-login`, external `redirect_to` rejected, `confirm_admin_email` stub, application passwords; REST user create/update/DELETE (force + reassign); `user-new.php` POST create; `user-edit.php` stub; users.php bulk delete stub; login `nocache_headers` + `login_error` shake (`wp_shake_js`, including interim-login); `is_login` on `/wp-login.php` |
| Front | Home + singular + attachment HTML (password form), rewrite, feeds (RSS/Atom/RDF + comments RSS/Atom + multi enclosure meta + `rss_enclosure`/`atom_enclosure` filters; item `<comments>` via `comments_link_feed`; Atom HTML replies `#comments`; item `<guid isPermaLink="false">` even for pretty-permalink GUIDs; item `<content:encoded>` when available (`rss_use_excerpt` omits; empty content → excerpt; `the_content_feed` filter + `]]>` escape); optional item `<source url>` from `_rss_source_*` meta; item `<category domain="category\|post_tag">` from categories + tags; item author + `the_author` filter on `<dc:creator>`/Atom author; item `<slash:comments>` via `get_comments_number` filter; item `<wfw:commentRss>` via `get_post_comments_feed_link`; item Media RSS `<media:content>` + `<media:thumbnail>` from featured image + enclosures (`xmlns:media`); author-archive feed title/filter; RSS/Atom `rel=self` from request URI; RSS channel `<image>` from site icon; feed language from locale; optional `rss_copyright` → RSS `<copyright>` / Atom `<rights>`; optional `rss_ttl` → RSS `<ttl>`; `lastBuildDate` from newest post modified time; item `<pubDate>` / Atom dates GMT (+0000) with TZ-aware local fallback; RDF `admin:generatorAgent`; no PubSubHubbub hub links), robots/sitemap (`blog_public=0` → `Disallow: /` + meta noindex), comments, oEmbed discovery + endpoint, 404/embed/search (`search.html` + form)/date/author/tag/category archives (+ `rel=prev/next` on paginated archives; term-description via TermContext), pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header; `rel=canonical`; meta generator via `GetTheGenerator`; no `X-Pingback`; no `trackback_rdf` |
| Blocks / themes | Parser/render, core block subset (incl. post-title/content/excerpt/date/author/author-name/biography/avatar/terms, calendar/archives/latest-posts/latest-comments/tag-cloud/search, site-logo/title/tagline, home-link, loginout, page-list polish (`parentPageID`, nesting, `pages-list__item` classes, current/home), term-description (`textAlign`/`align`/`has-link-color`), read-more/more/nextpage, query-title polish (`showSearchTerm`/`showPrefix`/level=0→p/post-type/textAlign), post-navigation-link polish (`label`/`showTitle`/`linkLabel`/`arrow`/`textAlign`), gallery/image (+ align/dimensions/decorative/linkClass polish), table (+ hasFixedLayout/align/stripes synthesize), details (+ showContent/open, name, synthesize, fetchpriority=low when collapsed), html/freeform content-attr fallback, missing originalContent/originalUndelimitedContent fallbacks, legacy-widget empty (widgets gone), widget-group empty (sidebars gone), rss (local `/feed/` link list; remote blocked), audio/video synthesize + video dimensions, cover synthesize + featured image, media-text synthesize + featured image, buttons/button synthesize + empty/width polish, columns/column synthesize + stack/width/v-align, group synthesize + tagName/layout, spacer height/width polish, separator opacity/tagName polish, quote/pullquote synthesize + layout, code/preformatted/verse synthesize + align/textAlign, list/list-item `wp-block-list` + ordered attrs synthesize, heading/paragraph synthesize + class ensure, embed provider/type classes, social-links/social-link, footnotes polish, file synthesize + preview attrs), REST block-types library metadata (expanded titles/categories/descriptions), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations (theme development mode clears FileSHA256), patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages (GET `?sticky=` + `?orderby=title\|date\|id\|slug\|menu_order` + `order=` + `?search=` + `?author=` + `?categories=` + `?tags=` + `?status=` for editors + `?slug=` + `?include=`/`?exclude=` + `?after=`/`?before=`/`?modified_after=`/`?modified_before=` + `?offset=` + pages `?parent=`/`?parent_exclude=`)/media (POST create fields title/alt_text/caption/description/post + DELETE force)/comments (GET + write + DELETE trash/force)/taxonomies (incl. DELETE categories/tags)/users (GET + write + DELETE force/reassign)/settings (richer schema)/search (+ subtype post\|page)/statuses/themes (GET + status activate)/plugins (GET + status activate/deactivate), revisions (GET + DELETE force)/autosaves (GET + POST), menus/menu-items write/menu-locations (write + DELETE unset)/navigation (incl. DELETE force), templates (incl. DELETE trash/force)/global styles (incl. write + themes/{stylesheet} + /variations), font-families (GET + POST/DELETE force)/font-faces (GET + POST/DELETE force), block-renderer, block-patterns patterns/categories, url-details (same-site), batch (207 + header envelopes + require-all-validate), site-health stubs; widgets/sidebars disabled; remote directory proxies blocked |
| Admin | Dashboard (+ Quick Draft creates drafts; Activity lists recent comments; Site Health Status widget) + list tables (+ edit.php Quick Edit + Bulk Edit stubs + sticky toggle + search box `?s=` + author filter + date filter `?m=` + category filter + tag filter + All/Published/Drafts/Trash status views + Screen Options `edit_{type}_per_page` + Columns checkboxes (`manageedit-*columnshidden`) + hierarchical page indent `level-N` + em-dash pad + pages Order/`menu_order` column + Trash/Restore/Delete Permanently row+bulk + bottom `action2`), options screens (general/writing+reading+discussion+media save via options.php/permalink save/privacy save), media library grid/list (+ bulk delete permanently) + `media-new.php` + async upload, edit-comments bulk actions + comment.php reply form, plugins (row + bulk activate/deactivate + Uninstall for uninstallable)/themes (Activate + `switch_theme`)/users (+ bulk delete stub) + user-new create + user-edit stubs, edit-tags add/delete category/tag, nav menus, site editor shell, tools/import (WXR upload stub)/export (minimal WXR download)/privacy (requests table + erase confirm UI + suggested-text updated notice + policy guide TOC + accordion toggle + section search + export completion email + export download expiration (`wp_privacy_export_expiration`) + erasure fulfillment email), site-health status + info stubs, admin-ajax heartbeat (`wp_doing_ajax`) + `hidden-columns` |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins`), Hello Dolly shim (lyric rotation + texturize + CSS), activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter, `plugin_action_links` + `plugin_row_meta` filters), uninstall hook stub + Uninstall row button (`is-uninstallable`); `Requires Plugins` dependency stub (no remote API); PHP drop-ins detected as unsupported; Akismet not bundled (P9.8); Network Activate N/A on single site (P9.11); Plugin Automatic Updates N/A (P9.16); Plugin Updates via `update.php` N/A (P9.17) |

## Explicitly out of scope / gone

- Multisite, XML-RPC, trackbacks (`wp-trackback.php` + `trackback_rdf`), Customizer, Links Manager (`link_manager_enabled` always off)
- Theme/plugin install from wordpress.org, core updates, widgets REST / legacy-widget / widget-group server render
- Remote block/pattern directory fetches (`api.wordpress.org`)
- PHP drop-ins (`advanced-cache.php`, `object-cache.php`, `db.php`, …) — detected, never loaded
- Akismet (not shipped; activation rejected if files are dropped in)
- Network Activate / Network Deactivate (single-site only)
- Plugin Automatic Updates (enable/disable + wordpress.org update checks)
- Plugin Updates via `update.php` (upgrade-plugin / update-selected)
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

## Recent stack (through P10.71)

- P10.71 STATUS/PORTING refresh after P5.79–P8.20
- P8.20 Privacy export download expiration (`wp_privacy_export_expiration`)
- P3.73 Feed item `the_content_feed` on `content:encoded` / Atom content
- P2.75 `wp_get_referer` / `ValidateRedirect`
- P7.80 `edit.php` Screen Options Columns checkboxes
- P4.95 REST posts `?modified_after=` / `?modified_before=`
- P5.79 `core/rss` local feed link list (remote blocked)
- P10.70 E2E for `edit.php` Screen Options per page + STATUS/PORTING refresh after P5.78–P6.33
- P6.33 Interim login success `<title>` keeps `&lsaquo;` with empty `login_header` title
- P3.72 Feed item `wfw:commentRss` via `get_post_comments_feed_link`
- P2.74 `wp_get_raw_referer` / `GetRawReferer`
- P7.79 `edit.php` pages Order (`menu_order`) column
- P4.94 REST pages `?orderby=menu_order`
- P5.78 `core/widget-group` empty placeholder (widgets/sidebars gone)
