#!/bin/sh
# Smoke test for the running containers (`make up`): exercises the app through the one
# published port, the way a browser does. Exits non-zero on the first failure.
set -eu

base="http://localhost:${WEB_PORT:-8080}"
fail() { echo "FAIL: $*" >&2; exit 1; }
json() { python3 -c "import sys,json; print(json.load(sys.stdin)$1)"; }

# The single-page app and its client-side routes are served.
[ "$(curl -s -o /dev/null -w '%{http_code}' "$base/")" = 200 ] || fail "the app is not served at $base/"
[ "$(curl -s -o /dev/null -w '%{http_code}' "$base/vehicles/anything")" = 200 ] || fail "client-side routes are not served"

# The API is reachable through the same origin and refuses anonymous requests.
[ "$(curl -s -o /dev/null -w '%{http_code}' "$base/api/v1/me")" = 401 ] || fail "/api/v1/me should be 401 without a token"

# Sign in as a demo user and read the demo data the seed container loaded.
token=$(curl -fsS -X POST "$base/api/v1/auth/login" \
  -d '{"username":"northside.manager","password":"demo-northside-1"}' | json "['access_token']") \
  || fail "login failed"
auth="Authorization: Bearer $token"

total=$(curl -fsS -H "$auth" "$base/api/v1/vehicles/summary" | json "['total']")
[ "$total" = 50 ] || fail "expected 50 demo vehicles, got $total"

# Log an action on an aging vehicle, and see it come back.
id=$(curl -fsS -H "$auth" "$base/api/v1/vehicles?aging=true&page_size=1" | json "['items'][0]['id']")
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "$auth" "$base/api/v1/vehicles/$id/actions" \
  -d '{"action_type":"hold","note":"smoke test"}')
[ "$code" = 201 ] || fail "logging an action returned $code, expected 201"
action=$(curl -fsS -H "$auth" "$base/api/v1/vehicles/$id" | json "['current_action']['action_type']")
[ "$action" = hold ] || fail "the logged action is not the current action"

# A vehicle that is not aging must refuse one.
young=$(curl -fsS -H "$auth" "$base/api/v1/vehicles?aging=false&page_size=1" | json "['items'][0]['id']")
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "$auth" "$base/api/v1/vehicles/$young/actions" \
  -d '{"action_type":"hold"}')
[ "$code" = 409 ] || fail "an action on a vehicle that is not aging returned $code, expected 409"

echo "smoke test passed"
