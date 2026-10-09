#!/usr/bin/env bash
# P0.1: Idempotent GoPress development toolchain setup.
# Safe to run repeatedly. Requires sudo for apt and MariaDB admin.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PHP_REFERENCE_SHA="${PHP_REFERENCE_SHA:-02a3c7e0e2}"
PHP_REFERENCE_TAG="${PHP_REFERENCE_TAG:-php-reference}"
GOPRESS_DB_USER="${GOPRESS_DB_USER:-gopress}"
GOPRESS_DB_PASS="${GOPRESS_DB_PASS:-gopress}"
GOPATH_DIR="${GOPATH:-${HOME}/go}"
export PATH="${GOPATH_DIR}/bin:${PATH}"

log() { printf '+ %s\n' "$*"; }
warn() { printf '! %s\n' "$*" >&2; }

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		warn "required command not found after setup: $1"
		return 1
	fi
}

# --- 1. Tag the pinned WordPress reference commit ---
tag_php_reference() {
	cd "${REPO_ROOT}"
	if ! git rev-parse --verify "${PHP_REFERENCE_SHA}^{commit}" >/dev/null 2>&1; then
		warn "commit ${PHP_REFERENCE_SHA} not found locally; fetching origin"
		git fetch origin "${PHP_REFERENCE_SHA}" || git fetch origin
	fi
	local full_sha
	full_sha="$(git rev-parse "${PHP_REFERENCE_SHA}^{commit}")"

	if git rev-parse -q --verify "refs/tags/${PHP_REFERENCE_TAG}" >/dev/null; then
		local existing
		existing="$(git rev-parse "refs/tags/${PHP_REFERENCE_TAG}^{commit}")"
		if [[ "${existing}" != "${full_sha}" ]]; then
			warn "tag ${PHP_REFERENCE_TAG} points at ${existing}, expected ${full_sha}"
			exit 1
		fi
		log "tag ${PHP_REFERENCE_TAG} already at ${full_sha}"
	else
		log "creating tag ${PHP_REFERENCE_TAG} at ${full_sha}"
		git tag "${PHP_REFERENCE_TAG}" "${full_sha}"
	fi

	if git ls-remote --exit-code --tags origin "refs/tags/${PHP_REFERENCE_TAG}" >/dev/null 2>&1; then
		local remote_sha
		remote_sha="$(git ls-remote --tags origin "refs/tags/${PHP_REFERENCE_TAG}" | awk '{print $1; exit}')"
		# Annotated tags show ^{} peel; handle both.
		local remote_peeled
		remote_peeled="$(git ls-remote --tags origin "refs/tags/${PHP_REFERENCE_TAG}^{}" 2>/dev/null | awk '{print $1; exit}' || true)"
		if [[ -n "${remote_peeled}" ]]; then
			remote_sha="${remote_peeled}"
		fi
		if [[ "${remote_sha}" != "${full_sha}" ]]; then
			warn "remote tag ${PHP_REFERENCE_TAG} is ${remote_sha}, expected ${full_sha}"
			exit 1
		fi
		log "remote tag ${PHP_REFERENCE_TAG} already pushed"
	else
		log "pushing tag ${PHP_REFERENCE_TAG} to origin"
		git push origin "refs/tags/${PHP_REFERENCE_TAG}"
	fi
}

# --- 2. Install OS packages ---
install_packages() {
	log "installing apt packages"
	sudo apt-get update
	sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
		mariadb-server \
		php8.3-cli \
		php8.3-mysql \
		php8.3-gd \
		php-imagick \
		php8.3-intl \
		php8.3-mbstring \
		php8.3-xml \
		php8.3-zip \
		php8.3-curl \
		libvips-dev \
		pkg-config \
		unzip \
		curl \
		ca-certificates
}

