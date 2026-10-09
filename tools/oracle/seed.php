<?php
/**
 * Deterministic GoPress fixture seeder (P0.4).
 *
 * Run: wp eval-file tools/oracle/seed.php --path=/opt/gopress-ref --url=http://127.0.0.1:8081 --user=admin
 *
 * @package GoPress
 */

if ( ! defined( 'ABSPATH' ) ) {
	fwrite( STDERR, "Run via: wp eval-file tools/oracle/seed.php --path=/opt/gopress-ref --user=admin\n" );
	exit( 1 );
}

require_once ABSPATH . 'wp-admin/includes/file.php';
require_once ABSPATH . 'wp-admin/includes/media.php';
require_once ABSPATH . 'wp-admin/includes/image.php';
require_once ABSPATH . 'wp-admin/includes/image-edit.php';
require_once ABSPATH . 'wp-admin/includes/post.php';

// Avoid sendmail noise during seeding (no MTA in the dev VM).
add_filter(
	'pre_wp_mail',
	static function () {
		return true;
	}
);

/** @var string Fixed "now" used for all generated timestamps. */
const GOPRESS_FIXTURE_NOW = '2024-06-15 12:00:00';
const GOPRESS_FIXTURE_NOW_GMT = '2024-06-15 12:00:00';

/**
 * Repo-root media directory (committed samples).
 *
 * @return string
 */
function gopress_fixture_media_dir() {
	// tools/oracle/seed.php → repo root is two levels up when run from the checkout;
	// when copied, fall back to GOPRESS_REPO /workspace.
	$candidates = array(
		getenv( 'GOPRESS_REPO' ) ? rtrim( getenv( 'GOPRESS_REPO' ), '/' ) . '/tests/fixtures/media' : '',
		dirname( __DIR__, 2 ) . '/tests/fixtures/media',
		'/workspace/tests/fixtures/media',
	);
	foreach ( $candidates as $dir ) {
		if ( $dir && is_dir( $dir ) ) {
			return $dir;
		}
	}
	WP_CLI::error( 'tests/fixtures/media not found' );
}

/**
 * @param string $local Local datetime Y-m-d H:i:s.
 * @return string GMT datetime.
 */
function gopress_fixture_gmt( $local ) {
	// Oracle timezone is UTC after install defaults unless changed; keep equal.
	return $local;
}

/**
 * Create or update a user with a fixed role and profile.
 *
 * @param array $args User fields.
 * @return int User ID.
 */
function gopress_fixture_user( array $args ) {
	$login = $args['user_login'];
	$existing = get_user_by( 'login', $login );
	$data = array(
		'user_login'   => $login,
		'user_pass'    => $args['user_pass'] ?? 'password',
		'user_email'   => $args['user_email'],
		'display_name' => $args['display_name'] ?? $login,
		'nickname'     => $args['nickname'] ?? $login,
		'first_name'   => $args['first_name'] ?? '',
		'last_name'    => $args['last_name'] ?? '',
		'description'  => $args['description'] ?? '',
		'role'         => $args['role'],
	);
	if ( $existing ) {
		$data['ID'] = $existing->ID;
		$id = wp_update_user( $data );
	} else {
		$id = wp_insert_user( $data );
	}
	if ( is_wp_error( $id ) ) {
		WP_CLI::error( $id->get_error_message() );
	}
	global $wpdb;
	$wpdb->update(
		$wpdb->users,
		array(
			'user_registered' => $args['user_registered'] ?? '2024-01-01 00:00:00',
			'user_url'        => $args['user_url'] ?? '',
		),
		array( 'ID' => $id )
	);
	clean_user_cache( $id );
	return (int) $id;
}

WP_CLI::log( 'Seeding GoPress fixture…' );

