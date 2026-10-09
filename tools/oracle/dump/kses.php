<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: KSES allowed HTML / protocols / style CSS + vectors.
 *
 * @package GoPress
 */

$allowed = array();

if ( function_exists( 'wp_kses_allowed_html' ) ) {
	foreach ( array( 'post', 'data', 'strip', 'entities' ) as $context ) {
		$allowed['allowed_html'][ $context ] = wp_kses_allowed_html( $context );
	}
} else {
	WP_CLI::warning( 'wp_kses_allowed_html missing' );
}

if ( function_exists( 'wp_allowed_protocols' ) ) {
	$allowed['protocols'] = wp_allowed_protocols();
} else {
	WP_CLI::warning( 'wp_allowed_protocols missing' );
}

if ( function_exists( 'apply_filters' ) ) {
	$allowed['safe_style_css'] = apply_filters( 'safe_style_css', array() );
} else {
	WP_CLI::warning( 'apply_filters missing' );
}

gopress_golden_write( 'kses/allowed.json', $allowed );

$corpus  = gopress_golden_corpus_strings();
$vectors = array(
	'corpus_size' => count( $corpus ),
	'items'       => array(),
);

foreach ( $corpus as $i => $s ) {
	$item = array(
		'i'     => $i,
		'input' => $s,
	);
	if ( function_exists( 'wp_kses_post' ) ) {
		$item['wp_kses_post'] = wp_kses_post( $s );
	} else {
		WP_CLI::warning( 'wp_kses_post missing' );
		break;
	}
	if ( function_exists( 'wp_kses_data' ) ) {
		$item['wp_kses_data'] = wp_kses_data( $s );
	}
	$vectors['items'][] = $item;
}

gopress_golden_write( 'kses/vectors.json', $vectors );
WP_CLI::success( 'kses dump complete' );
