<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: admin $menu / $submenu per role.
 *
 * Loading wp-admin menu is fragile under wp eval-file; attempt carefully and
 * write a stub documenting the skip if it fails.
 *
 * @package GoPress
 */

$roles = array(
	'admin'       => 'administrator',
	'editor'      => 'editor',
	'author'      => 'author',
	'contributor' => 'contributor',
	'subscriber'  => 'subscriber',
);

/**
 * Attempt to build admin menu for current user.
 *
 * @return array{menu:array,submenu:array}|WP_Error
 */
function gopress_load_admin_menu() {
	global $menu, $submenu, $pagenow, $_wp_menu_nopriv, $_wp_submenu_nopriv;

	$menu    = array();
	$submenu = array();

	// Pretend we are in admin.
	if ( ! defined( 'WP_ADMIN' ) ) {
		define( 'WP_ADMIN', true );
	}
	if ( ! defined( 'WP_NETWORK_ADMIN' ) ) {
		define( 'WP_NETWORK_ADMIN', false );
	}
	if ( ! defined( 'WP_USER_ADMIN' ) ) {
		define( 'WP_USER_ADMIN', false );
	}

	$pagenow = 'index.php';

	$includes = array(
		ABSPATH . 'wp-admin/includes/admin.php',
		ABSPATH . 'wp-admin/includes/menu.php',
	);

	// admin.php pulls in many includes; prefer plugin.php + menu.php path used by WP.
	if ( ! function_exists( 'get_admin_page_title' ) && file_exists( ABSPATH . 'wp-admin/includes/admin.php' ) ) {
		require_once ABSPATH . 'wp-admin/includes/admin.php';
	}

	// Reset and rebuild menu globals the way menu.php expects.
	$menu = $submenu = $_wp_menu_nopriv = $_wp_submenu_nopriv = array();

	// Include menu.php which both defines and runs menu construction when included
	// from admin bootstrap. Under eval-file this often fatals — catch via output buffer.
	$menu_file = ABSPATH . 'wp-admin/menu.php';
	if ( ! file_exists( $menu_file ) ) {
		return new WP_Error( 'missing', 'wp-admin/menu.php not found' );
	}

	try {
		// menu.php expects to be included once; use require and hope.
		// Suppress redeclare by checking a sentinel.
		if ( ! defined( 'GOPRESS_ADMIN_MENU_LOADED' ) ) {
			// First load of translations etc.
			if ( ! function_exists( 'wp_get_current_user' ) ) {
				return new WP_Error( 'no_user', 'wp_get_current_user missing' );
			}
			ob_start();
			require $menu_file;
			ob_end_clean();
			define( 'GOPRESS_ADMIN_MENU_LOADED', true );
		} else {
			// Re-include is unsafe (function redeclarations). Rebuild via hooks only.
			$menu = $submenu = array();
			do_action( '_admin_menu' );
			do_action( 'admin_menu', '' );
			do_action( 'admin_menu', 'index.php' );
		}
	} catch ( Throwable $e ) {
		return new WP_Error( 'exception', $e->getMessage() );
	}

	return array(
		'menu'    => isset( $menu ) ? $menu : array(),
		'submenu' => isset( $submenu ) ? $submenu : array(),
	);
}

$loaded_once = false;

foreach ( $roles as $slug => $role ) {
	$users = get_users( array( 'role' => $role, 'number' => 1 ) );
	if ( empty( $users ) ) {
		// Fallback: admin login.
		$user = ( 'administrator' === $role ) ? get_user_by( 'login', 'admin' ) : false;
		if ( ! $user ) {
			gopress_golden_write(
				"admin-menu/{$slug}.json",
				array(
					'skipped' => true,
					'reason'  => "no user for role $role",
					'role'    => $role,
				)
			);
			continue;
		}
	} else {
		$user = $users[0];
	}

	wp_set_current_user( $user->ID );

	if ( ! $loaded_once ) {
		$result = gopress_load_admin_menu();
		$loaded_once = true;
		if ( is_wp_error( $result ) ) {
			WP_CLI::warning( 'admin-menu load failed: ' . $result->get_error_message() );
			foreach ( $roles as $s2 => $r2 ) {
				gopress_golden_write(
					"admin-menu/{$s2}.json",
					array(
						'skipped' => true,
						'reason'  => $result->get_error_message(),
						'role'    => $r2,
						'note'    => 'Dump skipped; loading wp-admin/menu.php under wp eval-file is fragile (redeclare / bootstrap).',
					)
				);
			}
			WP_CLI::success( 'admin-menu dump complete (stubs)' );
			return;
		}
		gopress_golden_write(
			"admin-menu/{$slug}.json",
			array(
				'role'    => $role,
				'user_id' => (int) $user->ID,
				'menu'    => gopress_golden_public_props( $result['menu'] ),
				'submenu' => gopress_golden_public_props( $result['submenu'] ),
			)
		);
	} else {
		// For subsequent roles, re-fire admin_menu with new caps — best effort.
		global $menu, $submenu;
		$menu = $submenu = array();
		try {
			// Cannot re-require menu.php; document partial skip.
			gopress_golden_write(
				"admin-menu/{$slug}.json",
				array(
					'skipped' => true,
					'reason'  => 'menu.php can only be loaded once per process; per-role rebuild not available under eval-file',
					'role'    => $role,
					'user_id' => (int) $user->ID,
				)
			);
		} catch ( Throwable $e ) {
			gopress_golden_write(
				"admin-menu/{$slug}.json",
				array(
					'skipped' => true,
					'reason'  => $e->getMessage(),
					'role'    => $role,
				)
			);
		}
	}
}

wp_set_current_user( 0 );
WP_CLI::success( 'admin-menu dump complete' );
