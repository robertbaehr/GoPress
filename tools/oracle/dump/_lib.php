<?php
/**
 * Shared helpers for GoPress golden dump scripts (P0.5).
 *
 * @package GoPress
 */

if ( ! defined( 'ABSPATH' ) ) {
	fwrite( STDERR, "Dump scripts must be run via wp eval-file.\n" );
	exit( 1 );
}

/**
 * Absolute path to tests/golden/.
 *
 * @return string
 */
function gopress_golden_dir() {
	$repo = getenv( 'GOPRESS_REPO' ) ?: '/workspace';
	return rtrim( $repo, '/' ) . '/tests/golden';
}

/**
 * Write JSON with stable encoding (pretty, unescaped unicode/slashes, sorted keys).
 *
 * @param string $relative Path under tests/golden/.
 * @param mixed  $data     Data to encode.
 * @param array  $args {
 *     Optional.
 *     @type bool $scrub_volatiles Replace nonce/time-like substrings (default false).
 * }
 */
function gopress_golden_write( $relative, $data, $args = array() ) {
	$args = array_merge(
		array(
			'scrub_volatiles' => false,
		),
		$args
	);
	$path = gopress_golden_dir() . '/' . ltrim( $relative, '/' );
	$dir  = dirname( $path );
	if ( ! is_dir( $dir ) ) {
		mkdir( $dir, 0755, true );
	}
	$data = gopress_golden_normalize( $data, (bool) $args['scrub_volatiles'] );
	$json = wp_json_encode( $data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE );
	if ( false === $json ) {
		WP_CLI::error( 'json_encode failed for ' . $relative );
	}
	// Trailing newline for POSIX friendliness.
	file_put_contents( $path, $json . "\n" );
	WP_CLI::log( 'wrote ' . $relative );
}

/**
 * Recursively sort associative arrays by key; optionally scrub volatile strings.
 *
 * @param mixed $value           Value.
 * @param bool  $scrub_volatiles Whether to replace nonce/time-like substrings.
 * @return mixed
 */
function gopress_golden_normalize( $value, $scrub_volatiles = false ) {
	if ( is_array( $value ) ) {
		$out = array();
		foreach ( $value as $k => $v ) {
			$out[ $k ] = gopress_golden_normalize( $v, $scrub_volatiles );
		}
		// Sort only associative arrays (preserve list order).
		if ( gopress_golden_is_assoc( $out ) ) {
			ksort( $out );
		}
		return $out;
	}
	if ( is_string( $value ) ) {
		return $scrub_volatiles ? gopress_golden_normalize_string( $value ) : $value;
	}
	if ( is_object( $value ) ) {
		if ( $value instanceof WP_Error ) {
			return array(
				'__type'     => 'WP_Error',
				'errors'     => $value->errors,
				'error_data' => $value->error_data,
			);
		}
		// Reduce callables / resources.
		return array( '__type' => get_class( $value ) );
	}
	if ( is_resource( $value ) ) {
		return '__resource__';
	}
	return $value;
}

/**
 * @param array $arr Array.
 * @return bool
 */
function gopress_golden_is_assoc( array $arr ) {
	if ( array() === $arr ) {
		return false;
	}
	return array_keys( $arr ) !== range( 0, count( $arr ) - 1 );
}

/**
 * Replace volatile substrings.
 *
 * @param string $s String.
 * @return string
 */
function gopress_golden_normalize_string( $s ) {
	// Nonces (10 hex-ish chars commonly).
	$s = preg_replace( '/\b[0-9a-f]{10}\b/', 'NONCE', $s );
	// ISO / mysql datetimes.
	$s = preg_replace( '/\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}/', 'TIME', $s );
	// Unix timestamps in strings (10 digits).
	$s = preg_replace( '/\b1[6-9]\d{8}\b/', 'TIME', $s );
	return $s;
}

/**
 * Shared string corpus for formatting / kses / phpx dumps.
 *
 * @return string[]
 */
function gopress_golden_corpus() {
	return array(
		'',
		'0',
		'1',
		'-1',
		'12abc',
		'  spaced  ',
		"line1\nline2",
		"a\tb",
		'<script>alert(1)</script>',
		'<p onclick="x">hi</p>',
		'<a href="javascript:alert(1)">x</a>',
		'<a href="https://example.com/path?q=1&r=2">link</a>',
		'https://example.com/foo bar',
		'//example.com/scheme-relative',
		'mailto:test@example.com',
		'foo@bar.com',
		'not-an-email',
		"O'Reilly & Sons",
		'"quoted"',
		'café résumé naïve',
		"emoji 😀",
		'&amp; &lt; &gt;',
		'&#38; &#x26;',
		'hello...world',
		"it's a test",
		'--double-dash--',
		'word "quote" word',
		"http://example.com\nhttp://evil.com",
		'<img src=x onerror=alert(1)>',
		'<div style="color:red; expression(alert(1))">x</div>',
		'../../etc/passwd',
		'file:///etc/passwd',
		'data:text/html,<b>x</b>',
		str_repeat( 'a', 200 ),
		'Foo_Bar-baz.123',
		'UPPER lower MiXeD',
		"null\0byte",
		'<![CDATA[test]]>',
		'<?xml version="1.0"?>',
		'select * from users; drop table users;--',
		'https://xn--fsq.com/',
		'1.5e+2',
		'000123',
		'-0',
		'true',
		'false',
		'null',
		"\r\n",
		'&nbsp;',
		'<br>',
		'<br/>',
		'<BR />',
		'http://user:pass@example.com:8080/path?x=1#frag',
		'[shortcode attr="1"]content[/shortcode]',
		'<!--more-->',
		'<!--nextpage-->',
		'Hello <!--comment--> world',
	);
}

/**
 * Flatten corpus to strings only.
 *
 * @return string[]
 */
function gopress_golden_corpus_strings() {
	$out = array();
	foreach ( gopress_golden_corpus() as $item ) {
		if ( is_array( $item ) ) {
			foreach ( $item as $s ) {
				$out[] = (string) $s;
			}
		} else {
			$out[] = (string) $item;
		}
	}
	// Deduplicate while preserving order.
	$seen = array();
	$uniq = array();
	foreach ( $out as $s ) {
		if ( isset( $seen[ $s ] ) ) {
			continue;
		}
		$seen[ $s ] = true;
		$uniq[]     = $s;
	}
	return $uniq;
}

/**
 * Reduce a value for JSON: callables → bool presence, WP_Error → structure.
 *
 * @param mixed $v Value.
 * @return mixed
 */
function gopress_golden_public_props( $v ) {
	if ( is_object( $v ) ) {
		$out = array();
		foreach ( get_object_vars( $v ) as $k => $val ) {
			if ( is_callable( $val ) && ! is_string( $val ) && ! is_array( $val ) ) {
				$out[ $k ] = true;
			} elseif ( is_object( $val ) && $val instanceof Closure ) {
				$out[ $k ] = true;
			} else {
				$out[ $k ] = gopress_golden_public_props( $val );
			}
		}
		return $out;
	}
	if ( is_array( $v ) ) {
		$out = array();
		foreach ( $v as $k => $val ) {
			if ( is_callable( $val ) && ! is_string( $val ) ) {
				$out[ $k ] = true;
			} else {
				$out[ $k ] = gopress_golden_public_props( $val );
			}
		}
		return $out;
	}
	return $v;
}
