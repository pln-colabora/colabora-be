# RBAC.md — Backend Authorization Model

Ports `hifi-colabora/assets/rbac.js`'s client-side simulation into server-enforced authorization. The mockup's version is explicitly a demo shortcut (`localStorage`, bypassable via devtools — see `hifi-colabora/DEVELOPMENT.md` §6/§9); this document describes the real version.

## Roles (`fn`)

Same 12 functional roles as `COLABORA_ROLES` in the mockup. Store as the existing `User.Role varchar(50)` column (already free-form, no schema change needed — see `DATA_MODEL.md`):

`pelayanan-pelanggan`, `teknik`, `perencanaan`, `konstruksi`, `transaksi-energi`, `jaringan`, `nps`, `pdkb`, `vendor-tiang`, `vendor-konstruksi`, `vendor-sr-app`, `super-user`.

`super-user` is never a stage owner — it's the read-only cross-unit monitoring role (`CanViewAll = true` on `User`, see `DATA_MODEL.md`). Never grant it write access to any activity endpoint, no matter how "convenient" that seems for an admin/superuser shortcut.

## Stage ownership table

Direct port of `COLABORA_STAGE_OWNERS`. Keep this as a Go constant map (e.g. `pkg/constants/stage_owners.go` or a small `pkg/rbac` package) — **single source of truth**, don't scatter role checks across controllers.

| Stage | Owner(s) | Notes |
|---|---|---|
| 1 | `pelayanan-pelanggan` | opens the permohonan |
| 2 | `teknik` | survey |
| 3 | `teknik` \| `perencanaan` (RAB/KKO/KKF, split by `JenisSambungan`) **+** `nps` (Permohonan Perluasan/Persetujuan) | co-owned; narrow per-permohonan via `OwnerFnOverride` (see below) — `teknik`/`perencanaan`'s RAB work and `nps`'s approval are different tasks that happen to share a stage number |
| 4 | `perencanaan` (#6 Tiang) \| `konstruksi` (#7 Konstruksi + PDKB decision + PK Vendor + WO PDKB) \| `transaksi-energi` (#8 APP) | co-owned, one WO variant each |
| 5 | `vendor-tiang` (#11) \| `vendor-konstruksi` (#12) \| `pdkb` (optional docs) | co-owned; `konstruksi` has **no** ownership here — its part ended at Stage 4 |
| 6 | `teknik` \| `jaringan` (#13, split by `JenisSambungan`) \| `vendor-sr-app` \| `vendor-konstruksi` (#14, split by `JenisSambungan`) | co-owned; `konstruksi` has no ownership here either |
| 7 | `pelayanan-pelanggan` | closing |

## Per-permohonan owner override

A co-owned stage is shared at the *stage* level, but a given permohonan is normally blocked on exactly one role (e.g. Stage 3 is `teknik`+`perencanaan`+`nps`, but a specific permohonan waiting on NPS approval shouldn't show as actionable to `perencanaan`). Port `Permohonan.OwnerFnOverride` (comma-separated `fn` list, see `DATA_MODEL.md`) directly from the mockup's `CURRENT_STAGE_OWNERS`/`data-owner-fn` pattern: when set, authorization checks against this list instead of the full stage-wide owner list; when empty, fall back to the stage-wide list.

## `JenisSambungan`-conditional ownership

Stage 3 and Stage 6 split ownership by connection type, not just by stage:
- Stage 3 RAB/KKO/KKF: `teknik` owns it for `JTR`/`JTM-Gardu`, `perencanaan` owns it for `PLG TM <5 GWNG`/`PLG TM >5 GWNG`.
- Stage 6 energize (#13): `teknik` for JTR/JTM, `jaringan` for PLG TM.
- Stage 6 SR/APP (#14): `vendor-sr-app` for JTR/JTM, `vendor-konstruksi` for PLG TM.

Any authorization check for these activities must read `Permohonan.JenisSambungan`, not just the activity number.

## Unit scoping (new — not in the mockup)

The mockup's demo data always has the logged-in ULP role matching the permohonan's ULP unit, so it never needed to check this. A real multi-tenant backend should also verify `User.Unit == Permohonan.UlpUnit` for ULP roles (`pelayanan-pelanggan`, `teknik`) before granting write access — otherwise a `teknik` at ULP Taman could act on a Karang Pilang permohonan just by having the right `fn`. See `PRD.md` §2.

## Implementation shape

Recommend a middleware/helper pair mirroring the mockup's two functions:

- `colaboraOwnsStage(session, stage, ownerOverride)` → a Go function `OwnsActivity(user, permohonan, activityNumber) bool` that: looks up the activity's stage from the constant map, resolves the effective owner list (override if set, else stage-wide, filtered by `JenisSambungan` where applicable), checks `user.Role` is in it, and — for ULP roles — checks `user.Unit == permohonan.UlpUnit`.
- `colaboraApplyFormGuard([...])` → a Gin middleware, e.g. `middlewares.RequireActivityOwner(activityNumber)`, applied per-route in each module's `routes.go` the same way `middlewares.Authenticate(jwtService)` already is — reject with 403 (not a silently-disabled button) before the controller runs, since this is a real API, not a client-rendered form.

Don't reimplement the ownership check inline in each controller — one shared function/middleware, same as the mockup's single `rbac.js`.
