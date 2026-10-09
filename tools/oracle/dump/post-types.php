<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: post types and post statuses.
 *
 * @package GoPress
 */

if ( ! function_exists( 'get_post_types' ) ) {
	WP_CLI::error( 'get_post_types missing' );
}

$types = get_post_types( array(), 'objects' );
$types_out = array();
foreach ( $types as $name => $obj ) {
	$types_out[ $name ] = gopress_golden_public_props( $obj );
}
gopress_golden_write( 'post-types.json', $types_out );

if ( ! function_exists( 'get_post_stati' ) ) {
	WP_CLI::warning( 'get_post_stati missing' );
} else {
	$stati = get_post_stati( array(), 'objects' );
	$stati_out = array();
	foreach ( $stati as $name => $obj ) {
		$stati_out[ $name ] = gopress_golden_public_props( $obj );
	}
	gopress_golden_write( 'post-statuses.json', $stati_out );
}

WP_CLI::success( 'post-types dump complete' );
