---
name: react-ts-frontend
description: Conventions for the React + TypeScript frontend in frontend/. Use when building components, API calls, state, filters, or frontend tests.
---

# React + TypeScript conventions

- Vite + React + TypeScript with `strict: true`. No `any`; use `unknown` and narrow.
- API types and client are generated from `docs/api/openapi.yaml`; never hand-write response types.
- Server state: TanStack Query. Local UI state: `useState`. No global store unless a real need appears.
- Inventory filters live in the URL query string so views are shareable and survive refresh.
- Auth: keep the JWT in memory plus a refresh-safe strategy documented in the design doc; send via `Authorization: Bearer`. Redirect to login on 401.
- Aging stock is visually prominent (badge + sort/filter) and never conveyed by colour alone.
- Accessibility: semantic elements, labelled form controls, keyboard-operable tables and dialogs; `eslint-plugin-jsx-a11y` must pass.
- Components stay small and colocated with their feature folder; no premature shared "ui kit".

## Testing
- Vitest + React Testing Library; query by role/label, not test ids or class names.
- Test behaviour: filtering, aging flag shown, logging an action persists and appears in history.

## Verify before done
`make lint test` for the frontend must pass (`eslint`, `prettier --check`, `tsc --noEmit`, `vitest run`).
