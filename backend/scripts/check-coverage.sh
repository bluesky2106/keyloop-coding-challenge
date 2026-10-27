#!/bin/sh
# Fails if statement coverage in a Go cover profile is below a minimum.
# internal/testdb is test infrastructure, not production code, so it is excluded.
# Usage: check-coverage.sh <profile> <min-percent>
set -eu

profile=${1:?profile path required}
min=${2:?minimum percent required}

awk -v min="$min" '
	NR == 1 || $0 ~ /internal\/testdb\// { next }
	{
		stmts[$1] = $2
		if ($3 > 0) hit[$1] = 1
	}
	END {
		for (block in stmts) {
			total += stmts[block]
			if (block in hit) covered += stmts[block]
		}
		pct = total ? 100 * covered / total : 0
		printf "coverage: %.1f%% of %d statements (minimum %s%%)\n", pct, total, min
		exit (pct < min)
	}
' "$profile"