// --------------------------------------------------------------------------
// Users (one per role)
// --------------------------------------------------------------------------
$users = array(
	'admin' => gopress_fixture_user(
		array(
			'user_login'      => 'admin',
			'user_email'      => 'admin@example.com',
			'role'            => 'administrator',
			'display_name'    => 'Admin User',
			'first_name'      => 'Ada',
			'last_name'       => 'Admin',
			'description'     => 'Fixture administrator.',
			'user_registered' => '2024-01-01 10:00:00',
			'user_url'        => 'https://example.com/admin',
		)
	),
	'editor' => gopress_fixture_user(
		array(
			'user_login'      => 'editor',
			'user_email'      => 'editor@example.com',
			'role'            => 'editor',
			'display_name'    => 'Ed Editor',
			'first_name'      => 'Ed',
			'last_name'       => 'Editor',
			'description'     => 'Fixture editor.',
			'user_registered' => '2024-01-02 11:00:00',
			'user_url'        => 'https://example.com/editor',
		)
	),
	'author' => gopress_fixture_user(
		array(
			'user_login'      => 'author',
			'user_email'      => 'author@example.com',
			'role'            => 'author',
			'display_name'    => 'Ann Author',
			'first_name'      => 'Ann',
			'last_name'       => 'Author',
			'description'     => 'Fixture author.',
			'user_registered' => '2024-01-03 12:00:00',
		)
	),
	'contributor' => gopress_fixture_user(
		array(
			'user_login'      => 'contributor',
			'user_email'      => 'contributor@example.com',
			'role'            => 'contributor',
			'display_name'    => 'Con Contributor',
			'first_name'      => 'Con',
			'last_name'       => 'Contributor',
			'description'     => 'Fixture contributor.',
			'user_registered' => '2024-01-04 13:00:00',
		)
	),
	'subscriber' => gopress_fixture_user(
		array(
			'user_login'      => 'subscriber',
			'user_email'      => 'subscriber@example.com',
			'role'            => 'subscriber',
			'display_name'    => 'Sub Subscriber',
			'first_name'      => 'Sub',
			'last_name'       => 'Subscriber',
			'description'     => 'Fixture subscriber.',
			'user_registered' => '2024-01-05 14:00:00',
		)
	),
);

// Non-default admin color for the editor.
update_user_meta( $users['editor'], 'admin_color', 'coffee' );

// Application password row is written with fixed uuid/hash in the freeze step below.

// Register show_in_rest post meta used by posts.
register_post_meta(
	'post',
	'gopress_fixture_meta',
	array(
		'type'              => 'string',
		'single'            => true,
		'show_in_rest'      => true,
		'auth_callback'     => '__return_true',
		'sanitize_callback' => 'sanitize_text_field',
	)
);

// --------------------------------------------------------------------------
// Taxonomy: categories 3 levels deep, 20 tags
// --------------------------------------------------------------------------
function gopress_ensure_term( $name, $taxonomy, $args = array() ) {
	$existing = term_exists( $name, $taxonomy );
	if ( $existing ) {
		return (int) ( is_array( $existing ) ? $existing['term_id'] : $existing );
	}
	$r = wp_insert_term( $name, $taxonomy, $args );
	if ( is_wp_error( $r ) ) {
		WP_CLI::error( $r->get_error_message() );
	}
	return (int) $r['term_id'];
}

$cat_root = gopress_ensure_term( 'Fixture Root', 'category', array( 'slug' => 'fixture-root' ) );
$cat_mid  = gopress_ensure_term( 'Fixture Mid', 'category', array( 'slug' => 'fixture-mid', 'parent' => $cat_root ) );
$cat_leaf = gopress_ensure_term( 'Fixture Leaf', 'category', array( 'slug' => 'fixture-leaf', 'parent' => $cat_mid ) );

$tag_ids = array();
for ( $i = 1; $i <= 20; $i++ ) {
	$tag_ids[] = gopress_ensure_term(
		sprintf( 'Fixture Tag %02d', $i ),
		'post_tag',
		array( 'slug' => sprintf( 'fixture-tag-%02d', $i ) )
	);
}

