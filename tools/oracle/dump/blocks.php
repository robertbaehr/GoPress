<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: block types, server block settings, styles, patterns, bindings.
 *
 * @package GoPress
 */

// Ensure blocks are registered.
if ( function_exists( 'do_action' ) ) {
	// Core registers on init; wp-cli already ran init.
}

if ( class_exists( 'WP_Block_Type_Registry' ) ) {
	$registry = WP_Block_Type_Registry::get_instance();
	$types    = $registry->get_all_registered();
	$out      = array();
	foreach ( $types as $name => $block ) {
		$out[ $name ] = gopress_golden_public_props( $block );
	}
	ksort( $out );
	gopress_golden_write( 'blocks/block-types.json', $out );
} else {
	WP_CLI::warning( 'WP_Block_Type_Registry missing; skipping block-types' );
}

if ( function_exists( 'get_block_editor_server_block_settings' ) ) {
	$settings = get_block_editor_server_block_settings();
	gopress_golden_write( 'blocks/server-block-settings.json', gopress_golden_public_props( $settings ) );
} else {
	WP_CLI::warning( 'get_block_editor_server_block_settings missing; skipping' );
}

if ( function_exists( 'WP_Block_Styles_Registry' ) || class_exists( 'WP_Block_Styles_Registry' ) ) {
	$styles_reg = WP_Block_Styles_Registry::get_instance();
	$all_styles = $styles_reg->get_all_registered();
	gopress_golden_write( 'blocks/block-styles.json', gopress_golden_public_props( $all_styles ) );
} else {
	WP_CLI::warning( 'WP_Block_Styles_Registry missing; skipping block-styles' );
}

if ( class_exists( 'WP_Block_Pattern_Categories_Registry' ) ) {
	$cats = WP_Block_Pattern_Categories_Registry::get_instance()->get_all_registered();
	gopress_golden_write( 'blocks/pattern-categories.json', gopress_golden_public_props( $cats ) );
} elseif ( function_exists( 'get_block_pattern_categories' ) ) {
	gopress_golden_write( 'blocks/pattern-categories.json', get_block_pattern_categories() );
} else {
	WP_CLI::warning( 'pattern categories API missing; skipping' );
}

if ( class_exists( 'WP_Block_Bindings_Registry' ) ) {
	$bindings = WP_Block_Bindings_Registry::get_instance()->get_all_registered();
	$bind_out = array();
	foreach ( $bindings as $name => $source ) {
		$bind_out[ $name ] = gopress_golden_public_props( $source );
	}
	gopress_golden_write( 'blocks/bindings-sources.json', $bind_out );
} else {
	WP_CLI::warning( 'WP_Block_Bindings_Registry missing; skipping bindings-sources' );
}

WP_CLI::success( 'blocks dump complete' );
