<?php
require __DIR__ . '/_lib.php';

/**
 * Golden dump: PHP / WordPress-adjacent PHP behavior vectors.
 *
 * @package GoPress
 */

$out = array(
	'serialize'           => array(),
	'json_encode'         => array(),
	'sprintf'             => array(),
	'date'                => array(),
	'strtotime'           => array(),
	'parse_url'           => array(),
	'http_build_query'    => array(),
	'urlencode'           => array(),
	'htmlspecialchars'    => array(),
	'html_entity_decode'  => array(),
	'strip_tags'          => array(),
	'number_format'       => array(),
	'round'               => array(),
	'strnatcasecmp'       => array(),
	'ucwords'             => array(),
	'wordwrap'            => array(),
	'nl2br'               => array(),
	'preg_match'          => array(),
	'preg_replace'        => array(),
);

// --- serialize / unserialize roundtrips ---
$serialize_samples = array(
	null,
	true,
	false,
	0,
	1,
	-1,
	3.14,
	'',
	'hello',
	array(),
	array( 1, 2, 3 ),
	array( 'a' => 1, 'b' => 'two' ),
	array( 'nested' => array( 'x' => array( 1 ) ) ),
	"O'Reilly",
	"null\0byte",
	array( 'unicode' => 'café' ),
);

foreach ( $serialize_samples as $i => $sample ) {
	$ser = serialize( $sample );
	$uns = unserialize( $ser );
	$out['serialize'][] = array(
		'i'           => $i,
		'serialized'  => $ser,
		'roundtrip_ok' => $uns === $sample || ( is_float( $sample ) && is_float( $uns ) && abs( $sample - $uns ) < 1e-9 ),
		'input_type'  => gettype( $sample ),
	);
}

// --- json_encode flag combinations used in WP ---
$json_inputs = array(
	array( 'tag' => '<script>', 'slash' => 'a/b', 'uni' => 'café', 'emoji' => '😀' ),
	array( 1, 2, 3 ),
	(object) array( 'a' => 1 ),
	'',
	null,
	true,
	array(),
);

