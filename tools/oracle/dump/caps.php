<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: capability matrix per role.
 *
 * @package GoPress
 */

$capabilities = array(
	'read',
	'edit_posts',
	'publish_posts',
	'delete_posts',
	'edit_others_posts',
	'edit_private_posts',
	'edit_published_posts',
	'delete_others_posts',
	'delete_private_posts',
	'delete_published_posts',
	'read_private_posts',
	'moderate_comments',
	'manage_categories',
	'manage_links',
	'manage_options',
	'upload_files',
	'edit_theme_options',
	'install_plugins',
	'update_plugins',
	'delete_plugins',
	'install_themes',
	'update_themes',
	'delete_themes',
	'update_core',
	'customize',
	'edit_users',
	'list_users',
	'create_users',
	'delete_users',
	'promote_users',
	'remove_users',
	'unfiltered_html',
	'edit_pages',
	'edit_others_pages',
	'edit_published_pages',
	'publish_pages',
	'delete_pages',
	'delete_others_pages',
	'delete_published_pages',
	'read_private_pages',
	'edit_dashboard',
	'export',
	'import',
	'level_0',
	'level_1',
	'level_2',
	'level_7',
	'level_10',
);

$role_logins = array(
	'administrator' => 'admin',
	'editor'        => 'editor',
	'author'        => 'author',
	'contributor'   => 'contributor',
	'subscriber'    => 'subscriber',
);

// Resolve users by role; fall back to creating capability checks via WP_User with role.
foreach ( $role_logins as $role => $login ) {
	$user = get_user_by( 'login', $login );
	if ( ! $user ) {
		// Find any user with this role.
		$users = get_users(
			array(
				'role'   => $role,
				'number' => 1,
			)
		);
		if ( ! empty( $users ) ) {
			$user = $users[0];
		}
	}

	if ( ! $user ) {
		// Synthesize a user object with the role for cap checks.
		$user = new WP_User( 0 );
		$user->set_role( $role );
		// set_role on user 0 may not work; use WP_Roles directly.
		$wp_roles = wp_roles();
		$caps_map = array();
		foreach ( $capabilities as $cap ) {
			$caps_map[ $cap ] = user_can( $user, $cap );
		}
		// Better: check role object.
		$role_obj = $wp_roles->get_role( $role );
		$caps_map = array();
		foreach ( $capabilities as $cap ) {
			$has = false;
			if ( $role_obj ) {
				$has = ! empty( $role_obj->capabilities[ $cap ] );
			}
			$caps_map[ $cap ] = (bool) $has;
		}
		$slug = ( 'administrator' === $role ) ? 'admin' : $role;
		gopress_golden_write( "caps/{$slug}.json", array(
			'role'         => $role,
			'user_id'      => 0,
			'note'         => 'No user with role; used role object capabilities',
			'capabilities' => $caps_map,
		) );
		continue;
	}

	wp_set_current_user( $user->ID );
	$caps_map = array();
	foreach ( $capabilities as $cap ) {
		$caps_map[ $cap ] = (bool) current_user_can( $cap );
	}

	$slug = ( 'administrator' === $role ) ? 'admin' : $role;
	gopress_golden_write(
		"caps/{$slug}.json",
		array(
			'role'         => $role,
			'user_id'      => (int) $user->ID,
			'login'        => $user->user_login,
			'capabilities' => $caps_map,
		)
	);
}

wp_set_current_user( 0 );
WP_CLI::success( 'caps dump complete' );
