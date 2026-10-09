#!/usr/bin/env bash
# P0.1–P0.2: Idempotent GoPress development toolchain and upstream checkouts.
# Safe to run repeatedly. Requires sudo for apt, MariaDB admin, and /opt writes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PHP_REFERENCE_SHA="${PHP_REFERENCE_SHA:-02a3c7e0e2}"
PHP_REFERENCE_TAG="${PHP_REFERENCE_TAG:-php-reference}"
GOPRESS_DB_USER="${GOPRESS_DB_USER:-gopress}"
GOPRESS_DB_PASS="${GOPRESS_DB_PASS:-gopress}"
GOPATH_DIR="${GOPATH:-${HOME}/go}"
GOPRESS_REF_DIR="${GOPRESS_REF_DIR:-/opt/gopress-ref}"
WORDPRESS_DEVELOP_DIR="${WORDPRESS_DEVELOP_DIR:-/opt/wordpress-develop}"
GUTENBERG_DIR="${GUTENBERG_DIR:-/opt/gutenberg}"
export PATH="${GOPATH_DIR}/bin:${PATH}"

log() { printf '+ %s\n' "$*" >&2; }
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

# --- P0.2.1 Oracle site tree from php-reference ---
setup_gopress_ref() {
	cd "${REPO_ROOT}"
	local full_sha
	full_sha="$(git rev-parse "${PHP_REFERENCE_TAG}^{commit}")"
	local marker="${GOPRESS_REF_DIR}/.gopress-php-reference-sha"

	if [[ -f "${marker}" ]] && [[ "$(cat "${marker}")" == "${full_sha}" ]] \
		&& [[ -f "${GOPRESS_REF_DIR}/wp-includes/version.php" ]]; then
		log "oracle site ${GOPRESS_REF_DIR} already at ${full_sha}"
		return 0
	fi

	log "extracting ${PHP_REFERENCE_TAG} (${full_sha}) into ${GOPRESS_REF_DIR}"
	sudo mkdir -p "${GOPRESS_REF_DIR}"
	# Clear previous extract but keep the directory mount point.
	sudo find "${GOPRESS_REF_DIR}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
	git archive "${PHP_REFERENCE_TAG}" | sudo tar -x -C "${GOPRESS_REF_DIR}"
	printf '%s\n' "${full_sha}" | sudo tee "${marker}" >/dev/null
	# Make the tree writable by the invoking user for later oracle config (P0.3).
	if [[ -n "${SUDO_USER:-}" ]]; then
		sudo chown -R "${SUDO_USER}:${SUDO_USER}" "${GOPRESS_REF_DIR}"
	elif [[ "$(id -u)" -ne 0 ]]; then
		sudo chown -R "$(id -u):$(id -g)" "${GOPRESS_REF_DIR}"
	fi
	log "oracle site ready: ${GOPRESS_REF_DIR} @ ${full_sha}"
}

# --- P0.2.2 wordpress-develop at trunk@64260 ---
setup_wordpress_develop() {
	local repo_url="https://github.com/WordPress/wordpress-develop.git"
	local trunk_marker="trunk@64260"

	if [[ ! -d "${WORDPRESS_DEVELOP_DIR}/.git" ]]; then
		log "cloning wordpress-develop into ${WORDPRESS_DEVELOP_DIR}"
		sudo mkdir -p "$(dirname "${WORDPRESS_DEVELOP_DIR}")"
		# Clone as root into /opt, then chown.
		if [[ -e "${WORDPRESS_DEVELOP_DIR}" ]]; then
			sudo rm -rf "${WORDPRESS_DEVELOP_DIR}"
		fi
		sudo git clone --filter=blob:none "${repo_url}" "${WORDPRESS_DEVELOP_DIR}"
		if [[ "$(id -u)" -ne 0 ]]; then
			sudo chown -R "$(id -u):$(id -g)" "${WORDPRESS_DEVELOP_DIR}"
		fi
	else
		log "fetching wordpress-develop updates"
		git -C "${WORDPRESS_DEVELOP_DIR}" fetch --all --tags --prune
	fi

	local develop_sha
	develop_sha="$(git -C "${WORDPRESS_DEVELOP_DIR}" log --all --grep="${trunk_marker}" --format=%H | head -1)"
	if [[ -z "${develop_sha}" ]]; then
		warn "no commit matching ${trunk_marker} in ${WORDPRESS_DEVELOP_DIR}"
		exit 1
	fi
	log "checking out wordpress-develop ${develop_sha} (${trunk_marker})"
	git -C "${WORDPRESS_DEVELOP_DIR}" checkout --detach "${develop_sha}"
	printf '%s\n' "${develop_sha}"
}

# Resolve a Gutenberg ref matching wordpress-develop's package.json.
resolve_gutenberg_ref() {
	local pkg="${WORDPRESS_DEVELOP_DIR}/package.json"
	if [[ ! -f "${pkg}" ]]; then
		warn "missing ${pkg}"
		exit 1
	fi

	# Prefer package.json "gutenberg.sha" (current wordpress-develop layout),
	# then a string "gutenberg" key, then @wordpress/block-library version → tag.
	# PHP prints "sha|<ref>" or "tag|<version>" or nothing.
	local resolved
	resolved="$(
		php -r '
			$j = json_decode(file_get_contents($argv[1]), true);
			if (isset($j["gutenberg"]["sha"]) && is_string($j["gutenberg"]["sha"]) && $j["gutenberg"]["sha"] !== "") {
				echo "sha|" . $j["gutenberg"]["sha"];
				exit(0);
			}
			if (isset($j["gutenberg"]) && is_string($j["gutenberg"]) && $j["gutenberg"] !== "") {
				echo "sha|" . $j["gutenberg"];
				exit(0);
			}
			foreach (["dependencies", "devDependencies"] as $k) {
				if (isset($j[$k]["@wordpress/block-library"])) {
					echo "tag|" . $j[$k]["@wordpress/block-library"];
					exit(0);
				}
			}
			exit(1);
		' "${pkg}"
	)" || {
		warn "could not find gutenberg.sha or @wordpress/block-library in ${pkg}"
		exit 1
	}

	local kind="${resolved%%|*}"
	local value="${resolved#*|}"
	if [[ "${kind}" == "tag" ]]; then
		value="${value#^}"
		value="${value#~}"
		value="${value#=}"
		value="${value#v}"
		printf 'v%s\n' "${value}"
		return 0
	fi
	printf '%s\n' "${value}"
}