// --------------------------------------------------------------------------
// Media
// --------------------------------------------------------------------------
$media_dir = gopress_fixture_media_dir();
$attachment_ids = array();
$media_files = array(
	'sample-photo-a.jpg',
	'sample-photo-b.jpg',
	'sample-photo-c.jpg',
	'sample-graphic.png',
	'sample-photo.webp',
	'sample-doc.pdf',
	'sample-audio.mp3',
	'sample-video.mp4',
);

foreach ( $media_files as $filename ) {
	$path = $media_dir . '/' . $filename;
	if ( ! is_file( $path ) ) {
		WP_CLI::error( "missing media file: {$path}" );
	}
	// Skip re-upload if an attachment with this basename already exists.
	$found = get_posts(
		array(
			'post_type'      => 'attachment',
			'post_status'    => 'inherit',
			'posts_per_page' => 1,
			'meta_key'       => '_wp_attached_file',
			'meta_value'     => $filename, // may be in dated subdir; search by title instead
			'title'          => pathinfo( $filename, PATHINFO_FILENAME ),
			'fields'         => 'ids',
		)
	);
	// Title search is unreliable; use filename meta LIKE.
	global $wpdb;
	$like = '%' . $wpdb->esc_like( $filename );
	$existing_id = (int) $wpdb->get_var(
		$wpdb->prepare(
			"SELECT post_id FROM {$wpdb->postmeta} WHERE meta_key = '_wp_attached_file' AND meta_value LIKE %s LIMIT 1",
			$like
		)
	);
	if ( $existing_id ) {
		$attachment_ids[ $filename ] = $existing_id;
		continue;
	}

	$file_array = array(
		'name'     => $filename,
		'tmp_name' => $path,
	);
	// media_handle_sideload moves the file; copy first.
	$tmp = wp_tempnam( $filename );
	copy( $path, $tmp );
	$file_array['tmp_name'] = $tmp;
	$id = media_handle_sideload( $file_array, 0, null, array( 'post_author' => $users['admin'] ) );
	if ( is_wp_error( $id ) ) {
		@unlink( $tmp );
		WP_CLI::error( $filename . ': ' . $id->get_error_message() );
	}
	$attachment_ids[ $filename ] = (int) $id;
	$wpdb->update(
		$wpdb->posts,
		array(
			'post_date'         => '2024-03-01 09:00:00',
			'post_date_gmt'     => '2024-03-01 09:00:00',
			'post_modified'     => '2024-03-01 09:00:00',
			'post_modified_gmt' => '2024-03-01 09:00:00',
		),
		array( 'ID' => $id )
	);
}

// Crop sample-photo-c.jpg (image editor).
$crop_id = $attachment_ids['sample-photo-c.jpg'];
$file = get_attached_file( $crop_id );
if ( $file && file_exists( $file ) ) {
	$editor = wp_get_image_editor( $file );
	if ( ! is_wp_error( $editor ) ) {
		$editor->crop( 10, 10, 400, 300 );
		$saved = $editor->save( $file );
		if ( ! is_wp_error( $saved ) ) {
			$meta = wp_generate_attachment_metadata( $crop_id, $file );
			wp_update_attachment_metadata( $crop_id, $meta );
		}
	}
}

// --------------------------------------------------------------------------
// Posts covering statuses / formats / sticky / password / more / nextpage
// --------------------------------------------------------------------------
$statuses = array( 'publish', 'future', 'draft', 'pending', 'private', 'trash', 'auto-draft' );
$formats  = array( 'aside', 'gallery', 'link', 'image', 'quote', 'status', 'video', 'audio', 'chat' );

