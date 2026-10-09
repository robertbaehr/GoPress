<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: taxonomies.
 *
 * @package GoPress
 */

if ( ! function_exists( 'get_taxonomies' ) ) {
	WP_CLI::error( 'get_taxonomies missing' );
}

$taxes = get_taxonomies( array(), 'objects' );
$out   = array();
foreach ( $taxes as $name => $obj ) {
	$out[ $name ] = gopress_golden_public_props( $obj );
}

gopress_golden_write( 'taxonomies.json', $out );
WP_CLI::success( 'taxonomies dump complete' );
