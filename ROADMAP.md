# ROADMAP.md — Build Order

Phased implementation plan for turning `hifi-colabora/` into the real backend, sequenced against this repo's existing tooling (`make module`, `make migrate-create`, DI wiring in `providers/core.go`). See `PRD.md`/`DATA_MODEL.md`/`API_SPEC.md`/`RBAC.md` for the specs each phase implements.

## Phase 0 — extend auth/user

- Migration: add `Unit`, `CanViewAll` to `users` (`make migrate-create name=add_unit_and_can_view_all_to_users`).
- Seed the 12 demo accounts from `hifi-colabora/README.md`'s account table (or your real PLN account list) via `database/seeders/`.
- No new module needed — `modules/auth`/`modules/user` already exist.

## Phase 1 — SLA rules (seed-only, no dependents yet)

- Migration + entity for `SLARule` (`DATA_MODEL.md`).
- Seed from the 17×4 table in `PRD.md` §3 / `hifi-colabora/DEVELOPMENT.md` §3.
- No controller needed yet beyond an optional read-only `GET /api/sla-rules` for frontend reference.

## Phase 2 — core `permohonan` module

- `make module name=permohonan` — scaffolds controller/service/repository/dto/validation/query/tests.
- Migrations + entities: `Permohonan`, `PermohonanActivity`, `ActivityLog` (`DATA_MODEL.md`).
- Endpoints: `POST /api/permohonan` (Activity #1 only), `GET /api/permohonan` (dashboard list w/ filters), `GET /api/permohonan/:id` (detail).
- Wire into `providers/core.go` and `cmd/main.go` the same way `user`/`auth` already are.
- This phase alone should be enough to reproduce `dashboard.html` + a read-only `details/*.html` equivalent against real data.

## Phase 3 — RBAC middleware

- Implement `OwnsActivity()`/`RequireActivityOwner()` per `RBAC.md`.
- Apply to nothing yet (no write endpoints beyond Activity #1 exist at this point) — but land it before Phase 4 so every subsequent activity endpoint is gated from the start, not retrofitted.

## Phase 4 — activity endpoints, in swimlane order

Add one at a time, each as a new handler on the `permohonan` module (not a new top-level module — see `API_SPEC.md`), gated by `RequireActivityOwner`:

1. `/survei` (#2)
2. `/rab-kko-kkf` (#3, #3b decision)
3. `/permohonan-perluasan` (#4, #5 — mandatory NPS decision; **confirm the rejection-handling product decision from `PRD.md` §3 before building this one**, not after)
4. `/wo-vendor/tiang`, `/wo-vendor/konstruksi` (+ PDKB decision), `/wo-vendor/app` (#6, #7, #8)
5. `/reservasi-material` (#9, #10)
6. `/pk-vendor`, `/wo-pdkb` (conditional)
7. `/pelaksanaan-konstruksi` (#11, #12), `/pdkb-dokumentasi` (conditional)
8. `/energize-jaringan` (#13)
9. `/pemasangan-sr-app` (#14)
10. `/closing` (#15, #16, #17 — terminal, sets `Status = selesai`)

Each addition should include: the stage-completion recompute logic (`Permohonan.CurrentStage`/`Status`), an `ActivityLog` entry, and its own test file under `modules/permohonan/tests/`.

## Phase 5 — documents

- `make module name=document` (or fold into `permohonan` if by this point uploads never need to stand alone as their own resource — decide based on how Phase 4 actually shaped `permohonan`'s repository).
- Pick one storage backend (local disk vs. S3-compatible) before writing this phase — don't leave it configurable "just in case."
- Wire upload into each Phase 4 endpoint's required-evidence validation.

## Phase 6 — retire the mockup as a source of behavior

Once Phases 1–5 are done, `hifi-colabora/` should go back to being a pure design reference (its own `CLAUDE.md` already says as much) rather than something anyone still points a browser at for actual workflow tracking. Don't leave both a "real" backend and the `localStorage`-simulated mockup usable in parallel for the same users — that's the fastest way to get divergent SLA numbers and confused stakeholders.

## Explicitly not phased yet

- NPS-rejection flow (needs the product decision flagged in `PRD.md` §3 and `API_SPEC.md`).
- Real-time notifications / SLA-breach alerting.
- A non-mockup frontend to replace `hifi-colabora/` — out of scope for this backend repo.
