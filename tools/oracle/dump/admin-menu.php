<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: admin $menu / $submenu per role.
 *
 * wp-admin/menu.php can only be included once per process (function declarations).
 * When GOPRESS_ADMIN_MENU_ROLE is unset, this script re-invokes itself once per
 * role via a child `wp eval-file` process so each role gets a clean bootstrap.
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

$only = getenv( 'GOPRESS_ADMIN_MENU_ROLE' );

if ( ! $only ) {
	$wp_path = defined( 'ABSPATH' ) ? rtrim( ABSPATH, '/' ) : '/opt/gopress-ref';
	$self    = __FILE__;
	$url     = home_url( '/' );
	foreach ( array_keys( $roles ) as $slug ) {
		$cmd = sprintf(
			'GOPRESS_ADMIN_MENU_ROLE=%s GOPRESS_REPO=%s wp eval-file %s --path=%s --url=%s --user=admin',
			escapeshellarg( $slug ),
			escapeshellarg( getenv( 'GOPRESS_REPO' ) ?: '/workspace' ),
			escapeshellarg( $self ),
			escapeshellarg( $wp_path ),
			escapeshellarg( $url )
		);
		passthru( $cmd, $code );
		if ( 0 !== $code ) {
			WP_CLI::warning( "admin-menu child for $slug exited $code" );
			gopress_golden_write(
				"admin-menu/{$slug}.json",
				array(
					'skipped' => true,
					'reason'  => "child process exit $code",
					'role'    => $roles[ $slug ],
					'note'    => 'Dump skipped; loading wp-admin/menu.php under wp eval-file is fragile.',
				)
			);
		}
	}
	WP_CLI::success( 'admin-menu dump complete' );
	return;
}

if ( ! isset( $roles[ $only ] ) ) {
	WP_CLI::error( "Unknown role slug: $only" );
}

$role  = $roles[ $only ];
$users = get_users( array( 'role' => $role, 'number' => 1 ) );
if ( empty( $users ) ) {
	$user = ( 'administrator' === $role ) ? get_user_by( 'login', 'admin' ) : false;
	if ( ! $user ) {
		gopress_golden_write(
			"admin-menu/{$only}.json",
			array(
				'skipped' => true,
				'reason'  => "no user for role $role",
				'role'    => $role,
			)
		);
		WP_CLI::success( "admin-menu $only stubbed" );
		return;
	}
} else {
	$user = $users[0];
}

wp_set_current_user( $user->ID );

if ( ! defined( 'WP_ADMIN' ) ) {
	define( 'WP_ADMIN', true );
}
if ( ! defined( 'WP_NETWORK_ADMIN' ) ) {
	define( 'WP_NETWORK_ADMIN', false );
}
if ( ! defined( 'WP_USER_ADMIN' ) ) {
	define( 'WP_USER_ADMIN', false );
}

global $menu, $submenu, $pagenow, $_wp_menu_nopriv, $_wp_submenu_nopriv, $typenow;
$menu = $submenu = $_wp_menu_nopriv = $_wp_submenu_nopriv = array();
$pagenow = 'index.php';
$typenow = '';
// Critical: leave $plugin_page unset. An empty string makes isset($plugin_page)
// true and user_can_access_admin_page() returns false.
unset( $GLOBALS['plugin_page'] );

try {
	if ( ! function_exists( 'get_admin_page_title' ) && file_exists( ABSPATH . 'wp-admin/includes/admin.php' ) ) {
		require_once ABSPATH . 'wp-admin/includes/admin.php';
	}
	ob_start();
	require ABSPATH . 'wp-admin/menu.php';
	ob_end_clean();
} catch ( Throwable $e ) {
	gopress_golden_write(
		"admin-menu/{$only}.json",
		array(
			'skipped' => true,
			'reason'  => $e->getMessage(),
			'role'    => $role,
			'note'    => 'Dump skipped; loading wp-admin/menu.php under wp eval-file failed.',
		)
	);
	WP_CLI::warning( 'admin-menu load failed: ' . $e->getMessage() );
	return;
}

gopress_golden_write(
	"admin-menu/{$only}.json",
	array(
		'role'    => $role,
		'user_id' => (int) $user->ID,
		'menu'    => gopress_golden_public_props( $menu ),
		'submenu' => gopress_golden_public_props( $submenu ),
	)
);

WP_CLI::success( "admin-menu $only dump complete" );
