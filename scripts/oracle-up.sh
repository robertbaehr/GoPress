#!/usr/bin/env bash
# P0.3: Bring up the PHP oracle site at http://127.0.0.1:8081
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GOPRESS_REF_DIR="${GOPRESS_REF_DIR:-/opt/gopress-ref}"
ORACLE_ADDR="${ORACLE_ADDR:-127.0.0.1:8081}"
ORACLE_URL="http://${ORACLE_ADDR}"
TMUX_SESSION="${ORACLE_TMUX_SESSION:-oracle}"

log() { printf '+ %s\n' "$*" >&2; }
warn() { printf '! %s\n' "$*" >&2; }

tmux_cmd() {
	if [[ -f /exec-daemon/tmux.portal.conf ]]; then
		tmux -f /exec-daemon/tmux.portal.conf "$@"
	else
		tmux "$@"
	fi
}

if [[ ! -f "${GOPRESS_REF_DIR}/wp-includes/version.php" ]]; then
	warn "${GOPRESS_REF_DIR} is missing; run scripts/dev-setup.sh first (P0.1–P0.2)"
	exit 1
fi

log "installing oracle config, router, and mu-plugin into ${GOPRESS_REF_DIR}"
# Template is already a complete wp-config.php (no substitutions required).
cp "${REPO_ROOT}/tools/oracle/wp-config.php.tmpl" "${GOPRESS_REF_DIR}/wp-config.php"
cp "${REPO_ROOT}/tools/oracle/router.php" "${GOPRESS_REF_DIR}/router.php"
mkdir -p "${GOPRESS_REF_DIR}/wp-content/mu-plugins"
cp "${REPO_ROOT}/tools/oracle/mu-plugins/gopress-parity.php" \
	"${GOPRESS_REF_DIR}/wp-content/mu-plugins/gopress-parity.php"

log "removing classic themes (keep twentytwentytwo–twentytwentyfive)"
keep='twentytwentytwo|twentytwentythree|twentytwentyfour|twentytwentyfive'
for theme_dir in "${GOPRESS_REF_DIR}/wp-content/themes"/*/; do
	[[ -d "${theme_dir}" ]] || continue
	name="$(basename "${theme_dir}")"
	if [[ "${name}" =~ ^(${keep})$ ]]; then
		continue
	fi
	log "  delete theme ${name}"
	rm -rf "${theme_dir}"
done

# Ensure MariaDB is up and the oracle DB exists.
if ! mysql -ugopress -pgopress -e 'select 1' >/dev/null 2>&1; then
	warn "cannot connect as gopress; run scripts/dev-setup.sh"
	exit 1
fi

log "starting PHP built-in server in tmux session '${TMUX_SESSION}' (${ORACLE_ADDR})"
if tmux_cmd has-session -t "=${TMUX_SESSION}" 2>/dev/null; then
	log "tmux session '${TMUX_SESSION}' already exists"
else
	tmux_cmd new-session -d -s "${TMUX_SESSION}" -c "${GOPRESS_REF_DIR}" -- \
		env PHP_CLI_SERVER_WORKERS=4 \
		php -S "${ORACLE_ADDR}" -t "${GOPRESS_REF_DIR}" "${GOPRESS_REF_DIR}/router.php"
fi

# Wait for the server to accept connections.
for _ in $(seq 1 30); do
	if curl -sf -o /dev/null "${ORACLE_URL}/wp-login.php"; then
		break
	fi
	sleep 0.5
done

if ! wp core is-installed --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}" 2>/dev/null; then
	log "installing WordPress into gopress_ref"
	wp core install \
		--path="${GOPRESS_REF_DIR}" \
		--url="${ORACLE_URL}" \
		--title='GoPress Fixture' \
		--admin_user=admin \
		--admin_password=password \
		--admin_email=admin@example.com \
		--skip-email
	# Activate a block theme so the front page renders wp-site-blocks.
	wp theme activate twentytwentyfive --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
else
	log "WordPress already installed"
fi

log "oracle ready at ${ORACLE_URL}"
# Avoid `curl | grep -q` (curl exits 23 when grep closes the pipe early).
front_html="$(mktemp)"
curl -sf "${ORACLE_URL}/" -o "${front_html}"
if ! grep -q wp-site-blocks "${front_html}"; then
	warn "front page at ${ORACLE_URL}/ does not contain wp-site-blocks"
	rm -f "${front_html}"
	exit 1
fi
rm -f "${front_html}"
log "front page contains wp-site-blocks"
