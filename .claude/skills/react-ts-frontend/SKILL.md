---
name: react-ts-frontend
description: Conventions for the React + TypeScript frontend in frontend/. Use when building components, API calls, state, filters, or frontend tests.
---

# React + TypeScript conventions

- Vite + React + TypeScript with `strict: true`. No `any`; use `unknown` and narrow.
- API types are generated from `docs/api/openapi.yaml` (`npm run generate:api` writes `src/api/schema.d.ts`; `npm run check:api` fails when it is stale and runs in `make lint`). Never hand-write response types; alias them in `src/api/types.ts`. Use the `openapi-fetch` client in `src/api/client.ts` and wrap calls in the hooks in `src/api/hooks.ts`.
- Server state: TanStack Query. Local UI state: `useState`. No global store unless a real need appears.
- Inventory filters live in the URL query string so views are shareable and survive refresh.
- Auth: the token lives in `sessionStorage` through `src/api/tokenStore.ts` (trade-off in `docs/system-design.md` section 5) and is sent as `Authorization: Bearer`. A 401 clears it, which signs the user out and drops the query cache. Do not retry 4xx responses (`src/api/retry.ts`).
- Aging stock is visually prominent (badge + sort/filter) and never conveyed by colour alone.
- Accessibility: semantic elements, labelled form controls, keyboard-operable tables and dialogs; `eslint-plugin-jsx-a11y` must pass.
- Components stay small and colocated with their feature folder; no premature shared "ui kit".

## Testing
- Vitest + React Testing Library + MSW (`src/test/server.ts` mocks the API at the network level; unhandled requests fail the test). Query by role/label, not test ids or class names. Wait for requests with `waitFor`, never assume they have been sent. Coverage thresholds in `vite.config.ts` fail `npm test`; do not lower them.
- Labels must not wrap hints, counters or errors: keep those outside the `<label>` and link them with `aria-describedby`.
- A form with several text fields and no submit button is not submitted by Enter; give it one.
- Test behaviour: filtering, aging flag shown, logging an action persists and appears in history.

## Verify before done
`make lint test` for the frontend must pass (`eslint`, `prettier --check`, `tsc --noEmit`, `vitest run`).
