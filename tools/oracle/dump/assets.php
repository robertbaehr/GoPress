<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: registered scripts/styles (front + admin) and script modules.
 *
 * @package GoPress
 */

/**
 * Dump registered WP_Dependencies handles.
 *
 * @param WP_Dependencies $deps Dependencies instance.
 * @return array
 */
function gopress_dump_dependencies( $deps ) {
	$out = array();
	if ( ! $deps || ! is_object( $deps ) ) {
		return $out;
	}
	$registered = isset( $deps->registered ) ? $deps->registered : array();
	foreach ( $registered as $handle => $obj ) {
		$extra = isset( $obj->extra ) ? $obj->extra : array();
		// Drop volatile translation paths that include ABSPATH.
		$row = array(
			'src'               => isset( $obj->src ) ? $obj->src : null,
			'deps'              => isset( $obj->deps ) ? array_values( (array) $obj->deps ) : array(),
			'ver'               => isset( $obj->ver ) ? $obj->ver : null,
			'args'              => isset( $obj->args ) ? $obj->args : null,
			'extra'             => $extra,
			'textdomain'        => isset( $obj->textdomain ) ? $obj->textdomain : null,
			'translations_path' => isset( $obj->translations_path ) ? $obj->translations_path : null,
		);
		$out[ $handle ] = $row;
	}
	ksort( $out );
	return $out;
}

// --- Front ---
global $wp_scripts, $wp_styles;

$wp_scripts = new WP_Scripts();
$wp_styles  = new WP_Styles();

if ( function_exists( 'wp_default_scripts' ) ) {
	wp_default_scripts( $wp_scripts );
}
if ( function_exists( 'wp_default_styles' ) ) {
	wp_default_styles( $wp_styles );
}

// Simulate front enqueue context.
if ( ! defined( 'WP_ADMIN' ) ) {
	// Already may be defined false.
}
try {
	do_action( 'wp_enqueue_scripts' );
} catch ( Exception $e ) {
	WP_CLI::warning( 'wp_enqueue_scripts threw: ' . $e->getMessage() );
}

gopress_golden_write( 'scripts-front.json', gopress_dump_dependencies( $wp_scripts ) );
gopress_golden_write( 'styles-front.json', gopress_dump_dependencies( $wp_styles ) );

// --- Admin ---
$wp_scripts = new WP_Scripts();
$wp_styles  = new WP_Styles();
if ( function_exists( 'wp_default_scripts' ) ) {
	wp_default_scripts( $wp_scripts );
}
if ( function_exists( 'wp_default_styles' ) ) {
	wp_default_styles( $wp_styles );
}

// Set admin context flags as best-effort.
if ( ! defined( 'WP_ADMIN' ) ) {
	define( 'WP_ADMIN', true );
}
try {
	if ( function_exists( 'wp_set_current_user' ) ) {
		$admins = get_users( array( 'role' => 'administrator', 'number' => 1 ) );
		if ( ! empty( $admins ) ) {
			wp_set_current_user( $admins[0]->ID );
		}
	}
	// Avoid null $current_screen notices in admin enqueue callbacks.
	if ( ! function_exists( 'set_current_screen' ) && file_exists( ABSPATH . 'wp-admin/includes/screen.php' ) ) {
		require_once ABSPATH . 'wp-admin/includes/screen.php';
	}
	if ( function_exists( 'set_current_screen' ) ) {
		set_current_screen( 'dashboard' );
	}
	do_action( 'admin_enqueue_scripts', 'index.php' );
} catch ( Exception $e ) {
	WP_CLI::warning( 'admin_enqueue_scripts threw: ' . $e->getMessage() );
}

gopress_golden_write( 'scripts-admin.json', gopress_dump_dependencies( $wp_scripts ) );
gopress_golden_write( 'styles-admin.json', gopress_dump_dependencies( $wp_styles ) );

// Script modules (WP 6.5+).
if ( class_exists( 'WP_Script_Modules' ) || function_exists( 'wp_script_modules' ) ) {
	$modules = array();
	if ( function_exists( 'wp_script_modules' ) ) {
		$sm = wp_script_modules();
		// Reflect registered modules if accessible.
		if ( is_object( $sm ) ) {
			$ref = new ReflectionClass( $sm );
			if ( $ref->hasProperty( 'registered' ) ) {
				$prop = $ref->getProperty( 'registered' );
				$prop->setAccessible( true );
				$modules = $prop->getValue( $sm );
			} elseif ( $ref->hasProperty( 'modules' ) ) {
				$prop = $ref->getProperty( 'modules' );
				$prop->setAccessible( true );
				$modules = $prop->getValue( $sm );
			}
		}
	}
	gopress_golden_write( 'script-modules.json', gopress_golden_public_props( $modules ) );
} else {
	WP_CLI::warning( 'WP_Script_Modules missing; skipping script-modules.json' );
}

WP_CLI::success( 'assets dump complete' );
