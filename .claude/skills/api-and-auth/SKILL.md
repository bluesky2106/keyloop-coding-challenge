---
name: api-and-auth
description: REST API, authentication and multi-tenant security rules. Use when adding endpoints, auth, middleware, or anything touching dealership data access.
---

# API and auth rules

- `docs/api/openapi.yaml` is the contract. Change it first, then code. Router tests in `internal/api` validate every response against the spec (`contract_test.go`); a new endpoint must be exercised there, not only in unit tests.
- REST, JSON, versioned under `/api/v1`. Plural nouns; correct status codes (201 create, 204 no body, 400 validation, 401 unauthenticated, 403 forbidden, 404 not found).
- Error body is always `{"error": {"code": "...", "message": "..."}}`; `validation_error` may also include `"fields": {"<field>": "<message>"}`. Matches `Error` in `docs/api/openapi.yaml`.
- Passwords: `bcrypt`, never logged or returned. Login returns a short-lived JWT (HS256, secret from env, `exp` set).
- JWT claims: `sub` is the user id and `did` is the dealership id (plus `iat` and a required `exp`). Middleware validates the token and puts the claims in the request context.
- **Tenant isolation:** `dealership_id` always comes from the token, never from the request path/body/query. Every repository method takes it as a parameter. Accessing another dealership's vehicle returns 404.
- Validate all input at the handler boundary (types, enums, lengths, pagination limits).
- List endpoints are paginated with a capped page size.
- Add tests for: missing token, bad token, cross-dealership access, invalid payload.
