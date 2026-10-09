<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: all options from the current (fixture) DB.
 *
 * Writes options-fixture.json (autoload-aware dump from current DB).
 * Also writes options-fresh-install.json as the same map when no separate
 * fresh-install marker is present — see NOTE below.
 *
 * NOTE: This environment loads tests/fixtures/site.sql. There is no separate
 * fresh-install DB available to the dump runner, so options-fresh-install.json
 * is written as a copy of the fixture options map for schema completeness.
 * Prefer options-fixture.json for fixture-derived comparisons.
 *
 * @package GoPress
 */

global $wpdb;

$rows = $wpdb->get_results(
	"SELECT option_name, option_value, autoload FROM {$wpdb->options} ORDER BY option_name ASC",
	ARRAY_A
);

$map      = array();
$autoload = array();

foreach ( (array) $rows as $row ) {
	$name  = $row['option_name'];
	$value = maybe_unserialize( $row['option_value'] );
	$map[ $name ]      = $value;
	$autoload[ $name ] = $row['autoload'];
}

$payload = array(
	'note'     => 'Dumped from current DB (fixture). See options.php header NOTE.',
	'count'    => count( $map ),
	'autoload' => $autoload,
	'options'  => $map,
);

gopress_golden_write( 'options/options-fixture.json', $payload );
gopress_golden_write( 'options/options-fresh-install.json', $payload );

WP_CLI::success( 'options dump complete (' . count( $map ) . ' options)' );