# --- P0.2.3 Gutenberg at the matching ref ---
setup_gutenberg() {
	local repo_url="https://github.com/WordPress/gutenberg.git"
	local ref
	ref="$(resolve_gutenberg_ref)"
	log "resolved Gutenberg ref: ${ref}"

	if [[ ! -d "${GUTENBERG_DIR}/.git" ]]; then
		log "cloning gutenberg into ${GUTENBERG_DIR}"
		if [[ -e "${GUTENBERG_DIR}" ]]; then
			sudo rm -rf "${GUTENBERG_DIR}"
		fi
		sudo git clone --filter=blob:none "${repo_url}" "${GUTENBERG_DIR}"
		if [[ "$(id -u)" -ne 0 ]]; then
			sudo chown -R "$(id -u):$(id -g)" "${GUTENBERG_DIR}"
		fi
	else
		log "fetching gutenberg updates"
		git -C "${GUTENBERG_DIR}" fetch --all --tags --prune
	fi

	# Ensure the commit/tag is present (shallow clones or new SHAs may need a fetch).
	if ! git -C "${GUTENBERG_DIR}" rev-parse --verify "${ref}^{commit}" >/dev/null 2>&1; then
		log "fetching Gutenberg ref ${ref}"
		git -C "${GUTENBERG_DIR}" fetch --depth=1 origin "${ref}" \
			|| git -C "${GUTENBERG_DIR}" fetch origin "${ref}" \
			|| git -C "${GUTENBERG_DIR}" fetch --unshallow 2>/dev/null \
			|| true
	fi

	# Accept full SHA, short SHA, branch, or tag (with or without v prefix).
	if git -C "${GUTENBERG_DIR}" rev-parse --verify "${ref}^{commit}" >/dev/null 2>&1; then
		git -C "${GUTENBERG_DIR}" checkout --detach "${ref}"
	elif [[ "${ref}" == v* ]] \
		&& git -C "${GUTENBERG_DIR}" rev-parse --verify "${ref#v}^{commit}" >/dev/null 2>&1; then
		git -C "${GUTENBERG_DIR}" checkout --detach "${ref#v}"
	else
		warn "Gutenberg ref ${ref} not found after fetch"
		exit 1
	fi
	git -C "${GUTENBERG_DIR}" rev-parse HEAD
}

# --- Verification (Done when) ---
verify() {
	log "verifying toolchain (P0.1)"
	php -v
	mysql -u"${GOPRESS_DB_USER}" -p"${GOPRESS_DB_PASS}" -e 'select 1'
	wp --info
	go version
	pkg-config --modversion vips
	# Extra: staticcheck must be reachable via GOPATH/bin (exported above).
	staticcheck -version
	log "all P0.1 checks passed"
}

verify_upstream_checkouts() {
	log "verifying upstream checkouts (P0.2)"
	local ref_sha develop_sha gutenberg_sha

	[[ -f "${GOPRESS_REF_DIR}/wp-includes/version.php" ]] || {
		warn "missing ${GOPRESS_REF_DIR}/wp-includes/version.php"
		exit 1
	}
	ref_sha="$(cat "${GOPRESS_REF_DIR}/.gopress-php-reference-sha")"
	[[ "${ref_sha}" == "$(git -C "${REPO_ROOT}" rev-parse "${PHP_REFERENCE_TAG}^{commit}")" ]]

	develop_sha="$(git -C "${WORDPRESS_DEVELOP_DIR}" rev-parse HEAD)"
	gutenberg_sha="$(git -C "${GUTENBERG_DIR}" rev-parse HEAD)"

	# Confirm develop commit still mentions trunk@64260 (usually in the
	# git-svn-id footer of the body, not the subject line).
	if ! git -C "${WORDPRESS_DEVELOP_DIR}" log -1 --format=%B "${develop_sha}" | grep -q 'trunk@64260'; then
		warn "wordpress-develop HEAD ${develop_sha} does not mention trunk@64260"
		exit 1
	fi

	# Print SHAs to stdout so the Done-when criterion is visible in logs.
	printf 'SHAs:\n'
	printf '  gopress-ref (php-reference): %s\n' "${ref_sha}"
	printf '  wordpress-develop:           %s\n' "${develop_sha}"
	printf '  gutenberg:                   %s\n' "${gutenberg_sha}"
	log "all P0.2 checks passed"
}

main() {
	log "GoPress P0.1–P0.2 dev-setup (repo=${REPO_ROOT})"
	tag_php_reference
	install_packages
	start_mariadb
	setup_databases
	install_wp_cli
	install_go_tools
	verify
	setup_gopress_ref
	setup_wordpress_develop >/tmp/gopress-wordpress-develop.sha
	setup_gutenberg >/tmp/gopress-gutenberg.sha
	verify_upstream_checkouts
}

main "$@"
