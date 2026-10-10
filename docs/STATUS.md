# GoPress status summary

Short overview of what is implemented versus remaining. Detail lives in `PORTING.md` (generated from `tools/porting-map.yaml`).

## Implemented (working subsets)

| Area | Coverage |
|---|---|
| Runtime core | Boot, config, DB, hooks, object cache, options/meta, i18n stub, KSES, formatting (`wp_strip_all_tags`, `wp_html_split`, `wp_basename`, `wp_is_uuid`, `wp_is_hex_color`/`sanitize_hex_color`, `wp_get_image_mime`, `wp_unique_filename`, `wp_ext2type`/`wp_get_ext_types`, `wp_getimagesize`, `wp_check_filetype`/`wp_check_filetype_and_ext`/`wp_get_mime_types`/`get_allowed_mime_types($user)` + `upload_mimes` on uploads, `wp_get_upload_dir` path/url/subdir, `wp_mkdir_p`, `wp_tempnam`/`get_temp_dir`, `wp_is_ini_value_changeable`, `wp_is_json_request`/`wp_is_json_media_type`, `wp_is_rest_endpoint`/`wp_is_serving_rest_request`, `wp_doing_ajax`, `wp_doing_cron`, `is_login`, `wp_get_environment_type` (getenv then define/config; `IsEnvironmentType`), `wp_debug_mode` (`WP_DEBUG`/`DISPLAY`/`LOG`, `enable_wp_debug_mode_checks`, REST/ajax/JSON display suppression), `wp_get_development_mode`/`wp_is_development_mode`, `get_current_user_id`/`wp_get_current_user`/`wp_set_current_user` (`SetCurrentUser`, ID≤0 anonymous, `set_current_user` action), `wp_get_session_token`, `wp_get_raw_referer`, `wp_get_referer`/`wp_validate_redirect`/`wp_sanitize_redirect`/`wp_safe_redirect`, `is_ssl` (TLS + HTTPS CGI on/1 + port 443 + X-Forwarded-Proto), `wp_parse_url` (schemeless host:port, empty pass, bare http(s) authority), `get_user_option`/`update_user_option`, `wp_check_invalid_utf8`, `wp_specialchars_decode`, `wp_validate_boolean`, `wp_slash`/`wp_unslash`/`stripslashes_deep` (map[string]string + map[any]any walks), `wp_is_numeric_array`, `wp_list_pluck`, `wp_filter_object_list`, `wp_list_sort`, `wp_recursive_ksort`, `wp_array_key_first`/`wp_array_key_last`, `wp_array_slice_assoc`, `wp_unique_id`, `wp_normalize_path`/`wp_is_stream` (case-sensitive wrappers; used by MkdirP/UniqueFilename/TempNam), `WP_Http::is_ip_address`, `wp_spaces_regexp`, intermediate image sizes (`get_intermediate_image_sizes` + filter, `has_image_size`, `wp_get_registered_image_subsizes`, `image_size_names_choose`), `wp_read_image_metadata` + generate/store `image_meta` on image upload, …), shortcodes, transients, cron stub (`DOING_CRON` on `/wp-cron.php`); `wp-emoji-settings` JSON in head (loader skipped) |
| Auth / users | Password hashes (phpass / `$wp$`), roles/caps, auth cookies (Remember Me → 14d + MaxAge; HTTPS → `wordpress_sec_` + Secure flags), usermeta `session_tokens`, nonces, login/logout/lost-password (`redirect_to` + `lostpassword_redirect`)/resetpass/postpass/`reauth` (any non-empty REQUEST value; `SetCurrentUser(0)`)/interim session-expired message (`login_messages`) + `aria-describedby` focus association + POST-body interim + blank-target links + interim success (transparent chrome, `login_message`, auth cookies for auth-check, empty-title `&lsaquo;` document title) + auth-check modal same-domain/`wp_lang`, `testcookie` / `wordpress_test_cookie`, `wp_lang` cookie from `?wp_lang=` + language switcher preserves POST `redirect_to`/`action`/`interim-login`, external `redirect_to` rejected, default login → `admin_url` + `login_redirect` (no `edit_posts` → `profile.php`), `confirm_admin_email` stub, application passwords; REST user create/update/DELETE (force + reassign) + `/users/me` avatar/capabilities/`_links`/self-update; `user-new.php` POST create; `user-edit.php` stub; users.php bulk delete + Change role (`new_role`/`changeit`); login `nocache_headers` + `login_error` shake (`wp_shake_js`, including interim-login); `is_login` on `/wp-login.php` |
| Front | Home + singular + attachment HTML (password form), rewrite, `show_on_front`/`page_on_front`/`page_for_posts` (static front + posts page conditionals), feeds (RSS/Atom/RDF + comments RSS/Atom + multi enclosure meta + `rss_enclosure`/`atom_enclosure` filters; item `<comments>` via `comments_link_feed`; Atom HTML replies `#comments`; item `<guid isPermaLink="false">` even for pretty-permalink GUIDs (`get_the_guid` filter); comments RSS2 item GUID from post GUID + `get_comment_text` “In reply to” prefix; item `<content:encoded>` when available (`rss_use_excerpt` omits; empty content → excerpt; `the_content_feed` filter + `]]>` escape); item `<title>` via `the_title_rss`; item `<description>` / Atom summary via `the_excerpt_rss`; optional item `<source url>` from `_rss_source_*` meta; item `<category domain="category\|post_tag">` from categories + tags; item author + `the_author` filter on `<dc:creator>`/Atom author; item `<slash:comments>` via `get_comments_number` filter; item `<wfw:commentRss>` via `get_post_comments_feed_link`; item Media RSS `<media:content>` + `<media:thumbnail>` from featured image + enclosures (`xmlns:media`); author-archive feed title/filter; RSS/Atom `rel=self` from request URI; RSS channel `<image>` from site icon; feed language from locale; optional `rss_copyright` → RSS `<copyright>` / Atom `<rights>`; optional `rss_ttl` → RSS `<ttl>`; `lastBuildDate` from newest post modified time; item `<pubDate>` / Atom dates GMT (+0000) with TZ-aware local fallback; RDF `admin:generatorAgent`; no PubSubHubbub hub links), robots/sitemap (`blog_public=0` → `Disallow: /` + meta noindex + sitemap 404; `robots_txt` filter), comments, oEmbed discovery + endpoint, 404 (Scope.Is404 + `document_title_parts` + `404_template_content` + search/home shim)/embed/search (`search.html` + form; `document_title_parts` + `search_template_content`; shim form/excerpts)/date/author/tag/category archives (+ `rel=prev/next` on paginated archives; term-description via TermContext), pagination, admin bar; `home_url`/`site_url`/`admin_url` helpers; `?rest_route=` REST front door; `/favicon.ico`; `/rsd.xml` stub (`xmlrpc.php?rsd` stays 403); `?p=` shortlink head + `Link` header; `rel=canonical`; bidirectional trailing-slash `redirect_canonical` (+ collapse `//`); meta generator via `GetTheGenerator`; no `X-Pingback`; no `trackback_rdf` |
| Blocks / themes | Parser/render, core block subset (incl. post-title polish (level 0→p, textAlign/has-link-color/className, isLink+target/rel, empty omits)/post-content polish (tagName, align/className, seen_ids reentrancy + DebugDisplay halt)/excerpt/date polish (`displayType=modified`, format/human-diff, textAlign/has-link-color; isLink href-only unless target/rel)/author polish (showAvatar/avatarSize/showBio/itemsJustification)/author-name/biography/avatar/terms polish (prefix/suffix/separator spans), calendar/archives/latest-posts/latest-comments/tag-cloud (`showTagCounts`/`numberOfTags`/`taxonomy`)/search, site-logo/title/tagline, home-link, loginout, page-list polish (`parentPageID`, nesting, `pages-list__item` classes, current/home), term-description (`textAlign`/`align`/`has-link-color`), read-more/more/nextpage, query-title polish (`showSearchTerm`/`showPrefix`/level=0→p/post-type/textAlign), query/post-template polish (`tagName`/`align`, order/orderBy/search/author/sticky/exclude/format, layout grid/flex/`has-link-color`), post-featured-image polish (`sizeSlug`/dimensions/`scale`/`isLink`+target/rel/`dimRatio` overlay/align), post-navigation-link polish (`label`/`showTitle`/`linkLabel`/`arrow`/`textAlign`), gallery/image (+ align/dimensions/decorative/linkClass polish), table (+ hasFixedLayout/align/stripes synthesize), details (+ showContent/open, name, synthesize, fetchpriority=low when collapsed), html/freeform content-attr fallback, missing originalContent/originalUndelimitedContent fallbacks, legacy-widget empty (widgets gone), widget-group empty (sidebars gone), rss (local `/feed/` link list; remote blocked), categories (term list/hierarchy), shortcode (`wpautop` on text; expand via `DoShortcode`), audio/video synthesize + video dimensions, cover synthesize + featured image, media-text synthesize + featured image, buttons/button synthesize + empty/width polish, columns/column synthesize + stack/width/v-align, group synthesize + tagName/layout, spacer height/width polish, separator opacity/tagName polish, quote/pullquote synthesize + layout, code/preformatted/verse synthesize + align/textAlign, list/list-item `wp-block-list` + ordered attrs synthesize, heading/paragraph synthesize + class ensure, embed provider/type classes, social-links/social-link (custom icon colors, flex layout, has-visible-labels, bluesky/mastodon/…), navigation-link/submenu (opensInNewTab/rel/description/current-menu-item/has-child), footnotes polish, file synthesize + preview attrs), REST block-types library metadata (expanded titles/categories/descriptions), supports (typography/border/layout/shadow/position/elements/background/dimensions/CSS filter+duotone), style engine, theme.json + style variations (theme development mode clears FileSHA256), patterns + pattern categories, template hierarchy/parts (404 header/footer), bindings, block styles/variations, expanded Block Hooks (nested + ignored metadata) |
| REST (`/wp-json`) | Index, posts/pages (GET `?sticky=` + POST/PUT `sticky` write + `?orderby=title\|date\|id\|slug\|menu_order` + `order=` + `?search=` + `?author=` + `?categories=` + `?tags=` + `?status=` for editors + `?slug=` + `?include=`/`?exclude=` + `?after=`/`?before=`/`?modified_after=`/`?modified_before=` + `?format=` (post_format / standard) + `?offset=` + pages `?parent=`/`?parent_exclude=`)/media (POST create fields title/alt_text/caption/description/post + DELETE force + `wp_get_attachment_metadata` → media details/sizes)/comments (GET `?status=` approve|hold|spam|trash|all + write + DELETE trash/force)/taxonomies (incl. DELETE categories/tags)/users (GET + write + DELETE force/reassign + `/users/me` richer fields/self-update)/settings (richer schema)/search (+ subtype post|page + include/exclude)/statuses/themes (GET + status activate)/plugins (GET + status activate/deactivate), revisions (GET + DELETE force)/autosaves (GET + POST), menus/menu-items write/menu-locations (write + DELETE unset)/navigation (incl. DELETE force), templates (incl. DELETE trash/force)/global styles (incl. write + themes/{stylesheet} + /variations), font-families (GET + POST/DELETE force)/font-faces (GET + POST/DELETE force), block-renderer, block-patterns patterns/categories, url-details (same-site), batch (207 + header envelopes + require-all-validate), site-health stubs; widgets/sidebars/widget-types disabled (`edit_theme_options` → empty collections; else `rest_cannot_manage_widgets`; auth GET sidebar item → `rest_sidebar_invalid_id`; auth GET widget-type item → `rest_widget_type_invalid`); remote directory proxies blocked (block-directory requires `term`, then 403) |
| Admin | Dashboard (+ Quick Draft creates drafts; Activity lists recent comments; Site Health Status widget) + list tables (+ edit.php Quick Edit + Bulk Edit stubs + sticky toggle + search box `?s=` + author filter + date filter `?m=` + category filter + tag filter + post format filter + All/Published/Sticky/Drafts/Trash status views (`show_sticky`) + Screen Options `edit_{type}_per_page` + Columns checkboxes (`manageedit-*columnshidden`) + hierarchical page indent `level-N` + em-dash pad + pages Order/`menu_order` column + Trash/Restore/Delete Permanently row+bulk + bottom `action2`), options screens (general/writing+reading (`page_on_front`/`page_for_posts` selects)+discussion+media save via options.php/permalink save/privacy save), media library grid/list (+ bulk delete permanently with bulk-media nonce + action2) + `media-new.php` + async upload, edit-comments bulk actions + All/Pending/Approved/Spam/Trash views + unspam/untrash + comment.php reply form, plugins (All/Active/Inactive views + search + row + bulk activate/deactivate + Uninstall for uninstallable)/themes (Activate + `switch_theme` + search/`theme-count`/Add Theme)/users (+ bulk delete + Change role promote) + user-new create + user-edit stubs, edit-tags add/delete category/tag, nav menus, site editor shell, Customizer stub (`customize` cap denial + validated `?return=` + Site Editor fallback), tools (`tool_box`, import-cap converter card, privacy-guide 301)/import (WXR upload stub)/export (minimal WXR download)/privacy (requests table + erase `user_request` create stub + erase confirm UI + suggested-text updated notice + policy guide TOC + accordion toggle + section search + export completion email + export download expiration (`wp_privacy_export_expiration`) + erasure fulfillment email), site-health status + info stubs, admin-ajax heartbeat (`wp_doing_ajax`) + `hidden-columns` |
| Plugins | MU-plugin scan + header parse (`get_mu_plugins` + filter) + `muplugins_loaded`, Hello Dolly shim (lyric rotation + texturize + CSS), activate/deactivate (+ activate_/activated_/deactivate_/deactivated_ hooks, `active_plugins` filter, `plugin_action_links` + `plugin_row_meta` filters), uninstall hook stub + Uninstall row button (`is-uninstallable`); `Requires Plugins` dependency stub (no remote API); PHP drop-ins detected as unsupported; Akismet not bundled (P9.8); Network Activate N/A on single site (P9.11); Plugin Automatic Updates N/A (P9.16); Plugin Updates via `update.php` N/A (P9.17) |

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

