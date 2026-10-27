---
name: design-docs
description: Templates and rules for writing the System Design Document and database/API docs in docs/. Use when writing or updating anything under docs/.
---

# Design document rules

- Docs live in `docs/`: `system-design.md`, `database-design.md`, `validation.md`, `api/openapi.yaml`. Do not add more without need.
- Write for a reviewer who has 10 minutes: lead with the decision, then the reason.
- Diagrams use Mermaid in markdown (architecture flow, ERD).

## `system-design.md` outline
1. Goal and scope (in / out of scope)
2. Assumptions (numbered `A1, A2...`; the brief says ambiguity must be documented here)
3. Architecture (diagram + component responsibilities)
4. Key decisions table: decision | options considered | choice | trade-off
5. Security and multi-tenancy
6. Future evolution: configurable aging threshold (per-dealership setting), real-time push (SSE), caching, scaling, observability

## `database-design.md` outline
ERD, table-by-table columns and constraints, indexes with the query each serves, migration and seed approach.

## `validation.md`
Only real evidence: what was verified and how, what the AI got wrong and was corrected, known gaps. Never invent results.
