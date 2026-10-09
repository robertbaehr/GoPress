#!/usr/bin/env bash
# P0.5: Regenerate tests/golden/ from the PHP oracle.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GOPRESS_REF_DIR="${GOPRESS_REF_DIR:-/opt/gopress-ref}"
ORACLE_ADDR="${ORACLE_ADDR:-127.0.0.1:8081}"
ORACLE_URL="http://${ORACLE_ADDR}"
export GOPRESS_REPO="${REPO_ROOT}"

log() { printf '+ %s\n' "$*" >&2; }

make -C "${REPO_ROOT}" oracle-up

log "loading fixture DB into gopress_ref"
wp db reset --yes --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
wp db import "${REPO_ROOT}/tests/fixtures/site.sql" --path="${GOPRESS_REF_DIR}" --url="${ORACLE_URL}"
rm -rf "${GOPRESS_REF_DIR}/wp-content/uploads"
mkdir -p "${GOPRESS_REF_DIR}/wp-content/uploads"
if [[ -d "${REPO_ROOT}/tests/fixtures/uploads" ]]; then
	cp -a "${REPO_ROOT}/tests/fixtures/uploads/." "${GOPRESS_REF_DIR}/wp-content/uploads/"
fi

mkdir -p "${REPO_ROOT}/tests/golden"

# Order matters little; admin-menu spawns child processes per role.
dumps=(
	phpx.php
	auth.php
	formatting.php
	kses.php
	options.php
	roles.php
	caps.php
	post-types.php
	taxonomies.php
	image-sizes.php
	registered-settings.php
	registered-meta.php
	blocks.php
	assets.php
	rewrite.php
	theme-json.php
	editor-settings.php
	admin-menu.php
	rest-index.php
	dates.php
)

for dump in "${dumps[@]}"; do
	script="${REPO_ROOT}/tools/oracle/dump/${dump}"
	if [[ ! -f "${script}" ]]; then
		log "skip missing ${dump}"
		continue
	fi
	log "dump ${dump}"
	wp eval-file "${script}" \
		--path="${GOPRESS_REF_DIR}" \
		--url="${ORACLE_URL}" \
		--user=admin
done

log "golden regeneration complete ($(find "${REPO_ROOT}/tests/golden" -type f | wc -l) files)"