$post_ids = array();
$base_day = 1;
for ( $n = 1; $n <= 40; $n++ ) {
	$status = $statuses[ ( $n - 1 ) % count( $statuses ) ];
	$date   = sprintf( '2024-02-%02d %02d:00:00', min( 28, $base_day + (int) ( ( $n - 1 ) / 7 ) ), 8 + ( $n % 10 ) );
	$title  = sprintf( 'Fixture Post %02d', $n );
	$slug   = sprintf( 'fixture-post-%02d', $n );
	$content = "<!-- wp:paragraph -->\n<p>Fixture post {$n} body.</p>\n<!-- /wp:paragraph -->";

	if ( 5 === $n ) {
		$content .= "\n<!--more-->\n<!-- wp:paragraph -->\n<p>After the more tag.</p>\n<!-- /wp:paragraph -->";
	}
	if ( 6 === $n ) {
		$content .= "\n<!--nextpage-->\n<!-- wp:paragraph -->\n<p>Page two.</p>\n<!-- /wp:paragraph -->";
	}
	if ( 7 === $n ) {
		// Password-protected.
	}

	$postarr = array(
		'post_title'    => $title,
		'post_name'     => $slug,
		'post_content'  => $content,
		'post_status'   => $status,
		'post_author'   => $users[ array( 'admin', 'editor', 'author' )[ ( $n - 1 ) % 3 ] ],
		'post_date'     => $date,
		'post_date_gmt' => gopress_fixture_gmt( $date ),
		'post_category' => array( $cat_leaf ),
		'tags_input'    => array( sprintf( 'Fixture Tag %02d', ( ( $n - 1 ) % 20 ) + 1 ) ),
	);
	if ( 'future' === $status ) {
		$postarr['post_date']     = '2026-12-01 10:00:00';
		$postarr['post_date_gmt'] = '2026-12-01 10:00:00';
	}
	if ( 7 === $n ) {
		$postarr['post_password'] = 'secret';
		$postarr['post_status']   = 'publish';
	}

	$existing = get_page_by_path( $slug, OBJECT, 'post' );
	if ( $existing ) {
		$postarr['ID'] = $existing->ID;
		$id = wp_update_post( $postarr, true );
	} else {
		$id = wp_insert_post( $postarr, true );
	}
	if ( is_wp_error( $id ) ) {
		WP_CLI::error( $id->get_error_message() );
	}
	$post_ids[ $n ] = (int) $id;
	update_post_meta( $id, 'gopress_fixture_meta', 'meta-for-' . $slug );
}

// Sticky.
stick_post( $post_ids[1] );

// One post per post format (reuse posts 10+).
foreach ( $formats as $i => $format ) {
	$pid = $post_ids[ 10 + $i ];
	set_post_format( $pid, $format );
}

// Revisions on 3 posts, one autosave, one post lock.
foreach ( array( 1, 2, 3 ) as $n ) {
	$pid = $post_ids[ $n ];
	for ( $r = 1; $r <= 3; $r++ ) {
		wp_update_post(
			array(
				'ID'           => $pid,
				'post_content' => get_post( $pid )->post_content . "\n<!-- revision {$r} -->",
			)
		);
	}
}
// Autosave for post 1.
wp_create_post_autosave(
	array(
		'post_ID'      => $post_ids[1],
		'post_title'   => get_the_title( $post_ids[1] ),
		'post_content' => get_post( $post_ids[1] )->post_content . "\n<!-- autosave -->",
		'post_type'    => 'post',
	)
);
// Post lock by editor on post 2.
wp_set_post_lock( $post_ids[2] );
// Ensure lock user is editor.
update_post_meta( $post_ids[2], '_edit_lock', time() . ':' . $users['editor'] );

// Gutenberg block fixture posts.
$fixture_glob = '/opt/gutenberg/test/integration/fixtures/blocks/core__*.serialized.html';
$block_files  = glob( $fixture_glob );
sort( $block_files );
$block_count = 0;
foreach ( $block_files as $file ) {
	$base = basename( $file, '.serialized.html' );
	$slug = 'fixture-' . sanitize_title( $base );
	$content = file_get_contents( $file );
	$existing = get_page_by_path( $slug, OBJECT, 'post' );
	$postarr = array(
		'post_title'   => 'Block fixture ' . $base,
		'post_name'    => $slug,
		'post_content' => $content,
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-04-01 10:00:00',
		'post_date_gmt'=> '2024-04-01 10:00:00',
	);
	if ( $existing ) {
		$postarr['ID'] = $existing->ID;
		wp_update_post( $postarr );
	} else {
		wp_insert_post( $postarr );
	}
	$block_count++;
}
WP_CLI::log( "Block fixture posts: {$block_count}" );

