<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: WP date/time helpers across timezones.
 *
 * @package GoPress
 */

$timestamps = array( 0, 946684800, 1609459200, 1700000000, 2000000000 );
$timezones  = array( 'UTC', 'America/New_York', 'Europe/Berlin', 'Asia/Tokyo', 'Pacific/Auckland' );
$formats    = array( 'Y-m-d H:i:s', 'c', 'U', 'F j, Y' );
$mysql_dates = array(
	'2020-01-01 00:00:00',
	'2020-06-15 12:30:45',
	'2021-12-31 23:59:59',
);

$out = array(
	'wp_date'           => array(),
	'date_i18n'         => array(),
	'mysql2date'        => array(),
	'get_gmt_from_date' => array(),
	'get_date_from_gmt' => array(),
);

$original_tz = get_option( 'timezone_string' );
$original_offset = get_option( 'gmt_offset' );

foreach ( $timezones as $tz ) {
	update_option( 'timezone_string', $tz );
	update_option( 'gmt_offset', 0 );
	// Clear WP timezone cache if present.
	if ( function_exists( 'wp_timezone' ) ) {
		// Force re-read.
		unset( $GLOBALS['wp_locale'] );
	}

	foreach ( $timestamps as $ts ) {
		foreach ( $formats as $fmt ) {
			if ( function_exists( 'wp_date' ) ) {
				$out['wp_date'][] = array(
					'tz'     => $tz,
					'ts'     => $ts,
					'format' => $fmt,
					'result' => wp_date( $fmt, $ts, new DateTimeZone( $tz ) ),
				);
			}
			if ( function_exists( 'date_i18n' ) ) {
				$out['date_i18n'][] = array(
					'tz'     => $tz,
					'ts'     => $ts,
					'format' => $fmt,
					'result' => date_i18n( $fmt, $ts, false ),
				);
			}
		}
	}

	foreach ( $mysql_dates as $md ) {
		if ( function_exists( 'mysql2date' ) ) {
			$out['mysql2date'][] = array(
				'tz'     => $tz,
				'mysql'  => $md,
				'format' => 'Y-m-d H:i:s',
				'result' => mysql2date( 'Y-m-d H:i:s', $md, false ),
				'gmt'    => mysql2date( 'Y-m-d H:i:s', $md, true ),
			);
		}
		if ( function_exists( 'get_gmt_from_date' ) ) {
			$out['get_gmt_from_date'][] = array(
				'tz'     => $tz,
				'local'  => $md,
				'result' => get_gmt_from_date( $md, 'Y-m-d H:i:s' ),
			);
		}
		if ( function_exists( 'get_date_from_gmt' ) ) {
			$out['get_date_from_gmt'][] = array(
				'tz'     => $tz,
				'gmt'    => $md,
				'result' => get_date_from_gmt( $md, 'Y-m-d H:i:s' ),
			);
		}
	}
}

// Restore site timezone.
update_option( 'timezone_string', $original_tz );
update_option( 'gmt_offset', $original_offset );

foreach ( array( 'wp_date', 'date_i18n', 'mysql2date', 'get_gmt_from_date', 'get_date_from_gmt' ) as $fn ) {
	if ( ! function_exists( $fn ) ) {
		WP_CLI::warning( "$fn missing" );
	}
}

gopress_golden_write( 'dates/vectors.json', $out );
WP_CLI::success( 'dates dump complete' );