## Recent stack (through P10.87)

- P10.87 STATUS/PORTING refresh after P5.88–P10.86
- P10.86 Diff corpus expand (165/165 match)
- P6.37 Login `reauth` REQUEST deepen + `SetCurrentUser(0)`
- P3.82 Deepen search (`document_title_parts`, `search_template_content`, shim form)
- P2.84 `wp_set_current_user` / `GetCurrentUserID` ID≤0 anonymous
- P7.89 `tools.php` deepen (`tool_box`, converter caps, privacy-guide redirect)
- P4.104 Widget-types REST item `rest_widget_type_invalid`
- P5.88 `core/post-date` / `post-author` / `post-terms` polish
- P10.85 STATUS/PORTING refresh after P5.87–P10.84
- P10.84 Diff corpus expand (142/142 match)
- P8.24 `wp_check_filetype_and_ext` + `upload_mimes` on uploads
- P3.81 Deepen 404 (`Is404`, `document_title_parts`, search/home shim)
- P2.83 `wp_debug_mode` / `IsDebugDisplay`
- P7.88 `customize.php` stub (cap denial + return URL)
- P4.103 Sidebars REST item `rest_sidebar_invalid_id`
- P5.87 `core/post-title` / `post-content` polish
- P10.83 STATUS/PORTING refresh after P5.86–P10.82
- P10.82 Diff corpus expand (113/113 match)
- P9.19 `muplugins_loaded` + `get_mu_plugins` filter
- P3.80 Front page display (`show_on_front` / `page_on_front` / `page_for_posts`)
- P2.82 `wp_get_environment_type` resolution + `IsEnvironmentType`
- P7.87 `themes.php` search, theme-count, Add Theme
- P4.102 Widgets REST `edit_theme_options` permission gate
- P5.86 `core/post-featured-image` polish
- P10.81 STATUS/PORTING refresh after P5.85–P10.80
- P10.80 Diff corpus expand (87/87 match)
- P8.23 Intermediate image sizes registry (`get_intermediate_image_sizes`, `has_image_size`, subsizes, `image_size_names_choose`)
- P3.79 Bidirectional trailing-slash canonical + collapse `//`
- P2.81 `is_ssl` HTTPS CGI on/1 + port 443
- P7.86 `plugins.php` All/Active/Inactive views + search
- P4.101 REST block-directory requires `term` (remote still 403)
- P5.85 `core/query` / `post-template` polish
- P10.79 STATUS/PORTING refresh after P5.84–P10.78
- P10.78 Diff corpus expand (74/74 match) + `/users/me` Allow GET when anon
- P6.36 Interim login auth-check modal (`wp_auth_check_same_domain`, `wp_lang`)
- P3.78 `robots_txt` filter + sitemaps 404 when `blog_public=0`
- P2.80 `wp_normalize_path` / case-sensitive `wp_is_stream` + call sites
- P7.85 Media bulk delete nonce + `action2` + mode preserve
- P4.100 REST search `?include=` / `?exclude=`
- P5.84 `core/navigation-link` / `navigation-submenu` polish
- P10.77 STATUS/PORTING refresh after P5.83–P10.76
- P10.76 Diff corpus expand (63/63 match)
- P8.22 `wp_get_attachment_metadata` read path + REST media details
- P3.77 Comments RSS2 `get_comment_guid` + reply prefix
- P2.79 `wp_parse_url` / `ParseURL` host:port + empty pass
- P7.84 `users.php` bulk Change role (`new_role`/`changeit`)
- P4.99 REST `/wp/v2/users/me` fields + self-update
- P5.83 `core/social-links` custom colors, flex layout, more services
- P10.75 STATUS/PORTING refresh after P5.82–P10.74
- P10.74 Diff corpus expand (45/45 match)
- P6.35 Lost-password `redirect_to` / `lostpassword_redirect`
- P3.76 Feed `get_the_guid` filter on item GUID/id
- P2.78 `wp_specialchars_decode` / `SpecialcharsDecode`
- P7.83 `edit-comments.php` status views + bulk unspam/untrash
- P4.98 REST comments `?status=` filter + moderator item fetch
- P5.82 `core/tag-cloud` `numberOfTags` + `taxonomy`
- P10.73 STATUS/PORTING refresh after P5.81–P8.21
- P8.21 Privacy erase `user_request` create stub (`wp_create_user_request`)
- P3.75 Feed item `the_title_rss` on item titles
- P2.77 `wp_safe_redirect` / `SafeRedirect` wrapping `ValidateRedirect`
- P7.82 `edit.php` Sticky view (`?show_sticky=1`)
- P4.97 REST posts sticky write on create/update
- P5.81 `core/categories` term list block
- P10.72 E2E for `edit.php` Screen Options Columns + STATUS/PORTING refresh after P5.80–P6.34
- P6.34 Default login redirect to `admin_url` + `login_redirect` / profile for no `edit_posts`
- P3.74 Feed item `the_excerpt_rss` on `<description>` / Atom summary
- P2.76 `wp_sanitize_redirect` + relative `ValidateRedirect`
- P7.81 `edit.php` post format filter (`?post_format=`)
- P4.96 REST posts `?format=`
- P5.80 `core/shortcode` block (`wpautop`)
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