// --------------------------------------------------------------------------
// Pages: 3-level hierarchy + templates
// --------------------------------------------------------------------------
function gopress_fixture_page( $title, $slug, $parent = 0, $template = '' ) {
	$existing = get_page_by_path( ( $parent ? get_page_uri( $parent ) . '/' : '' ) . $slug );
	$postarr = array(
		'post_title'   => $title,
		'post_name'    => $slug,
		'post_content' => "<!-- wp:paragraph --><p>{$title}</p><!-- /wp:paragraph -->",
		'post_status'  => 'publish',
		'post_type'    => 'page',
		'post_parent'  => $parent,
		'post_author'  => 1,
		'post_date'    => '2024-02-10 09:00:00',
		'post_date_gmt'=> '2024-02-10 09:00:00',
	);
	if ( $existing ) {
		$postarr['ID'] = $existing->ID;
		$id = wp_update_post( $postarr, true );
	} else {
		$id = wp_insert_post( $postarr, true );
	}
	if ( is_wp_error( $id ) ) {
		WP_CLI::error( $id->get_error_message() );
	}
	if ( $template ) {
		update_post_meta( $id, '_wp_page_template', $template );
	}
	return (int) $id;
}

$page_l1 = gopress_fixture_page( 'Fixture Parent', 'fixture-parent' );
$page_l2 = gopress_fixture_page( 'Fixture Child', 'fixture-child', $page_l1 );
$page_l3 = gopress_fixture_page( 'Fixture Grandchild', 'fixture-grandchild', $page_l2, 'page-with-sidebar' );

$blog_page = gopress_fixture_page( 'Blog', 'blog' );
$home_page = gopress_fixture_page( 'Home', 'home-static' );

// Default: latest posts. Static variant dumped separately by fixture.sh.
update_option( 'show_on_front', 'posts' );
update_option( 'page_on_front', 0 );
update_option( 'page_for_posts', 0 );

// --------------------------------------------------------------------------
// Comments: 3-level threading + statuses
// --------------------------------------------------------------------------
function gopress_fixture_comment( $args ) {
	$defaults = array(
		'comment_author'       => 'Commenter',
		'comment_author_email' => 'commenter@example.com',
		'comment_author_url'   => '',
		'comment_content'      => 'A fixture comment.',
		'comment_type'         => 'comment',
		'comment_parent'       => 0,
		'user_id'              => 0,
		'comment_approved'     => 1,
		'comment_date'         => '2024-05-01 10:00:00',
		'comment_date_gmt'     => '2024-05-01 10:00:00',
	);
	$args = array_merge( $defaults, $args );
	$id = wp_insert_comment( $args );
	if ( ! $id ) {
		WP_CLI::warning( 'comment insert failed' );
	}
	return (int) $id;
}