$flag_sets = array(
	array( 'name' => '0', 'flags' => 0 ),
	array( 'name' => 'HexTag', 'flags' => JSON_HEX_TAG ),
	array( 'name' => 'UnescapedSlashes', 'flags' => JSON_UNESCAPED_SLASHES ),
	array( 'name' => 'UnescapedUnicode', 'flags' => JSON_UNESCAPED_UNICODE ),
	array( 'name' => 'PrettyPrint', 'flags' => JSON_PRETTY_PRINT ),
	array( 'name' => 'ForceObject', 'flags' => JSON_FORCE_OBJECT ),
	array( 'name' => 'HexTag|UnescapedSlashes', 'flags' => JSON_HEX_TAG | JSON_UNESCAPED_SLASHES ),
	array( 'name' => 'UnescapedSlashes|UnescapedUnicode', 'flags' => JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE ),
	array( 'name' => 'PrettyPrint|UnescapedUnicode|UnescapedSlashes', 'flags' => JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
	array( 'name' => 'HexTag|UnescapedSlashes|UnescapedUnicode', 'flags' => JSON_HEX_TAG | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE ),
	array( 'name' => 'ForceObject|UnescapedUnicode', 'flags' => JSON_FORCE_OBJECT | JSON_UNESCAPED_UNICODE ),
);

foreach ( $json_inputs as $ji => $jin ) {
	foreach ( $flag_sets as $fs ) {
		$encoded = json_encode( $jin, $fs['flags'] );
		$out['json_encode'][] = array(
			'input_i' => $ji,
			'flags'   => $fs['name'],
			'result'  => $encoded,
		);
	}
}

// --- sprintf ---
$sprintf_cases = array(
	array( '%s', array( 'hello' ) ),
	array( '%d', array( 42 ) ),
	array( '%05d', array( 42 ) ),
	array( '%.2f', array( 3.14159 ) ),
	array( '%s-%d', array( 'post', 7 ) ),
	array( '%1$s %2$s', array( 'a', 'b' ) ),
	array( '%2$s %1$s', array( 'a', 'b' ) ),
	array( '%%s', array() ),
	array( '%u', array( -1 ) ),
	array( '%x', array( 255 ) ),
	array( '%o', array( 8 ) ),
	array( '%c', array( 65 ) ),
	array( '%.0f', array( 1.6 ) ),
	array( '%+d', array( 5 ) ),
	array( '% 8s', array( 'x' ) ),
);

foreach ( $sprintf_cases as $sc ) {
	$out['sprintf'][] = array(
		'format' => $sc[0],
		'args'   => $sc[1],
		'result' => call_user_func_array( 'sprintf', array_merge( array( $sc[0] ), $sc[1] ) ),
	);
}

// --- date() for YmdHisu across timestamps / timezones ---
$timestamps = array( 0, 946684800, 1609459200, 1700000000, 2000000000 );
$timezones  = array( 'UTC', 'America/New_York', 'Europe/Berlin', 'Asia/Tokyo', 'Pacific/Auckland' );
$date_fmt   = 'Y-m-d H:i:s u YmdHis';

foreach ( $timezones as $tz ) {
	$prev = date_default_timezone_get();
	date_default_timezone_set( $tz );
	foreach ( $timestamps as $ts ) {
		$out['date'][] = array(
			'tz'     => $tz,
			'ts'     => $ts,
			'Y'      => date( 'Y', $ts ),
			'm'      => date( 'm', $ts ),
			'd'      => date( 'd', $ts ),
			'H'      => date( 'H', $ts ),
			'i'      => date( 'i', $ts ),
			's'      => date( 's', $ts ),
			'u'      => date( 'u', $ts ),
			'YmdHis' => date( 'YmdHis', $ts ),
			'full'   => date( $date_fmt, $ts ),
		);
	}
	date_default_timezone_set( $prev );
}

// --- strtotime (~40 inputs) ---
$strtotime_inputs = array(
	'now',
	'today',
	'tomorrow',
	'yesterday',
	'+1 day',
	'-1 day',
	'+1 week',
	'+1 month',
	'+1 year',
	'next Monday',
	'last Friday',
	'2020-01-01',
	'2020-01-01 12:00:00',
	'2020/01/01',
	'01/02/2020',
	'January 1, 2020',
	'1 Jan 2020',
	'2020-W01-1',
	'@1609459200',
	'midnight',
	'noon',
	'first day of this month',
	'last day of next month',
	'+2 hours',
	'-30 minutes',
	'2020-02-29',
	'2019-02-29',
	'Thu, 01 Jan 1970 00:00:00 GMT',
	'2020-01-01T00:00:00Z',
	'2020-01-01T00:00:00+00:00',
	'2020-01-01T12:00:00-05:00',
	'next year',
	'previous month',
	'third wednesday of january 2020',
	'2020-12-31 23:59:59',
	'invalid date xyz',
	'',
	'0',
	'+0 seconds',
	'1970-01-01',
);

$prev_tz = date_default_timezone_get();
date_default_timezone_set( 'UTC' );
foreach ( $strtotime_inputs as $si ) {
	// Anchor "relative" parses to a fixed base for determinism where possible.
	$base   = 1609459200; // 2021-01-01 00:00:00 UTC
	$result = strtotime( $si, $base );
	$out['strtotime'][] = array(
		'input'  => $si,
		'base'   => $base,
		'result' => false === $result ? false : $result,
	);
}
date_default_timezone_set( $prev_tz );

// --- parse_url ---
$parse_urls = array(
	'https://example.com/path?q=1&r=2#frag',
	'http://user:pass@example.com:8080/path',
	'//example.com/scheme-relative',
	'/relative/path',
	'mailto:test@example.com',
	'https://xn--fsq.com/',
	'https://example.com',
	'https://example.com/',
	'https://example.com:443/path',
	'data:text/html,<b>x</b>',
	'file:///etc/passwd',
	'',
	'not a url',
	'https://example.com/foo bar',
	'http://[::1]/80/path',
);

foreach ( $parse_urls as $pu ) {
	$out['parse_url'][] = array(
		'input'  => $pu,
		'result' => parse_url( $pu ),
	);
}

// --- http_build_query ---
$hbq_cases = array(
	array( 'a' => 1, 'b' => 2 ),
	array( 'a' => array( 1, 2 ) ),
	array( 'q' => 'hello world' ),
	array( 'tag' => '<b>' ),
	array( 'uni' => 'café' ),
	array(),
	array( 'a' => null, 'b' => false, 'c' => true ),
);

foreach ( $hbq_cases as $hb ) {
	$out['http_build_query'][] = array(
		'input'  => $hb,
		'result' => http_build_query( $hb ),
	);
}

// --- urlencode / rawurlencode ---
$url_strings = array(
	'hello world',
	'a/b',
	'café',
	'a+b',
	'a&b=c',
	'',
	'~!@#$%^&*()_+',
	'😀',
);

foreach ( $url_strings as $us ) {
	$out['urlencode'][] = array(
		'input'         => $us,
		'urlencode'     => urlencode( $us ),
		'rawurlencode'  => rawurlencode( $us ),
	);
}

// --- htmlspecialchars / html_entity_decode / strip_tags ---
$html_strings = array(
	'<script>alert(1)</script>',
	'"quoted"',
	"O'Reilly & Sons",
	'&amp; &lt; &gt;',
	'café',
	'<p onclick="x">hi</p>',
	'',
	"a\tb",
);

foreach ( $html_strings as $hs ) {
	$out['htmlspecialchars'][] = array(
		'input'  => $hs,
		'result' => htmlspecialchars( $hs, ENT_QUOTES, 'UTF-8' ),
	);
	$out['html_entity_decode'][] = array(
		'input'  => $hs,
		'result' => html_entity_decode( $hs, ENT_QUOTES, 'UTF-8' ),
	);
	$out['strip_tags'][] = array(
		'input'  => $hs,
		'result' => strip_tags( $hs ),
	);
}

// --- number_format / round ---
foreach ( array( 0, 1, 1234.567, -1234.567, 1000000, 0.1 + 0.2 ) as $n ) {
	$out['number_format'][] = array(
		'input'  => $n,
		'result' => number_format( $n, 2, '.', ',' ),
	);
	$out['round'][] = array(
		'input'   => $n,
		'round0'  => round( $n, 0 ),
		'round2'  => round( $n, 2 ),
		'round_1' => round( $n, -1 ),
	);
}

// --- strnatcasecmp pairs ---
$nat_pairs = array(
	array( 'img1', 'img2' ),
	array( 'img10', 'img2' ),
	array( 'img2', 'img10' ),
	array( 'A', 'a' ),
	array( 'a', 'A' ),
	array( '', 'a' ),
	array( 'file-1.txt', 'file-10.txt' ),
	array( '1', '01' ),
	array( 'v1.10', 'v1.9' ),
	array( 'café', 'cafe' ),
);

foreach ( $nat_pairs as $np ) {
	$out['strnatcasecmp'][] = array(
		'a'      => $np[0],
		'b'      => $np[1],
		'result' => strnatcasecmp( $np[0], $np[1] ),
	);
}

// --- ucwords / wordwrap / nl2br ---
$text_samples = array(
	'hello world',
	'HELLO WORLD',
	"hello\nworld",
	'a-b c_d',
	'',
	"line1\nline2\nline3",
);

foreach ( $text_samples as $ts ) {
	$out['ucwords'][] = array(
		'input'  => $ts,
		'result' => ucwords( $ts ),
	);
	$out['wordwrap'][] = array(
		'input'  => $ts,
		'result' => wordwrap( $ts, 5, "\n", true ),
	);
	$out['nl2br'][] = array(
		'input'  => $ts,
		'result' => nl2br( $ts ),
	);
}

// --- ~50 preg_match / preg_replace cases from WP-ish patterns ---
$preg_cases = array(
	array( '/^[a-z0-9-]+$/i', 'Hello-World', 'match' ),
	array( '/^[a-z0-9-]+$/i', 'Hello World', 'match' ),
	array( '/[^a-z0-9_\-]/i', 'Foo_Bar-baz!', 'replace', '_' ),
	array( '/\s+/', 'a  b   c', 'replace', ' ' ),
	array( '/[\r\n\t ]+/', "a\tb\nc", 'replace', ' ' ),
	array( '/<[^>]*>/', '<p>hi</p>', 'replace', '' ),
	array( '/<!--.*?-->/s', 'Hello <!--comment--> world', 'replace', '' ),
	array( '/\[(\[?)([A-Za-z0-9\-_]+)(?![\w-])([^\]\/]*(?:\/(?!\])[^\]\/]*)*?)(?:(\/)\]|\](?:([^\[]*+(?:\[(?!\/\2\])[^\[]*+)*+)\[\/\2\])?)(\]?)/', '[shortcode]x[/shortcode]', 'match' ),
	array( '/https?:\/\/[^\s<>"\']+/i', 'see https://example.com/path now', 'match' ),
	array( '/^https?:\/\//i', 'https://example.com', 'match' ),
	array( '/^https?:\/\//i', '//example.com', 'match' ),
	array( '/[#].*$/', 'color#fff', 'replace', '' ),
	array( '/\.+/', 'hello...world', 'replace', '.' ),
	array( '/[^\P{C}\n]+/u', "a\0b", 'replace', '' ),
	array( '/^[-a-z0-9+_]+(\.[-a-z0-9+_]+)*@([a-z0-9]([-a-z0-9]*[a-z0-9])?\.)+[a-z0-9]([-a-z0-9]*[a-z0-9])?$/i', 'foo@bar.com', 'match' ),
	array( '/^[-a-z0-9+_]+(\.[-a-z0-9+_]+)*@([a-z0-9]([-a-z0-9]*[a-z0-9])?\.)+[a-z0-9]([-a-z0-9]*[a-z0-9])?$/i', 'not-an-email', 'match' ),
	array( '/[^a-z0-9]/i', 'Foo Bar!', 'replace', '' ),
	array( '/-+/', 'a---b--c', 'replace', '-' ),
	array( '/^-|-$/', '-trim-', 'replace', '' ),
	array( '/\s/', 'a b c', 'match' ),
	array( '/^\d+$/', '12345', 'match' ),
	array( '/^\d+$/', '12abc', 'match' ),
	array( '/^(?:\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})$/', '2020-01-01 12:00:00', 'match' ),
	array( '/%([a-f0-9]{2})/i', '%20hello%2F', 'match' ),
	array( '/%([a-f0-9]{2})/i', '%20hello%2F', 'replace', 'X' ),
	array( '/&(?:[a-z\d]+|#\d+|#x[a-f\d]+);/i', '&amp; &lt; &#38;', 'match' ),
	array( '/&(?:[a-z\d]+|#\d+|#x[a-f\d]+);/i', '&amp; &lt;', 'replace', '' ),
	array( '/[\x00-\x08\x0B\x0C\x0E-\x1F]/', "a\x01b", 'replace', '' ),
	array( '/^(https?|ftp|mailto):/i', 'https://x', 'match' ),
	array( '/^(https?|ftp|mailto):/i', 'javascript:alert(1)', 'match' ),
	array( '/\.(jpe?g|png|gif|webp)$/i', 'photo.JPG', 'match' ),
	array( '/\.(jpe?g|png|gif|webp)$/i', 'photo.txt', 'match' ),
	array( '/[^a-z0-9\-\._]/i', '../../etc/passwd', 'replace', '' ),
	array( '/\/+/', 'a///b//c', 'replace', '/' ),
	array( '/\b(script|style)\b/i', '<script>', 'match' ),
	array( '/expression\s*\(/i', 'color: expression(alert(1))', 'match' ),
	array( '/javascript\s*:/i', 'javascript:alert(1)', 'match' ),
	array( '/\A[a-z0-9_]+\z/i', 'valid_key', 'match' ),
	array( '/\A[a-z0-9_]+\z/i', 'bad key', 'match' ),
	array( '/([A-Z])/', 'camelCase', 'replace', '_$1' ),
	array( '/_+/', 'a__b', 'replace', '_' ),
	array( '/^_+/', '__lead', 'replace', '' ),
	array( '/_+$/', 'trail__', 'replace', '' ),
	array( '/\s*<br\s*\/?>\s*/i', "a<br>b<br/>c", 'replace', "\n" ),
	array( '/\r\n|\r|\n/', "a\r\nb\nc", 'replace', "\n" ),
	array( '/^(www\.)/i', 'www.example.com', 'replace', '' ),
	array( '/[<>"\']/', '<a href="x">', 'replace', '' ),
	array( '/\p{L}+/u', 'café résumé', 'match' ),
	array( '/\d{1,3}(,\d{3})*(\.\d+)?/', '1,234.56', 'match' ),
	array( '/^(post|page|attachment)$/', 'post', 'match' ),
	array( '/^(post|page|attachment)$/', 'revision', 'match' ),
	array( '/\s+class=("|\')(.*?)\\1/i', '<div class="foo bar">', 'replace', '' ),
);

foreach ( $preg_cases as $i => $pc ) {
	$pattern = $pc[0];
	$subject = $pc[1];
	$mode    = $pc[2];
	if ( 'match' === $mode ) {
		$matches = array();
		$code    = @preg_match( $pattern, $subject, $matches );
		$out['preg_match'][] = array(
			'i'       => $i,
			'pattern' => $pattern,
			'subject' => $subject,
			'code'    => $code,
			'matches' => $matches,
		);
	} else {
		$replace = $pc[3];
		$result  = @preg_replace( $pattern, $replace, $subject );
		$out['preg_replace'][] = array(
			'i'       => $i,
			'pattern' => $pattern,
			'subject' => $subject,
			'replace' => $replace,
			'result'  => $result,
		);
	}
}

gopress_golden_write( 'phpx/vectors.json', $out );
WP_CLI::success( 'phpx dump complete' );
