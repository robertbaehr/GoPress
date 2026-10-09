<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: rewrite rules for common permalink structures + URL parse samples.
 *
 * @package GoPress
 */

global $wp_rewrite, $wp;

$structures = array(
	'plain'          => '',
	'day-name'       => '/%year%/%monthnum%/%day%/%postname%/',
	'month-name'     => '/%year%/%monthnum%/%postname%/',
	'numeric'        => '/archives/%post_id%',
	'postname'       => '/%postname%/',
	'category-name'  => '/blog/%category%/%postname%/',
);

$parse_urls = array(
	'/',
	'/sample-page/',
	'/hello-world/',
	'/2020/01/01/hello-world/',
	'/2020/01/hello-world/',
	'/archives/1',
	'/category/uncategorized/',
	'/tag/test/',
	'/author/admin/',
	'/page/2/',
	'/feed/',
	'/comments/feed/',
	'/?p=1',
	'/?page_id=2',
	'/wp-json/',
	'/wp-json/wp/v2/posts',
	'/2020/',
	'/2020/01/',
	'/blog/uncategorized/hello-world/',
	'/index.php/hello-world/',
);

foreach ( $structures as $slug => $structure ) {
	if ( ! $wp_rewrite ) {
		WP_CLI::warning( '$wp_rewrite unavailable' );
		break;
	}

	$wp_rewrite->set_permalink_structure( $structure );
	$wp_rewrite->flush_rules( false );
	$rules = $wp_rewrite->rewrite_rules();
	gopress_golden_write(
		"rewrite/rules-{$slug}.json",
		array(
			'structure' => $structure,
			'rules'     => $rules ? $rules : array(),
		)
	);

	// Parse a smaller set of URLs against this structure.
	$parsed = array();
	foreach ( $parse_urls as $url ) {
		$parsed[] = gopress_rewrite_parse_url( $url );
	}
	gopress_golden_write(
		"rewrite/parse-{$slug}.json",
		array(
			'structure' => $structure,
			'parsed'    => $parsed,
		)
	);
}

/**
 * Best-effort parse of a path using WP rewrite matching.
 *
 * @param string $url URL or path.
 * @return array
 */
function gopress_rewrite_parse_url( $url ) {
	global $wp_rewrite;

	$path = $url;
	if ( preg_match( '#^https?://[^/]+(.*)$#', $url, $m ) ) {
		$path = $m[1];
	}
	$path = '/' . ltrim( $path, '/' );
	// Strip query for matching; keep separately.
	$query = '';
	if ( false !== strpos( $path, '?' ) ) {
		$parts = explode( '?', $path, 2 );
		$path  = $parts[0];
		$query = isset( $parts[1] ) ? $parts[1] : '';
	}

	$request = trim( $path, '/' );
	$matched_rule = null;
	$matched_query = null;
	$query_vars = array();

	$rules = $wp_rewrite ? $wp_rewrite->rewrite_rules() : array();
	if ( $rules ) {
		foreach ( $rules as $match => $query_str ) {
			if ( preg_match( "#^{$match}#", $request, $matches ) ) {
				$matched_rule  = $match;
				$matched_query = $query_str;
				// Substitute $matches into query.
				$query_str_filled = preg_replace_callback(
					'/\$matches\[(\d+)\]/',
					function ( $mm ) use ( $matches ) {
						$i = (int) $mm[1];
						return isset( $matches[ $i ] ) ? $matches[ $i ] : '';
					},
					$query_str
				);
				parse_str( (string) $query_str_filled, $query_vars );
				break;
			}
		}
	}

	if ( $query ) {
		parse_str( $query, $extra );
		$query_vars = array_merge( $query_vars, $extra );
	}

	$is_flags = array();
	foreach ( array( 'p', 'page_id', 'name', 'pagename', 'category_name', 'tag', 'author_name', 'feed', 'year', 'monthnum', 'day', 'attachment' ) as $qv ) {
		if ( isset( $query_vars[ $qv ] ) && '' !== $query_vars[ $qv ] && null !== $query_vars[ $qv ] ) {
			$is_flags[ 'has_' . $qv ] = true;
		}
	}

	return array(
		'url'           => $url,
		'request'       => $request,
		'matched_rule'  => $matched_rule,
		'matched_query' => $matched_query,
		'query_vars'    => $query_vars,
		'flags'         => $is_flags,
	);
}

WP_CLI::success( 'rewrite dump complete' );
