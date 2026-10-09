#!/usr/bin/env bash
# P0.4: Build a deterministic fixture DB dump and uploads tree.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GOPRESS_REF_DIR="${GOPRESS_REF_DIR:-/opt/gopress-ref}"
ORACLE_ADDR="${ORACLE_ADDR:-127.0.0.1:8081}"
ORACLE_URL="http://${ORACLE_ADDR}"
OUT_SQL="${REPO_ROOT}/tests/fixtures/site.sql"
OUT_STATIC_SQL="${REPO_ROOT}/tests/fixtures/site-static.sql"
OUT_UPLOADS="${REPO_ROOT}/tests/fixtures/uploads"
export GOPRESS_REPO="${REPO_ROOT}"

log() { printf '+ %s\n' "$*" >&2; }
warn() { printf '! %s\n' "$*" >&2; }

dump_db() {
	local dest="$1"
	log "mysqldump → ${dest}"
	mysqldump \
		-ugopress -pgopress \
		--skip-dump-date \
		--skip-comments \
		--skip-extended-insert \
		--order-by-primary \
		--routines=false \
		--triggers=false \
		--default-character-set=utf8mb4 \
		gopress_ref \
		| sed -e '/^-- Dump completed on /d' \
		> "${dest}.tmp"
	mv "${dest}.tmp" "${dest}"
}

# Ensure oracle site files + server exist.
make -C "${REPO_ROOT}" oracle-up

log "resetting gopress_ref for a clean install"
wp db reset --yes --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
wp core install \
	--path="${GOPRESS_REF_DIR}" \
	--url="${ORACLE_URL}" \
	--title='GoPress Fixture' \
	--admin_user=admin \
	--admin_password=password \
	--admin_email=admin@example.com \
	--skip-email
wp theme activate twentytwentyfive --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"

# Clear uploads from prior runs inside the oracle root.
rm -rf "${GOPRESS_REF_DIR}/wp-content/uploads"
mkdir -p "${GOPRESS_REF_DIR}/wp-content/uploads"

log "running seed.php"
wp eval-file "${REPO_ROOT}/tools/oracle/seed.php" \
	--path="${GOPRESS_REF_DIR}" \
	--url="${ORACLE_URL}" \
	--user=admin

mkdir -p "$(dirname "${OUT_SQL}")"
dump_db "${OUT_SQL}"

log "copying uploads → ${OUT_UPLOADS}"
rm -rf "${OUT_UPLOADS}"
mkdir -p "${OUT_UPLOADS}"
if [[ -d "${GOPRESS_REF_DIR}/wp-content/uploads" ]]; then
	cp -a "${GOPRESS_REF_DIR}/wp-content/uploads/." "${OUT_UPLOADS}/"
fi

# Static front-page variant.
log "creating site-static.sql variant"
wp option update show_on_front page --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
HOME_ID="$(wp post list --post_type=page --name=home-static --field=ID --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}")"
BLOG_ID="$(wp post list --post_type=page --name=blog --field=ID --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}")"
wp option update page_on_front "${HOME_ID}" --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
wp option update page_for_posts "${BLOG_ID}" --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
# Re-freeze modified times and drop transients recreated by theme/option loads.
wp db query "UPDATE wp_posts SET post_modified='2024-06-15 12:00:00', post_modified_gmt='2024-06-15 12:00:00';" \
	--path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}" >/dev/null
wp db query "DELETE FROM wp_options WHERE option_name LIKE '\\_site\\_transient\\_%' OR option_name LIKE '\\_transient\\_%';" \
	--path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}" >/dev/null
dump_db "${OUT_STATIC_SQL}"

# Restore posts front for the live oracle (matches site.sql).
log "reloading site.sql into gopress_ref so the oracle matches the committed dump"
wp db reset --yes --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
wp db import "${OUT_SQL}" --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
rm -rf "${GOPRESS_REF_DIR}/wp-content/uploads"
mkdir -p "${GOPRESS_REF_DIR}/wp-content/uploads"
cp -a "${OUT_UPLOADS}/." "${GOPRESS_REF_DIR}/wp-content/uploads/"

log "fixture written:"
log "  ${OUT_SQL} ($(wc -c < "${OUT_SQL}") bytes)"
log "  ${OUT_STATIC_SQL} ($(wc -c < "${OUT_STATIC_SQL}") bytes)"
log "  ${OUT_UPLOADS}/"
