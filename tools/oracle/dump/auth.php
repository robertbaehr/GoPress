<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: auth salts, hashes, cookies, nonces, fast hash.
 *
 * @package GoPress
 */

$out = array(
	'salts'         => array(),
	'hashes'        => array(),
	'hash_password' => null,
	'phpass_check'  => null,
	'auth_cookie'   => null,
	'nonces'        => array(),
	'fast_hash'     => null,
);

$schemes = array( 'auth', 'secure_auth', 'logged_in', 'nonce' );

foreach ( $schemes as $scheme ) {
	if ( function_exists( 'wp_salt' ) ) {
		$salt = wp_salt( $scheme );
		// Site-specific secrets: store length + sha256 for fixture stability checks.
		$out['salts'][ $scheme ] = array(
			'length' => strlen( $salt ),
			'hash'   => hash( 'sha256', $salt ),
		);
	} else {
		WP_CLI::warning( 'wp_salt missing' );
		break;
	}
	if ( function_exists( 'wp_hash' ) ) {
		$out['hashes'][ $scheme ] = wp_hash( 'message', $scheme );
	} else {
		WP_CLI::warning( 'wp_hash missing' );
	}
}

if ( function_exists( 'wp_hash_password' ) ) {
	$hp = wp_hash_password( 'password' );
	$out['hash_password'] = array(
		'input'  => 'password',
		'hash'   => 'HASH',
		'length' => strlen( $hp ),
		'prefix' => substr( $hp, 0, 3 ),
	);
} else {
	WP_CLI::warning( 'wp_hash_password missing' );
}

// Known phpass portable hash verification boolean.
if ( ! class_exists( 'PasswordHash', false ) ) {
	$phpass = ABSPATH . WPINC . '/class-phpass.php';
	if ( file_exists( $phpass ) ) {
		require_once $phpass;
	}
}

if ( class_exists( 'PasswordHash' ) ) {
	$hasher = new PasswordHash( 8, true );
	// Well-known WordPress portable hash for the string "password"
	// (HashPassword output varies; use CheckPassword against a freshly made portable hash
	// and a fixed invalid hash for a stable boolean pair).
	$portable = $hasher->HashPassword( 'password' );
	$out['phpass_check'] = array(
		'password_matches_portable' => (bool) $hasher->CheckPassword( 'password', $portable ),
		'wrong_rejects_portable'    => (bool) $hasher->CheckPassword( 'wrong-password', $portable ),
		// Fixed classic invalid / truncated portable stub → false.
		'known_bad_hash'            => (bool) $hasher->CheckPassword( 'password', '$P$Bxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' ),
		'stored_hash'               => 'HASH',
	);
	if ( function_exists( 'wp_check_password' ) ) {
		$out['phpass_check']['wp_check_password'] = (bool) wp_check_password( 'password', $portable );
	}
} else {
	WP_CLI::warning( 'PasswordHash unavailable' );
}

if ( function_exists( 'wp_generate_auth_cookie' ) ) {
	$user_id    = 1;
	$expiration = 2000000000;
	$token      = 'fixedtokenfixedtokenfixedtokenfixedtok';
	$cookies    = array();
	foreach ( array( 'auth', 'secure_auth', 'logged_in' ) as $scheme ) {
		$cookie = wp_generate_auth_cookie( $user_id, $expiration, $scheme, $token );
		// Normalize long hex hmac segments.
		$cookies[ $scheme ] = preg_replace( '/[0-9a-f]{32,}/i', 'HASH', $cookie );
	}
	$out['auth_cookie'] = array(
		'user_id'    => $user_id,
		'expiration' => $expiration,
		'token'      => $token,
		'cookies'    => $cookies,
	);
} else {
	WP_CLI::warning( 'wp_generate_auth_cookie missing' );
}

// Nonces via substr(wp_hash("$tick|$action|$uid|$token", 'nonce'), -12, 10)
if ( function_exists( 'wp_hash' ) ) {
	foreach ( array( 1000, 1001 ) as $tick ) {
		foreach ( array( 'wp_rest', '-1' ) as $action ) {
			foreach ( array( 1, 0 ) as $uid ) {
				$token = 'tok';
				$raw   = wp_hash( "$tick|$action|$uid|$token", 'nonce' );
				$nonce = substr( $raw, -12, 10 );
				$out['nonces'][] = array(
					'tick'   => $tick,
					'action' => $action,
					'uid'    => $uid,
					'token'  => $token,
					'nonce'  => $nonce,
				);
			}
		}
	}
}

if ( function_exists( 'wp_fast_hash' ) ) {
	$out['fast_hash'] = wp_fast_hash( 'message' );
} else {
	WP_CLI::warning( 'wp_fast_hash missing; skipping' );
}

gopress_golden_write( 'auth/vectors.json', $out );
WP_CLI::success( 'auth dump complete' );