$c1 = gopress_fixture_comment(
	array(
		'comment_post_ID' => $post_ids[1],
		'comment_content' => 'Top-level comment.',
		'user_id'         => $users['subscriber'],
	)
);
$c2 = gopress_fixture_comment(
	array(
		'comment_post_ID' => $post_ids[1],
		'comment_content' => 'Reply level 2.',
		'comment_parent'  => $c1,
		'user_id'         => $users['author'],
		'comment_date'    => '2024-05-01 11:00:00',
		'comment_date_gmt'=> '2024-05-01 11:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID' => $post_ids[1],
		'comment_content' => 'Reply level 3.',
		'comment_parent'  => $c2,
		'user_id'         => $users['editor'],
		'comment_date'    => '2024-05-01 12:00:00',
		'comment_date_gmt'=> '2024-05-01 12:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID'  => $post_ids[1],
		'comment_content'  => 'Pending comment.',
		'comment_approved' => 0,
		'comment_date'     => '2024-05-02 10:00:00',
		'comment_date_gmt' => '2024-05-02 10:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID'  => $post_ids[1],
		'comment_content'  => 'Spam comment.',
		'comment_approved' => 'spam',
		'comment_date'     => '2024-05-02 11:00:00',
		'comment_date_gmt' => '2024-05-02 11:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID'  => $post_ids[1],
		'comment_content'  => 'Trash comment.',
		'comment_approved' => 'trash',
		'comment_date'     => '2024-05-02 12:00:00',
		'comment_date_gmt' => '2024-05-02 12:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID'      => $post_ids[1],
		'comment_type'         => 'pingback',
		'comment_author'       => 'Ping Source',
		'comment_author_url'   => 'https://example.com/ping',
		'comment_content'      => 'Pingback body.',
		'comment_date'         => '2024-05-03 10:00:00',
		'comment_date_gmt'     => '2024-05-03 10:00:00',
	)
);
gopress_fixture_comment(
	array(
		'comment_post_ID'      => $post_ids[1],
		'comment_author'       => 'Anonymous Guest',
		'comment_author_email' => 'anon@example.com',
		'comment_author_url'   => 'https://example.com/anon',
		'comment_content'      => 'Anonymous with URL.',
		'comment_date'         => '2024-05-03 11:00:00',
		'comment_date_gmt'     => '2024-05-03 11:00:00',
	)
);

// --------------------------------------------------------------------------
// Site editor: patterns, template, part, global styles, navigation
// --------------------------------------------------------------------------
$synced = wp_insert_post(
	array(
		'post_type'    => 'wp_block',
		'post_title'   => 'Fixture Synced Pattern',
		'post_name'    => 'fixture-synced-pattern',
		'post_content' => '<!-- wp:paragraph --><p>Synced pattern</p><!-- /wp:paragraph -->',
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-10 10:00:00',
		'post_date_gmt'=> '2024-05-10 10:00:00',
		'meta_input'   => array( 'wp_pattern_sync_status' => 'fully' ),
	),
	true
);
$unsynced = wp_insert_post(
	array(
		'post_type'    => 'wp_block',
		'post_title'   => 'Fixture Unsynced Pattern',
		'post_name'    => 'fixture-unsynced-pattern',
		'post_content' => '<!-- wp:paragraph --><p>Unsynced pattern</p><!-- /wp:paragraph -->',
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-10 11:00:00',
		'post_date_gmt'=> '2024-05-10 11:00:00',
		'meta_input'   => array( 'wp_pattern_sync_status' => 'unsynced' ),
	),
	true
);
$pattern_cat = gopress_ensure_term( 'Fixture Patterns', 'wp_pattern_category', array( 'slug' => 'fixture-patterns' ) );
if ( ! is_wp_error( $synced ) ) {
	wp_set_object_terms( $synced, array( $pattern_cat ), 'wp_pattern_category' );
}
if ( ! is_wp_error( $unsynced ) ) {
	wp_set_object_terms( $unsynced, array( $pattern_cat ), 'wp_pattern_category' );
}

$theme = wp_get_theme();
$stylesheet = $theme->get_stylesheet();

$index_tpl = wp_insert_post(
	array(
		'post_type'    => 'wp_template',
		'post_title'   => 'Index',
		'post_name'    => 'index',
		'post_content' => '<!-- wp:group {"tagName":"main"} --><main class="wp-block-group"><!-- wp:paragraph --><p>Customized index template</p><!-- /wp:paragraph --><!-- wp:query --><!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template --><!-- /wp:query --></main><!-- /wp:group -->',
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-11 10:00:00',
		'post_date_gmt'=> '2024-05-11 10:00:00',
	),
	true
);
if ( ! is_wp_error( $index_tpl ) ) {
	wp_set_object_terms( $index_tpl, $stylesheet, 'wp_theme' );
}

