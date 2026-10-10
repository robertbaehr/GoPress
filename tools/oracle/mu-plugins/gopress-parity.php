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

/**
 * Emit WXR-style domain=taxonomy on RSS2 item categories (GoPress P3.71).
 *
 * Replaces get_the_category_rss output for rss2 so the oracle matches GoPress
 * feedCategory.Domain (category / post_tag). Atom and RDF are unchanged.
 *
 * @param string $the_list Category markup from get_the_category_rss.
 * @param string $type     Feed type (rss2, atom, rdf, …).
 * @return string
 */
function gopress_oracle_category_rss_domain( $the_list, $type ) {
	if ( 'atom' === $type || 'rdf' === $type ) {
		return $the_list;
	}

	$categories = get_the_category();
	$tags       = get_the_tags();
	$out        = '';
	$seen       = array();
	$charset    = get_option( 'blog_charset' );

	if ( ! empty( $categories ) ) {
		foreach ( (array) $categories as $category ) {
			$name = sanitize_term_field( 'name', $category->name, $category->term_id, 'category', 'rss' );
			if ( isset( $seen[ $name ] ) ) {
				continue;
			}
			$seen[ $name ] = true;
			$out          .= "\t\t<category domain=\"category\"><![CDATA[" . html_entity_decode( $name, ENT_COMPAT, $charset ) . "]]></category>\n";
		}
	}

	if ( ! empty( $tags ) ) {
		foreach ( (array) $tags as $tag ) {
			$name = sanitize_term_field( 'name', $tag->name, $tag->term_id, 'post_tag', 'rss' );
			if ( isset( $seen[ $name ] ) ) {
				continue;
			}
			$seen[ $name ] = true;
			$out          .= "\t\t<category domain=\"post_tag\"><![CDATA[" . html_entity_decode( $name, ENT_COMPAT, $charset ) . "]]></category>\n";
		}
	}

	return $out;
}

add_filter( 'the_category_rss', 'gopress_oracle_category_rss_domain', 10, 2 );
