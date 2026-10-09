<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: get_registered_settings().
 *
 * @package GoPress
 */

if ( ! function_exists( 'get_registered_settings' ) ) {
	WP_CLI::warning( 'get_registered_settings missing' );
	gopress_golden_write(
		'registered-settings.json',
		array( 'skipped' => true, 'reason' => 'get_registered_settings missing' )
	);
} else {
	// Ensure settings are registered (REST / admin typically register on init/rest_api_init).
	do_action( 'rest_api_init' );
	$settings = get_registered_settings();
	gopress_golden_write( 'registered-settings.json', gopress_golden_public_props( $settings ) );
}

WP_CLI::success( 'registered-settings dump complete' );
