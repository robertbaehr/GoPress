<?php
/**
 * Router for PHP's built-in server serving the GoPress oracle site (P0.3).
 *
 * Usage:
 *   php -S 127.0.0.1:8081 -t /opt/gopress-ref /opt/gopress-ref/router.php
 *
 * @package GoPress
 */

$path = parse_url( $_SERVER['REQUEST_URI'], PHP_URL_PATH );
$file = __DIR__ . $path;
if ( is_file( $file ) ) {
	return false;
}
if ( is_dir( $file ) && is_file( rtrim( $file, '/' ) . '/index.php' ) ) {
	$_SERVER['SCRIPT_NAME'] = rtrim( $path, '/' ) . '/index.php';
	require rtrim( $file, '/' ) . '/index.php';
	return;
}
$_SERVER['SCRIPT_NAME'] = '/index.php';
require __DIR__ . '/index.php';