$header_part = wp_insert_post(
	array(
		'post_type'    => 'wp_template_part',
		'post_title'   => 'Header',
		'post_name'    => 'header',
		'post_content' => '<!-- wp:paragraph --><p>Customized header part</p><!-- /wp:paragraph -->',
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-11 11:00:00',
		'post_date_gmt'=> '2024-05-11 11:00:00',
	),
	true
);
if ( ! is_wp_error( $header_part ) ) {
	wp_set_object_terms( $header_part, $stylesheet, 'wp_theme' );
	wp_set_object_terms( $header_part, 'header', 'wp_template_part_area' );
}

$global_styles = wp_insert_post(
	array(
		'post_type'    => 'wp_global_styles',
		'post_title'   => 'Custom Styles',
		'post_name'    => 'wp-global-styles-' . $stylesheet,
		'post_content' => wp_json_encode(
			array(
				'version'                     => 3,
				'isGlobalStylesUserThemeJSON' => true,
				'settings'                    => array(
					'color' => array(
						'palette' => array(
							'theme' => array(
								array(
									'slug'  => 'fixture-accent',
									'color' => '#1266aa',
									'name'  => 'Fixture Accent',
								),
							),
						),
					),
				),
				'styles'                      => array(
					'color' => array(
						'background' => '#f7f7f7',
					),
				),
			)
		),
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-11 12:00:00',
		'post_date_gmt'=> '2024-05-11 12:00:00',
	),
	true
);
if ( ! is_wp_error( $global_styles ) ) {
	wp_set_object_terms( $global_styles, $stylesheet, 'wp_theme' );
}

$nav = wp_insert_post(
	array(
		'post_type'    => 'wp_navigation',
		'post_title'   => 'Fixture Navigation',
		'post_name'    => 'fixture-navigation',
		'post_content' => '<!-- wp:navigation-link {"label":"Home","type":"custom","url":"/"} /-->',
		'post_status'  => 'publish',
		'post_author'  => $users['admin'],
		'post_date'    => '2024-05-12 10:00:00',
		'post_date_gmt'=> '2024-05-12 10:00:00',
	),
	true
);

// Classic nav menu with 2 levels.
$menu_name = 'Fixture Classic Menu';
$menu = wp_get_nav_menu_object( $menu_name );
if ( ! $menu ) {
	$menu_id = wp_create_nav_menu( $menu_name );
} else {
	$menu_id = (int) $menu->term_id;
}
// Clear existing items for determinism.
$items = wp_get_nav_menu_items( $menu_id );
if ( $items ) {
	foreach ( $items as $item ) {
		wp_delete_post( $item->ID, true );
	}
}
$parent_item = wp_update_nav_menu_item(
	$menu_id,
	0,
	array(
		'menu-item-title'  => 'Parent Link',
		'menu-item-url'    => home_url( '/' ),
		'menu-item-status' => 'publish',
		'menu-item-type'   => 'custom',
	)
);
wp_update_nav_menu_item(
	$menu_id,
	0,
	array(
		'menu-item-title'     => 'Child Link',
		'menu-item-url'       => home_url( '/sample-page/' ),
		'menu-item-status'    => 'publish',
		'menu-item-type'      => 'custom',
		'menu-item-parent-id' => $parent_item,
	)
);

// --------------------------------------------------------------------------
// Determinism: freeze every "now" timestamp and other volatile fields
// --------------------------------------------------------------------------
global $wpdb;
$fixed     = GOPRESS_FIXTURE_NOW;
$fixed_gmt = GOPRESS_FIXTURE_NOW_GMT;
$fixed_ts  = strtotime( '2024-06-15 12:00:00 UTC' );

$wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->posts} SET post_modified = %s, post_modified_gmt = %s", $fixed, $fixed_gmt ) );

