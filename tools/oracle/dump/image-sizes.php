<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: registered image subsizes.
 *
 * @package GoPress
 */

if ( ! function_exists( 'wp_get_registered_image_subsizes' ) ) {
	WP_CLI::warning( 'wp_get_registered_image_subsizes missing' );
	gopress_golden_write(
		'image-sizes.json',
		array( 'skipped' => true, 'reason' => 'wp_get_registered_image_subsizes missing' )
	);
} else {
	$sizes = wp_get_registered_image_subsizes();
	ksort( $sizes );
	gopress_golden_write( 'image-sizes.json', $sizes );
}

WP_CLI::success( 'image-sizes dump complete' );
