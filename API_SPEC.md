# API_SPEC.md — COLABORA REST Endpoints

One endpoint per screen in `hifi-colabora/`, following this repo's existing module conventions (envelope via `pkg/utils.BuildResponseSuccess`/`BuildResponseFailed`, pagination via `github.com/Caknoooo/go-pagination`, auth via `middlewares.Authenticate`). Every write endpoint additionally needs the stage-ownership authorization described in [`RBAC.md`](./RBAC.md) — a valid JWT is necessary but not sufficient.

Entities referenced here are defined in [`DATA_MODEL.md`](./DATA_MODEL.md); process rules and decision branches in [`PRD.md`](./PRD.md) §3.

## Suggested module split

Given the module-per-domain convention (`make module name=<x>`), two new top-level modules cover this:

- **`permohonan`** — the aggregate: create, list/dashboard, detail, and all 13 activity-submission endpoints (they all mutate the same aggregate, so keep them in one module rather than 13 separate ones).
- **`document`** — evidence upload/download, shared across every activity endpoint above (or fold into `permohonan` if uploads never need to stand alone — decide when `permohonan`'s repository is scaffolded).

`user`/`auth` modules already exist and just need the `Unit`/`CanViewAll` fields from `DATA_MODEL.md`.

## Auth (existing, extend only)

Already implemented in `modules/auth`. No new endpoints — just make sure `Register`/seed data can set `Unit` and the `fn`-shaped `Role`.

## Dashboard & detail (read)

| Method | Path | Maps to | Auth |
|---|---|---|---|
| `GET` | `/api/permohonan` | `dashboard.html` list | any authenticated user; response scoped by `Unit`/`CanViewAll` — see RBAC.md |
| `GET` | `/api/permohonan/:id` | `details/detail-XXXX.html` | any authenticated user (view-only if not the stage owner) |
| `GET` | `/api/permohonan/:id/activities` | detail page's 7-stage timeline | same |
| `GET` | `/api/permohonan/:id/documents` | detail page's "Dokumen Terunggah" | same |
| `GET` | `/api/permohonan/:id/logs` | detail page's "Log Aktivitas" | same |

`GET /api/permohonan` query params (drives the same filters the mockup already validates): `ulp`, `jenis_sambungan`, `stage`, `status`, `sla` (`ontime`/`duesoon`/`overdue`), `search`, `scope=mine|all` (mine = rows the caller's `fn` owns, matching "Tugas Saya"), plus standard pagination params from `go-pagination`.

Response for `GET /api/permohonan/:id` should include a computed `can_act: bool` (server-side equivalent of `colaboraOwnsStage()`) so the frontend doesn't reimplement ownership logic — never trust a frontend-only check for this.

## Activity endpoints (write, one per form)

| Method | Path | Form | Activities | Required `fn` |
|---|---|---|---|---|
| `POST` | `/api/permohonan` | `forms/01-permohonan-pbpd.html` | #1 | `pelayanan-pelanggan` |
| `POST` | `/api/permohonan/:id/survei` | `forms/02-survei.html` | #2 | `teknik` |
| `POST` | `/api/permohonan/:id/rab-kko-kkf` | `forms/04-rab-kko-kkf.html` | #3, #3b | `teknik` (JTR/JTM) or `perencanaan` (PLG TM) |
| `POST` | `/api/permohonan/:id/permohonan-perluasan` | `forms/03-permohonan-perluasan.html` | #4, #5 | `nps` |
| `POST` | `/api/permohonan/:id/wo-vendor/tiang` | `forms/05-wo-vendor.html` (Tiang variant) | #6 | `perencanaan`, only if `KebutuhanTiang = true` |
| `POST` | `/api/permohonan/:id/wo-vendor/konstruksi` | `forms/05-wo-vendor.html` (Konstruksi variant) | #7 | `konstruksi`; body includes `perlu_pdkb` decision |
| `POST` | `/api/permohonan/:id/wo-vendor/app` | `forms/05-wo-vendor.html` (APP variant) | #8 | `transaksi-energi` |
| `POST` | `/api/permohonan/:id/reservasi-material` | `forms/06-reservasi-material.html` | #9, #10 | `transaksi-energi` |
| `POST` | `/api/permohonan/:id/pk-vendor` | `forms/07-pk-vendor-pelaksana.html` | (PK issuance) | `konstruksi` |
| `POST` | `/api/permohonan/:id/wo-pdkb` | `forms/07b-wo-pdkb.html` | (WO PDKB) | `konstruksi`, only if `PerluPdkb = true` |
| `POST` | `/api/permohonan/:id/pelaksanaan-konstruksi` | `forms/08-pelaksanaan-konstruksi.html` | #11, #12 | `vendor-tiang` or `vendor-konstruksi` |
| `POST` | `/api/permohonan/:id/pdkb-dokumentasi` | `forms/08b-pdkb-dokumentasi.html` | (PDKB docs) | `pdkb`, only if `PerluPdkb = true` |
| `POST` | `/api/permohonan/:id/energize-jaringan` | `forms/09-energize-jaringan.html` | #13 | `teknik` (JTR/JTM) or `jaringan` (PLG TM) |
| `POST` | `/api/permohonan/:id/pemasangan-sr-app` | `forms/12-pemasangan-sr-app.html` | #14 | `vendor-sr-app` (JTR/JTM) or `vendor-konstruksi` (PLG TM) |
| `POST` | `/api/permohonan/:id/closing` | `forms/11-closing.html` | #15, #16, #17 | `pelayanan-pelanggan`; marks `Permohonan.Status = selesai` |

Each of these:
1. Validates the caller's `fn` owns the target activity (`RBAC.md`), scoped further by `Permohonan.UlpUnit` for ULP roles and by `Permohonan.OwnerFnOverride` when a stage is co-owned.
2. Validates required evidence file(s) are attached (see `PRD.md` §6 — don't allow advancing without it).
3. Writes/updates the `PermohonanActivity` row (`payload` per that form's DTO), sets `Status = done`, `CompletedBy`, `CompletedAt`.
4. Recomputes `Permohonan.CurrentStage`/`Status` if this was the last activity gating the stage (see the "Gate to advance" column in `PRD.md` §3).
5. Appends an `ActivityLog` row.
6. Returns the updated permohonan/activity, using `pkg/utils.BuildResponseSuccess`.

Conditional endpoints (`wo-vendor/tiang`, `wo-pdkb`, `pdkb-dokumentasi`) should 409/400 with a clear message if called when the gating decision (`KebutuhanTiang`/`PerluPdkb`) is `false` or not yet set — don't silently no-op.

## Documents

| Method | Path | Notes |
|---|---|---|
| `POST` | `/api/permohonan/:id/documents` | multipart upload; `activity_number` in form data; called by the activity endpoints above, or standalone if a role needs to attach evidence separately from submitting the form |
| `GET` | `/api/permohonan/:id/documents/:doc_id` | download/redirect to storage URL |

## Reference/lookup endpoints (optional, for form dropdowns)

| Method | Path | Notes |
|---|---|---|
| `GET` | `/api/users?fn=vendor-tiang,vendor-konstruksi` | populate "Vendor Pelaksana" dropdowns in WO forms |
| `GET` | `/api/sla-rules` | expose the `SLARule` seed table if the frontend wants to show SLA figures without duplicating them (fixes the mockup's per-page hardcoded SLA copy problem, see `DATA_MODEL.md`) |

## Not building yet

Rejection handling for `NpsKeputusan = Ditolak` needs a product decision (see `PRD.md` §3) before an endpoint contract is finalized — don't guess at a `POST .../reject` shape without confirming what should happen to the permohonan afterward.
