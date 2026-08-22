# PRD.md — COLABORA Backend Product Requirements

This is the product spec for building the real backend behind **COLABORA**, PLN's workflow-tracking tool for **Permohonan PB/PD** (new electricity connection / power-change requests). The canonical UI/UX and domain spec already exists as a static hifi mockup at [`hifi-colabora/`](./hifi-colabora/) — see [`hifi-colabora/DEVELOPMENT.md`](./hifi-colabora/DEVELOPMENT.md) for the full source diagrams, the 17-activity SLA table, and the screen inventory. **Treat that file as the source of truth for exact SLA day-offsets and screen-to-role mapping; don't let it drift out of sync with this PRD.** This document translates that spec into what the Go backend (`colabora-be`, this repo) needs to implement, replacing the mockup's `localStorage`-simulated state and RBAC with real persistence and server-enforced authorization.

Companion docs (read alongside this one):
- [`DATA_MODEL.md`](./DATA_MODEL.md) — entities, fields, and the JSONB-vs-typed-column tradeoff for the 13 process forms.
- [`API_SPEC.md`](./API_SPEC.md) — REST endpoints, one per screen/form.
- [`RBAC.md`](./RBAC.md) — role/permission model, ported from `hifi-colabora/assets/rbac.js`.
- [`ROADMAP.md`](./ROADMAP.md) — build order, phased against this repo's module-generator workflow.

## 1. What we're building

A single aggregate — **Permohonan** — moves through **7 UI-level stages** grouping **17 numbered swimlane activities**, starting at a local unit (ULP) and ending at the area office (UP3), touching internal PLN roles and external vendors along the way. Every activity has an SLA deadline (calendar days from the request date, varying by connection type) and requires the responsible role to upload evidence before the process can advance. Today this handoff is untracked across disconnected teams; COLABORA gives every role one place to act on their step and management one place to see bottlenecks and SLA breaches.

This backend must:
1. Persist permohonan, their 17-activity progress, uploaded evidence, and an audit trail — replacing the mockup's per-page hardcoded state.
2. Enforce **stage ownership** server-side (a role can only submit the form for the activity/activities it owns) — replacing `assets/rbac.js`'s client-only `colaboraOwnsStage()`/`colaboraApplyFormGuard()`, which anyone can bypass via devtools.
3. Compute SLA deadlines and status (not started / in progress / done / overdue) from real dates, not hardcoded per detail page.
4. Serve the dashboard list with the same filters the mockup already validates (ULP unit, jenis sambungan, stage, status, SLA urgency, "Tugas Saya" vs "Semua Permohonan").
5. Support the two decision branches and the mandatory NPS approval step (§3 below) as first-class state, not free text.

Out of scope for the backend MVP (mirrors `hifi-colabora/DEVELOPMENT.md` §9): actual PDF/image storage can start as local disk / any S3-compatible bucket behind `pkg/utils`'s existing file helpers — pick one, don't build a custom DAM. Real-time push notifications, SLA-breach alerting/paging, and a mobile app are not implied by the mockup and are not in scope unless separately requested.

## 2. Actors & roles

Same functional roles (`fn`) as the mockup's `COLABORA_ROLES`. See [`RBAC.md`](./RBAC.md) for the full mapping to backend authorization; summarized here:

| Lane | Role (`fn`) | Owns (stage) |
|---|---|---|
| ULP | `pelayanan-pelanggan` | Stage 1 (open), Stage 7 (closing) |
| ULP | `teknik` | Stage 2 (survei), Stage 3 for JTR/JTM (RAB/KKO/KKF), Stage 6 for JTR/JTM (energize) |
| UP3 | `perencanaan` | Stage 3 for PLG TM (RAB/KKO/KKF), Stage 4 (WO Vendor Tiang) |
| UP3 | `konstruksi` | Stage 4 (WO Vendor Konstruksi, PK Vendor, WO PDKB decision) |
| UP3 | `transaksi-energi` | Stage 4 (WO Vendor APP), reservasi material & tera |
| UP3 | `jaringan` | Stage 6 for PLG TM (energize) |
| UP3 | `nps` | Stage 3 (Permohonan Perluasan + Persetujuan — **mandatory for every permohonan**) |
| UP3 | `pdkb` | Stage 5, only when Kebutuhan PDKB = Ya |
| Vendor | `vendor-tiang` | Stage 5 (pemasangan tiang) |
| Vendor | `vendor-konstruksi` | Stage 5 (konstruksi) & Stage 6 for PLG TM (SR/APP) |
| Vendor | `vendor-sr-app` | Stage 6 for JTR/JTM (SR/APP) |
| — | `super-user` | none — read-only cross-unit monitoring (`canViewAll`) |

