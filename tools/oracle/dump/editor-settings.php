<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: block editor settings for post and site contexts.
 *
 * @package GoPress
 */

if ( ! function_exists( 'get_block_editor_settings' ) ) {
	// Load editor bootstrap if needed.
	if ( file_exists( ABSPATH . 'wp-admin/includes/post.php' ) ) {
		require_once ABSPATH . 'wp-admin/includes/post.php';
	}
	if ( file_exists( ABSPATH . 'wp-includes/block-editor.php' ) ) {
		require_once ABSPATH . 'wp-includes/block-editor.php';
	}
}

$admins = get_users( array( 'role' => 'administrator', 'number' => 1 ) );
if ( ! empty( $admins ) ) {
	wp_set_current_user( $admins[0]->ID );
}

// Dummy / existing post for post editor context.
$post = get_post( 1 );
if ( ! $post ) {
	$post_id = wp_insert_post(
		array(
			'post_title'   => 'GoPress Golden Dummy',
			'post_content' => '<!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph -->',
			'post_status'  => 'draft',
			'post_type'    => 'post',
			'post_author'  => get_current_user_id() ? get_current_user_id() : 1,
		),
		true
	);
	if ( is_wp_error( $post_id ) ) {
		WP_CLI::warning( 'Could not create dummy post: ' . $post_id->get_error_message() );
		$post = null;
	} else {
		$post = get_post( $post_id );
	}
}

if ( function_exists( 'get_block_editor_settings' ) ) {
	$editor_context_class = class_exists( 'WP_Block_Editor_Context' );

	if ( $post && $editor_context_class ) {
		$ctx  = new WP_Block_Editor_Context( array( 'post' => $post ) );
		$settings = get_block_editor_settings( array(), $ctx );
		gopress_golden_write( 'editor-settings/post.json', gopress_golden_public_props( $settings ) );
	} elseif ( $post ) {
		$settings = get_block_editor_settings( array(), $post );
		gopress_golden_write( 'editor-settings/post.json', gopress_golden_public_props( $settings ) );
	} else {
		WP_CLI::warning( 'No post for editor settings; writing stub' );
		gopress_golden_write(
			'editor-settings/post.json',
			array( 'skipped' => true, 'reason' => 'no post available' )
		);
	}

	// Site editor context.
	if ( $editor_context_class ) {
		$site_ctx = new WP_Block_Editor_Context( array( 'name' => 'core/edit-site' ) );
		$site_settings = get_block_editor_settings( array(), $site_ctx );
		gopress_golden_write( 'editor-settings/site.json', gopress_golden_public_props( $site_settings ) );
	} else {
		WP_CLI::warning( 'WP_Block_Editor_Context missing for site settings' );
		gopress_golden_write(
			'editor-settings/site.json',
			array( 'skipped' => true, 'reason' => 'WP_Block_Editor_Context missing' )
		);
	}
} else {
	WP_CLI::warning( 'get_block_editor_settings missing; writing stubs' );
	gopress_golden_write(
		'editor-settings/post.json',
		array( 'skipped' => true, 'reason' => 'get_block_editor_settings missing' )
	);
	gopress_golden_write(
		'editor-settings/site.json',
		array( 'skipped' => true, 'reason' => 'get_block_editor_settings missing' )
	);
}

WP_CLI::success( 'editor-settings dump complete' );
