<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: theme.json-derived global settings/styles/stylesheet per theme.
 *
 * @package GoPress
 */

$original = get_stylesheet();

$themes = array();
if ( function_exists( 'wp_get_themes' ) ) {
	foreach ( wp_get_themes() as $slug => $theme ) {
		$themes[] = $slug;
	}
} else {
	$themes = array( 'twentytwentyfive' );
}

// Prefer bundled defaults first.
$preferred = array( 'twentytwentyfive', 'twentytwentyfour', 'twentytwentythree', 'twentytwentytwo', 'twentytwentyone', 'twentytwenty', 'twentynineteen' );
$ordered   = array();
foreach ( $preferred as $p ) {
	if ( in_array( $p, $themes, true ) ) {
		$ordered[] = $p;
	}
}
foreach ( $themes as $t ) {
	if ( ! in_array( $t, $ordered, true ) ) {
		$ordered[] = $t;
	}
}

foreach ( $ordered as $slug ) {
	try {
		switch_theme( $slug );
	} catch ( Exception $e ) {
		WP_CLI::warning( "switch_theme($slug) failed: " . $e->getMessage() );
		continue;
	}

	// Clear caches so globals refresh.
	if ( class_exists( 'WP_Theme_JSON_Resolver' ) ) {
		WP_Theme_JSON_Resolver::clean_cached_data();
	}

	$data = array(
		'theme' => $slug,
	);

	if ( function_exists( 'wp_get_global_settings' ) ) {
		$data['settings'] = wp_get_global_settings();
	} else {
		WP_CLI::warning( 'wp_get_global_settings missing' );
		$data['settings'] = null;
	}

	if ( function_exists( 'wp_get_global_styles' ) ) {
		$data['styles'] = wp_get_global_styles();
	} else {
		WP_CLI::warning( 'wp_get_global_styles missing' );
		$data['styles'] = null;
	}

	if ( function_exists( 'wp_get_global_stylesheet' ) ) {
		$css = wp_get_global_stylesheet();
		$data['stylesheet'] = array(
			'length'     => strlen( $css ),
			'sha256'     => hash( 'sha256', $css ),
			'first_200'  => substr( $css, 0, 200 ),
			'last_200'   => substr( $css, -200 ),
		);
	} else {
		WP_CLI::warning( 'wp_get_global_stylesheet missing' );
		$data['stylesheet'] = null;
	}

	gopress_golden_write( "theme-json/{$slug}.json", $data );
}

// Restore.
switch_theme( $original );
if ( class_exists( 'WP_Theme_JSON_Resolver' ) ) {
	WP_Theme_JSON_Resolver::clean_cached_data();
}

WP_CLI::success( 'theme-json dump complete' );
