<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: wp_user_roles option.
 *
 * @package GoPress
 */

$roles = get_option( 'wp_user_roles' );
if ( false === $roles ) {
	// Fallback via WP_Roles.
	if ( class_exists( 'WP_Roles' ) ) {
		$wp_roles = wp_roles();
		$roles    = $wp_roles->roles;
	} else {
		WP_CLI::error( 'wp_user_roles unavailable' );
	}
}

gopress_golden_write( 'roles.json', $roles );
WP_CLI::success( 'roles dump complete' );
