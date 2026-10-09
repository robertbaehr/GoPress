# GoPress development Makefile.
# Additional targets (build, test, check, golden, fixture, …) land in P0.4–P0.6.

.PHONY: oracle-up

# P0.3: configure /opt/gopress-ref, start the PHP oracle on :8081, install if needed.
oracle-up:
	@bash scripts/oracle-up.sh
