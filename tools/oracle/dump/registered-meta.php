<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: registered meta keys for post/user/term/comment.
 *
 * @package GoPress
 */

$out = array();

if ( ! function_exists( 'get_registered_meta_keys' ) ) {
	WP_CLI::warning( 'get_registered_meta_keys missing' );
	gopress_golden_write(
		'registered-meta.json',
		array( 'skipped' => true, 'reason' => 'get_registered_meta_keys missing' )
	);
	WP_CLI::success( 'registered-meta dump complete (skipped)' );
	return;
}

foreach ( array( 'post', 'user', 'term', 'comment' ) as $object_type ) {
	$out[ $object_type ] = array();
	// Empty subtype → all for that type.
	$keys = get_registered_meta_keys( $object_type );
	$out[ $object_type ][''] = gopress_golden_public_props( $keys );

	// Common subtypes.
	$subtypes = array();
	if ( 'post' === $object_type ) {
		$subtypes = array( 'post', 'page', 'attachment' );
	} elseif ( 'term' === $object_type ) {
		$subtypes = array( 'category', 'post_tag' );
	}
	foreach ( $subtypes as $subtype ) {
		$out[ $object_type ][ $subtype ] = gopress_golden_public_props(
			get_registered_meta_keys( $object_type, $subtype )
		);
	}
}

gopress_golden_write( 'registered-meta.json', $out );
WP_CLI::success( 'registered-meta dump complete' );
