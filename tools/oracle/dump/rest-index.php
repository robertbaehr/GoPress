<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: REST API index for admin and anonymous users.
 *
 * @package GoPress
 */

/**
 * Dispatch GET / on the REST server and reduce the response.
 *
 * @return array
 */
function gopress_rest_index_dump() {
	$server = rest_get_server();
	$request = new WP_REST_Request( 'GET', '/' );
	$response = $server->dispatch( $request );
	$data     = $response->get_data();

	// Reduce: keep namespaces, routes keys, site info; drop huge / volatile blobs.
	$reduced = array(
		'status'     => $response->get_status(),
		'name'       => isset( $data['name'] ) ? $data['name'] : null,
		'description'=> isset( $data['description'] ) ? $data['description'] : null,
		'url'        => isset( $data['url'] ) ? $data['url'] : null,
		'home'       => isset( $data['home'] ) ? $data['home'] : null,
		'namespaces' => isset( $data['namespaces'] ) ? $data['namespaces'] : array(),
		'authentication' => isset( $data['authentication'] ) ? $data['authentication'] : array(),
		'routes'     => array(),
	);

	if ( isset( $data['routes'] ) && is_array( $data['routes'] ) ) {
		foreach ( $data['routes'] as $route => $info ) {
			$reduced['routes'][ $route ] = array(
				'namespace' => isset( $info['namespace'] ) ? $info['namespace'] : null,
				'methods'   => isset( $info['methods'] ) ? $info['methods'] : array(),
				'endpoints' => isset( $info['endpoints'] ) ? gopress_golden_public_props( $info['endpoints'] ) : array(),
			);
		}
		ksort( $reduced['routes'] );
	}

	return $reduced;
}

// Admin index.
$admins = get_users( array( 'role' => 'administrator', 'number' => 1 ) );
if ( ! empty( $admins ) ) {
	wp_set_current_user( $admins[0]->ID );
} else {
	$user = get_user_by( 'login', 'admin' );
	if ( $user ) {
		wp_set_current_user( $user->ID );
	}
}

if ( function_exists( 'rest_get_server' ) ) {
	gopress_golden_write( 'rest/index-admin.json', gopress_rest_index_dump() );

	wp_set_current_user( 0 );
	gopress_golden_write( 'rest/index-anon.json', gopress_rest_index_dump() );
} else {
	WP_CLI::warning( 'rest_get_server missing' );
	gopress_golden_write( 'rest/index-admin.json', array( 'skipped' => true ) );
	gopress_golden_write( 'rest/index-anon.json', array( 'skipped' => true ) );
}

WP_CLI::success( 'rest-index dump complete' );
