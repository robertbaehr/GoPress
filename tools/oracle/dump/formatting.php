<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: WP formatting / escaping / sanitization over corpus.
 *
 * @package GoPress
 */

$corpus = gopress_golden_corpus_strings();
// Cap at 40 for manageable golden size while staying representative.
$corpus = array_slice( $corpus, 0, 40 );

$fns = array(
	'esc_html',
	'esc_attr',
	'esc_url',
	'esc_js',
	'esc_textarea',
	'sanitize_title',
	'sanitize_title_with_dashes',
	'sanitize_key',
	'sanitize_text_field',
	'sanitize_file_name',
	'sanitize_user',
	'sanitize_email',
	'is_email',
	'remove_accents',
	'wpautop',
	'wptexturize',
	'force_balance_tags',
	'wp_trim_words',
	'wp_html_excerpt',
	'wp_strip_all_tags',
	'normalize_whitespace',
	'_wp_to_kebab_case',
	'zeroise',
	'size_format',
	'human_time_diff',
);

$out = array(
	'corpus_size' => count( $corpus ),
	'vectors'     => array(),
);

foreach ( $corpus as $i => $s ) {
	$row = array(
		'i'     => $i,
		'input' => $s,
	);

	foreach ( array(
		'esc_html',
		'esc_attr',
		'esc_url',
		'esc_js',
		'esc_textarea',
		'sanitize_title',
		'sanitize_title_with_dashes',
		'sanitize_key',
		'sanitize_text_field',
		'sanitize_file_name',
		'sanitize_email',
		'remove_accents',
		'wpautop',
		'wptexturize',
		'force_balance_tags',
		'wp_strip_all_tags',
		'normalize_whitespace',
	) as $fn ) {
		if ( ! function_exists( $fn ) ) {
			continue;
		}
		$row[ $fn ] = call_user_func( $fn, $s );
	}

	if ( function_exists( 'sanitize_user' ) ) {
		$row['sanitize_user'] = sanitize_user( $s, true );
	}
	if ( function_exists( 'is_email' ) ) {
		$row['is_email'] = is_email( $s );
	}
	if ( function_exists( 'wp_trim_words' ) ) {
		$row['wp_trim_words'] = wp_trim_words( $s, 5, '&hellip;' );
	}
	if ( function_exists( 'wp_html_excerpt' ) ) {
		$row['wp_html_excerpt'] = wp_html_excerpt( $s, 40, '&hellip;' );
	}
	if ( function_exists( '_wp_to_kebab_case' ) ) {
		$row['_wp_to_kebab_case'] = _wp_to_kebab_case( $s );
	}

	// Query-arg helpers: treat input as URL when it looks like one, else base.
	$base_url = ( is_string( $s ) && preg_match( '#^https?://#i', $s ) ) ? $s : 'https://example.com/path';
	if ( function_exists( 'add_query_arg' ) ) {
		$row['add_query_arg'] = add_query_arg( 'x', '1', $base_url );
	}
	if ( function_exists( 'remove_query_arg' ) ) {
		$row['remove_query_arg'] = remove_query_arg( 'x', add_query_arg( 'x', '1', $base_url ) );
	}

	$out['vectors'][] = $row;
}

// Extra non-corpus samples for numeric helpers.
if ( function_exists( 'zeroise' ) ) {
	$out['zeroise'] = array();
	foreach ( array( 0, 1, 9, 10, 99, 100 ) as $n ) {
		$out['zeroise'][] = array( 'n' => $n, 'result' => zeroise( $n, 3 ) );
	}
} else {
	WP_CLI::warning( 'zeroise missing' );
}

if ( function_exists( 'size_format' ) ) {
	$out['size_format'] = array();
	foreach ( array( 0, 512, 1024, 1048576, 1073741824 ) as $bytes ) {
		$out['size_format'][] = array( 'bytes' => $bytes, 'result' => size_format( $bytes, 2 ) );
	}
} else {
	WP_CLI::warning( 'size_format missing' );
}

if ( function_exists( 'human_time_diff' ) ) {
	$out['human_time_diff'] = array();
	$from = 1609459200;
	foreach ( array( 60, 3600, 86400, 604800, 2592000, 31536000 ) as $delta ) {
		$out['human_time_diff'][] = array(
			'from'   => $from,
			'to'     => $from + $delta,
			'result' => human_time_diff( $from, $from + $delta ),
		);
	}
} else {
	WP_CLI::warning( 'human_time_diff missing' );
}

// Report skipped formatting fns.
foreach ( $fns as $fn ) {
	if ( ! function_exists( $fn ) ) {
		WP_CLI::warning( "formatting fn missing: $fn" );
	}
}

gopress_golden_write( 'formatting/vectors.json', $out );
WP_CLI::success( 'formatting dump complete' );
