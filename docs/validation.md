# Validation

How the code in this repository was checked, what the checking found, and what is still
unverified. The code was written with an AI agent (Claude Code) under direction; this
document is the record of how that output was verified rather than trusted.

## How to reproduce

```sh
make db-up          # a database for the tests
make check          # lint, types, API-spec lint, generated-types check, all tests, coverage gates
make up && make smoke   # build the images, run the whole app, exercise it end to end
```

| Check | Result when this was written |
|---|---|
| Backend tests (Go, `-race`) | 273 passing, including subtests |
| Backend statement coverage | 95.3% of 571 statements (gate: 90%, enforced by `make test`) |
| Frontend tests (Vitest) | 107 passing |
| Frontend coverage | 99.3% statements, 97.0% branches, 99.0% functions, 99.6% lines (gates: 90 / 85 / 90 / 90) |
| Linters | `golangci-lint` v2 (`gosec`, `revive`, `errorlint`, `noctx` and more), ESLint with type-checked strict rules and `jsx-a11y`, Prettier, `tsc`: 0 issues |
| API contract | `docs/api/openapi.yaml` lints clean; every response in the router tests is validated against it |
| API reference (`/docs`) | Driven in headless Chrome against the running containers: 8 operations listed, server `/api/v1`, authorized with a token from `POST /auth/login`, executed `GET /vehicles/summary` from the UI and got the demo totals, no Content-Security-Policy violations. The smoke test checks the redirect, the page, the bundle, a 404 for a missing file and that the served spec is byte-identical to `docs/api/openapi.yaml` |
| Container smoke test | `scripts/smoke.sh`: app and client-side routes served, anonymous request refused, login, demo data present, action logged and read back, non-aging vehicle refused |

CI runs `make check` and the container smoke test on every pull request. This document does
not report CI results, because they are visible on the pull requests and not something this
text should assert.

## What the checks are, and why they can be trusted

**Real database, not mocks, for anything that touches SQL.** Store tests and the router
tests run against PostgreSQL 18.4, each in a throwaway database with the real migrations.
Mocks (generated with mockery) are used only to test a layer in isolation from the one
below it, for example the handler against a fake service, where they make error paths
reachable that a healthy database never produces.

**The schema is tested, not just described.** `backend/migrations/schema_test.go` asserts
that each integrity rule in `docs/database-design.md` is enforced by the database, and
pins each case to its exact Postgres error code, so a rule cannot pass by failing for the
wrong reason.

**The contract is enforced by tests, in both directions.**
- Backend: every response in the router tests is validated against the OpenAPI document.
  The checker is itself tested: it must reject a missing field, a wrong type, a wrong
  constant, an undocumented status code, the wrong content type and a malformed error, and it
  must match every real route, so it cannot silently check nothing.
- Frontend: types are generated from the same document, and `npm run check:api` (part of
  `make lint`) fails if the generated file is stale. The action labels are a complete
  `Record<ActionType, string>`, so a new action type is a compile error until it has a label.

**The rules the brief cares about have direct tests.** The 90-day boundary (exactly 90 days
is not aging, 91 is) is tested in the store, in the seed data and through the API. The UI
shows whatever `is_aging` the API reports. Tenant isolation is tested on the vehicle and
action read and write paths: another dealership's vehicle is a 404, and a token for one
dealership never resolves to another's data. A refused action
(not aging, invalid input) is checked to have stored nothing.

**Frontend tests run the real client over a mocked network** (MSW), and any request a test
did not describe fails the test. They query by role and label, as a user and assistive
technology would, not by test ids.

**Visual check.** I drove a headless Chrome against the running containers and inspected
screenshots of every screen and state: login (empty and rejected), inventory (default, aging
filter, rejected filters, no results, with a logged action), vehicle page (aging, not aging,
not found, after logging an action), with real keyboard focus, hover and validation states,
in light and dark themes at 1280, 820 and 390 px widths. A first pass found the design below
production quality and a second pass after the fixes checked each change. This is a manual
check, not an automated browser or visual-regression suite, so it will not catch a future
regression.

## What verification found

The value of the checks is what they caught in the AI's first attempts. Each item was found
by a test, a linter, a review pass or by running the thing, then fixed and given a regression
test where one applies.

**Bugs in the code**
- The inventory list query passed an unused argument to the count statement. Postgres rejects
  that, so the default list request would have failed. Found by the store tests.
- Extreme values (`max_age_days=2147483647`) caused a 500 through a date overflow. A review pass
  proposed a fix that only handled 32-bit overflow; probing against a real database showed the
  overflow remained, so the bounds are capped (100000 days) and documented in the spec.
- A note containing a NUL character passed validation and then failed the insert with a 500.
- Frontend: focus was lost when tabbing between the age fields (the component was remounted
  on every URL change); pressing Enter in the age form did nothing (two text fields and no
  submit button); labels wrapped their counters and errors, polluting their accessible names;
  the vehicle query retried 404s.