ULP roles are further scoped to one of 3 units (Taman, Karang Pilang, Menganti); a permohonan belongs to exactly one ULP unit and that scoping must be enforced too (a `teknik` at Taman shouldn't act on a Karang Pilang permohonan) — the mockup doesn't model this restriction explicitly since demo data always matches the logged-in unit, but a real multi-tenant backend should.

## 3. Process flow — 17 activities, 7 stages

Full SLA table (day-offsets per connection type) lives in `hifi-colabora/DEVELOPMENT.md` §3 — reproduce it into `DATA_MODEL.md`'s SLA rule seed data, don't hand-copy it a third time. The stage grouping and decision points that affect backend state machine design:

| Stage | Activities | Gate to advance |
|---|---|---|
| 1 | #1 Permohonan PB/PD | Eviden permohonan uploaded |
| 2 | #2 Survei Perluasan Jaringan | Eviden survei uploaded |
| 3 | #3/#3b RAB/KKO/KKF (+ Kebutuhan Tiang? decision) **and** #4/#5 Permohonan Perluasan + Persetujuan NPS (mandatory) | Both RAB/KKO/KKF evidence **and** NPS `Disetujui` decision present |
| 4 | #6 WO Vendor Tiang (conditional on Kebutuhan Tiang), #7 WO Vendor Konstruksi (+ Perlu PDKB? decision), #8 WO Vendor APP | All applicable WOs issued |
| 5 | #11 Pemasangan Tiang, #12 Pelaksanaan Konstruksi, PDKB docs (conditional on Perlu PDKB) | Construction evidence uploaded (+ PDKB docs if required) |
| 6 | #13 Pengoperasian Jaringan Listrik, #14 Pemasangan SR/APP + Penyalaan | Both energize and SR/APP evidence uploaded |
| 7 | #15 Entri/Mutasi PDL, #16 Arsip AIL, #17 Selesai | Closing archive uploaded → permohonan marked `selesai` |

**Decision branches the state machine must model explicitly (not free text):**
- **Kebutuhan Tiang? (Ya/Tidak)**, decided inside the RAB/KKO/KKF step (`forms/04-rab-kko-kkf.html`) by `teknik` (JTR/JTM) or `perencanaan` (PLG TM). Gates whether Activity #6 (WO Vendor Tiang) is required before Stage 4 can complete.
- **Permohonan Perluasan → Persetujuan NPS (Disetujui/Ditolak)**, mandatory for every permohonan, decided by `nps` in `forms/03-permohonan-perluasan.html`. `Ditolak` should stop the process (exact rejection handling — hold vs. hard-close — needs a product decision before building; the mockup doesn't demo a rejected permohonan).
- **Perlu PDKB? (Ya/Tidak)**, decided by `konstruksi` right after issuing WO Vendor Konstruksi (`forms/05-wo-vendor.html`). Gates whether WO PDKB (`forms/07b-wo-pdkb.html`) and PDKB documentation (`forms/08b-pdkb-dokumentasi.html`) are required in Stages 4–5.
- **Jenis Sambungan** (JTR / JTM-Gardu / PLG TM <5 GWNG / PLG TM >5 GWNG), set once at Activity #1, determines which SLA column applies for every later activity and which role owns Stage 3/Stage 6 (`teknik` vs `perencanaan`/`jaringan`).

## 4. Status & SLA model

Per-activity status: `not_started` (upstream incomplete) → `in_progress` (owning role can act) → `done` (evidence uploaded, advanced) → `overdue` (past `request_date + H_offset(activity, jenis_sambungan)`, not done). SLA deadlines must be **computed from real dates**, not hardcoded per row — see `DATA_MODEL.md`'s `SLARule` table for how the 17×4 offset table becomes seed data instead of per-page copies.

## 5. Screens → backend surface

Each `forms/*.html` in the mockup becomes one backend write endpoint; `dashboard.html` and `details/detail-*.html` become read endpoints. Full mapping in [`API_SPEC.md`](./API_SPEC.md). One important mockup limitation to fix, not carry over: forms in the mockup are shared/generic and always prefilled with one sample permohonan's data — the real forms must be scoped to the specific permohonan the user is acting on.

## 6. Non-functional notes

- Every write endpoint needs both **authentication** (existing JWT middleware) and **stage-ownership authorization** (new — see `RBAC.md`) — a valid token alone isn't enough, the token's role must own the activity being submitted.
- Every stage transition should append to an **activity log / audit trail** (mirrors the "Log Aktivitas" section on each detail page) — this is what makes the SLA/bottleneck reporting trustworthy later; don't skip it to save a migration.
- Evidence uploads are the one hard requirement gating every stage transition (`upload before advance`) — validate presence of the required file(s) server-side per activity, don't rely on the client.
