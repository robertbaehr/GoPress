# GoPress development Makefile.
# Additional targets (build, test, check, golden, …) land in P0.5–P0.6.

.PHONY: oracle-up fixture

# P0.3: configure /opt/gopress-ref, start the PHP oracle on :8081, install if needed.
oracle-up:
	@bash scripts/oracle-up.sh

# P0.4: rebuild tests/fixtures/site.sql (+ site-static.sql) and uploads/ from a clean seed.
fixture:
	@bash scripts/fixture.sh