# --- 3. Start MariaDB ---
start_mariadb() {
	if command -v mysqladmin >/dev/null 2>&1 && mysqladmin ping >/dev/null 2>&1; then
		log "MariaDB already running"
		return 0
	fi
	if sudo service mariadb start 2>/dev/null; then
		log "started MariaDB via service"
		# Wait briefly for the socket.
		for _ in $(seq 1 30); do
			if mysqladmin ping >/dev/null 2>&1 || sudo mysqladmin ping >/dev/null 2>&1; then
				return 0
			fi
			sleep 1
		done
	fi
	warn "service mariadb start failed or timed out; launching mysqld_safe in tmux session 'mariadb'"
	if ! command -v tmux >/dev/null 2>&1; then
		sudo DEBIAN_FRONTEND=noninteractive apt-get install -y tmux
	fi
	if ! tmux -f /exec-daemon/tmux.portal.conf has-session -t "=mariadb" 2>/dev/null \
		&& ! tmux has-session -t "=mariadb" 2>/dev/null; then
		# Prefer portal tmux config when present (Cursor Cloud agents).
		if [[ -f /exec-daemon/tmux.portal.conf ]]; then
			tmux -f /exec-daemon/tmux.portal.conf new-session -d -s mariadb -- sudo mysqld_safe
		else
			tmux new-session -d -s mariadb -- sudo mysqld_safe
		fi
	fi
	for _ in $(seq 1 60); do
		if sudo mysqladmin ping >/dev/null 2>&1; then
			log "MariaDB ready via mysqld_safe"
			return 0
		fi
		sleep 1
	done
	warn "MariaDB did not become ready"
	exit 1
}

# --- 4. Create databases and application user ---
setup_databases() {
	log "ensuring databases and user ${GOPRESS_DB_USER}"
	# Prefer socket auth as root via sudo (Ubuntu/MariaDB default).
	sudo mysql <<SQL
CREATE DATABASE IF NOT EXISTS gopress_ref CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS gopress_go CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS gopress_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${GOPRESS_DB_USER}'@'localhost' IDENTIFIED BY '${GOPRESS_DB_PASS}';
CREATE USER IF NOT EXISTS '${GOPRESS_DB_USER}'@'%' IDENTIFIED BY '${GOPRESS_DB_PASS}';
ALTER USER '${GOPRESS_DB_USER}'@'localhost' IDENTIFIED BY '${GOPRESS_DB_PASS}';
ALTER USER '${GOPRESS_DB_USER}'@'%' IDENTIFIED BY '${GOPRESS_DB_PASS}';
GRANT ALL PRIVILEGES ON gopress_ref.* TO '${GOPRESS_DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON gopress_go.* TO '${GOPRESS_DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON gopress_test.* TO '${GOPRESS_DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON gopress_ref.* TO '${GOPRESS_DB_USER}'@'%';
GRANT ALL PRIVILEGES ON gopress_go.* TO '${GOPRESS_DB_USER}'@'%';
GRANT ALL PRIVILEGES ON gopress_test.* TO '${GOPRESS_DB_USER}'@'%';
FLUSH PRIVILEGES;
SQL
}

# --- 5. Install wp-cli ---
install_wp_cli() {
	local dest="/usr/local/bin/wp"
	if [[ -x "${dest}" ]] && "${dest}" --info >/dev/null 2>&1; then
		log "wp-cli already installed at ${dest}"
		return 0
	fi
	log "installing wp-cli to ${dest}"
	local tmp
	tmp="$(mktemp)"
	curl -fsSL -o "${tmp}" https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar
	chmod +x "${tmp}"
	sudo mv "${tmp}" "${dest}"
}

# --- 6. Install Go tools ---
install_go_tools() {
	if ! command -v go >/dev/null 2>&1; then
		warn "go is not installed; expected Go 1.22+ on this VM"
		exit 1
	fi
	# Pin to a release that builds with Go 1.22 (the VM baseline). @latest
	# currently requires Go >= 1.26 and would trigger an auto toolchain download.
	# P0.8 formalizes pinned tool versions in the Makefile.
	local staticcheck_ver="${STATICCHECK_VERSION:-v0.5.1}"
	log "installing staticcheck@${staticcheck_ver} into ${GOPATH_DIR}/bin"
	GOBIN="${GOPATH_DIR}/bin" GOTOOLCHAIN=local go install "honnef.co/go/tools/cmd/staticcheck@${staticcheck_ver}"
}

# --- Verification (Done when) ---
verify() {
	log "verifying toolchain"
	php -v
	mysql -u"${GOPRESS_DB_USER}" -p"${GOPRESS_DB_PASS}" -e 'select 1'
	wp --info
	go version
	pkg-config --modversion vips
	# Extra: staticcheck must be reachable via GOPATH/bin (exported above).
	staticcheck -version
	log "all P0.1 checks passed"
}

main() {
	log "GoPress P0.1 dev-setup (repo=${REPO_ROOT})"
	tag_php_reference
	install_packages
	start_mariadb
	setup_databases
	install_wp_cli
	install_go_tools
	verify
}

main "$@"