// Default install content, revisions, nav items, and anything stamped with "today"
// still carry wall-clock post_date values. Keep intentionally dated fixture posts
// (2024-*) and future posts (2026-12-*). Rewrite only "today"/install stamps.
$wpdb->query(
	$wpdb->prepare(
		"UPDATE {$wpdb->posts}
		SET post_date = %s, post_date_gmt = %s
		WHERE post_date >= %s
		  AND post_status <> 'future'",
		$fixed,
		$fixed_gmt,
		gmdate( 'Y-m-d 00:00:00' )
	)
);
$wpdb->query(
	$wpdb->prepare(
		"UPDATE {$wpdb->comments}
		SET comment_date = %s, comment_date_gmt = %s
		WHERE comment_date >= %s",
		$fixed,
		$fixed_gmt,
		gmdate( 'Y-m-d 00:00:00' )
	)
);

// Drop CLI login sessions created by --user=admin.
$wpdb->query( "DELETE FROM {$wpdb->usermeta} WHERE meta_key = 'session_tokens'" );

// Fixed application password row (uuid + hash must not change between runs).
update_user_meta(
	$users['admin'],
	WP_Application_Passwords::USERMETA_KEY_APPLICATION_PASSWORDS,
	array(
		array(
			'uuid'      => '00000000-0000-4000-8000-000000000001',
			'app_id'    => 'gopress-fixture',
			'name'      => 'GoPress Fixture',
			// wp_fast_hash of the literal password "passwordpasswordpassword" is not required;
			// tests that need a working app password can recreate one. Determinism matters here.
			'password'  => '$generic$gopressFixtureAppPassHash00000000000001',
			'created'   => strtotime( '2024-01-01 10:00:00 UTC' ),
			'last_used' => null,
			'last_ip'   => null,
		),
	)
);

// Normalize _edit_lock.
update_post_meta( $post_ids[2], '_edit_lock', $fixed_ts . ':' . $users['editor'] );

// Rewrite cron to fixed timestamps (keep structure / future-post args).
$cron = get_option( 'cron' );
if ( is_array( $cron ) ) {
	$rebuilt = array( 'version' => isset( $cron['version'] ) ? $cron['version'] : 2 );
	$slot    = $fixed_ts;
	foreach ( $cron as $timestamp => $hooks ) {
		if ( 'version' === $timestamp || ! is_array( $hooks ) ) {
			continue;
		}
		// Preserve absolute future-post schedule at its real timestamp.
		if ( isset( $hooks['publish_future_post'] ) ) {
			$rebuilt[ strtotime( '2026-12-01 10:00:00 UTC' ) ]['publish_future_post'] = $hooks['publish_future_post'];
			continue;
		}
		$rebuilt[ $slot ] = $hooks;
		$slot++;
	}
	update_option( 'cron', $rebuilt, true );
}

update_option( 'admin_email_lifespan', (string) ( $fixed_ts + ( 6 * MONTH_IN_SECONDS ) ), true );

// Delete volatile site/theme file transients (timeouts differ by seconds).
$wpdb->query( "DELETE FROM {$wpdb->options} WHERE option_name LIKE '\\_site\\_transient\\_%' OR option_name LIKE '\\_transient\\_%'" );

update_option( 'gopress_fixture_seeded', '1' );
update_option( 'gopress_fixture_seeded_at', $fixed );

// Stable user_activation_key / leftover randomness.
$wpdb->query( "UPDATE {$wpdb->users} SET user_activation_key = ''" );

// bcrypt salts differ every call to wp_hash_password; pin one verified hash for "password".
$fixed_pass_hash = '$wp$2y$10$6w0zN/qvwp7cHDgzOIfBPuqucz7bjufoiKA/CAk0RUJ3C3XA3YE/y';
$wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->users} SET user_pass = %s", $fixed_pass_hash ) );

WP_CLI::success( 'Fixture seed complete.' );
