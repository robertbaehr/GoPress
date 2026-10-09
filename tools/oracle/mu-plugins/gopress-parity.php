<?php
/**
 * Plugin Name: GoPress Parity Caps
 * Description: Applies the same map_meta_cap denials as GoPress (Tier 1 out-of-scope features). Oracle only.
 * Version: 1.0.0
 *
 * @package GoPress
 */

/**
 * Deny capabilities that GoPress treats as unavailable (section 1 of the porting plan).
 *
 * Mirrors GoPress's gopressDenyUnsupportedCaps filter on map_meta_cap.
 *
 * @param string[] $caps    Primitive capabilities required of the user.
 * @param string   $cap     Capability being checked.
 * @param int      $user_id User ID.
 * @param array    $args    Extra arguments.
 * @return string[]
 */
function gopress_oracle_deny_unsupported_caps( $caps, $cap, $user_id, $args ) {
	unset( $user_id, $args );

	static $denied = array(
		// Plugins.
		'install_plugins' => true,
		'upload_plugins'  => true,
		'edit_plugins'    => true,
		'delete_plugins'  => true,
		'update_plugins'  => true,
		// Themes.
		'install_themes'  => true,
		'upload_themes'   => true,
		'edit_themes'     => true,
		'update_themes'   => true,
		// Other.
		'update_core'      => true,
		'update_languages' => true,
		'customize'        => true,
	);

	if ( isset( $denied[ $cap ] ) ) {
		return array( 'do_not_allow' );
	}

	return $caps;
}

add_filter( 'map_meta_cap', 'gopress_oracle_deny_unsupported_caps', 10, 4 );