- After signing out, the next user to sign in on the same tab was sent to the previous user's
  page. A review pass proposed a fix that did not work (the redirect it tried to avoid still
  won); a test written for it failed, and the cause was fixed by recording why a session ended.
- Building the images failed because the seed program and the seed data directory both wanted
  the path `/app/seed`.
- Screenshots showed spacing and wrapping defects that no test could: two cards touching, a
  heading with excess top margin, and the "Sign out" button wrapping on a phone.
- A second visual pass found, among others: on a phone the table scrolled sideways so the Aging
  badge and the current action were off-screen (only the row tint remained); the age inputs sat
  5px higher than the selects next to them; the Model placeholder was truncated; the success
  message had double the intended spacing; the notice on a non-aging vehicle was misaligned with
  the details card. Native selects and a plain login card also looked unfinished. The interaction
  states were driven with real key events; an earlier version of my script silently filled
  nothing in, which showed up as screenshots with empty forms, so the script now reports
  JavaScript errors.

**Found by review passes and fixed with tests**
- After signing in from a deep link the app could pass through the inventory page, firing
  list requests, before reaching the destination. It now redirects once, straight there, and a
  test asserts that no list request is made. (This was flagged by a review, not observed first.)
- Errors from a failed make or model list were silent; the "Action logged." message stayed on
  screen while the next action was typed; logging an action waited for three refetches before
  reporting success; a model in the URL without a make filtered the list with no visible control.

**Mistakes in the AI's own claims and tests, corrected**
- It stated that `main()` was excluded from the coverage figure when the gate only excluded
  the test helper package. The docs and pull request were corrected to match the script.
- A test named as checking that the handler's action types match the database only counted
  them. It was replaced with a test that the database accepts every type the handler accepts.
- The first schema test would have passed for the wrong reason (an empty string failed UUID
  parsing, not the `NOT NULL` rule); it now asserts the error code.
- Several tests decoded JSON into structs without tags, so fields silently stayed empty, and
  several asserted on a request before it had been sent. Both were fixed.
- A mocking approach did not work for a call whose argument count varies, and was replaced by
  a small fake for that one case.

**Found while adding the API reference**
- `/docs` redirected to `/docs/` using nginx's own port (8080), so on any other published port (`WEB_PORT`) it would have sent the browser to the wrong place. The redirect is now relative, and the smoke test asserts it. This surfaced because my own test printed a URL that was not the one I had requested.
- My first test of whether the CSP exception is needed was invalid: the browser profile had cached the wrong permanent redirect, so it tested the real app, not the strict copy. After clearing the profile it showed two blocked `data:` images, which is why the exception exists.
- The first design used a Compose named build context to reach the spec; the installed Compose (2.13) predates that feature, so the web image is built from the repository root instead.

**Tooling and dependency problems**
- `typescript-eslint` did not support TypeScript 7, so TypeScript is pinned to 6.0; one
  generator declares a TypeScript 5 peer and is resolved with an npm `overrides` entry that the
  generated-types check exercises on every lint.
- Postgres 18 images keep data under `/var/lib/postgresql`; the volume mount was changed.
- An unpinned Redocly version was pinned; the MSW major version renamed an option.

**Review suggestions deliberately not taken**
- Equalising the response time of an unknown username with a wrong password (a decision to
  keep the code simple; see the known gaps).
- Returning 413 for oversized bodies (the API contract has no 413).
- Replacing, not adding, history entries per filter change (shareable URLs are intended).

## Known gaps

These are real limitations, not oversights to be discovered:

- **No rate limiting or lockout on login.** Combined with the timing difference below, valid
  usernames could be enumerated and passwords guessed online. Listed as future work.
- **Login timing.** An unknown username skips the bcrypt comparison, so it answers faster than
  a wrong password. The response body is identical.
- **The token is kept in `sessionStorage`**, readable by script on the page. The trade-off is
  written up in `docs/system-design.md`; a refresh flow with an `httpOnly` cookie is the
  production-grade alternative.
- **Oversized request bodies return 400, not 413.**
- **Re-running `make seed` does not reset vehicle ages** (existing rows are left as they are).
- **The aging boundary follows the database session's time zone** (`CURRENT_DATE`).
- **Prices are shown in USD** because the API does not state a currency.
- **The API reference exists only in the containers**, not under `make web` on the host, and has no dark theme. It is public (it shows only the contract). The Swagger UI page itself is checked by hand, not by an automated test.
- **No automated browser test suite.** End-to-end behaviour is covered by API-level tests, a
  container smoke test and component tests, plus the manual screenshot check above. There is no
  automated cross-browser run and no accessibility audit with a tool such as axe; accessibility
  rests on semantic HTML, `jsx-a11y` lint rules and tests that use roles and labels.
- **No load or performance testing.** The queries are indexed for the filters they serve
  (documented in `docs/database-design.md`), but only on 150 demo vehicles.
- **Single-instance deployment only.** The containers are for running and reviewing the app, not
  a production deployment: no TLS termination, secret management or backups.
