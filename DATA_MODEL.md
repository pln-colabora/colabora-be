# DATA_MODEL.md — COLABORA Entities

Proposed GORM entities for `database/entities/`, following this repo's existing conventions (`Timestamp` embed from `entities/common.go`, `uuid.UUID` primary keys with `uuid-ossp` default, snake_case columns via GORM tags — see `entities/user_entity.go`). This is a design proposal, not yet implemented — validate field names/sizes against real PLN forms before migrating to production.

## Design decision: typed columns vs. JSONB payload

The 13 process forms (`forms/*.html`) have mostly disjoint fields (survey GPS coordinates vs. RAB budget numbers vs. WO vendor names). Two options:

- **One child table per form** — strongest typing/queryability, but 13 extra migrations and 13 extra repositories for fields almost nothing else reads.
- **One `permohonan_activities` row per activity with a JSONB `payload` column** — one migration, one repository, form-specific fields validated at the `dto`/`validation` layer instead of the schema layer.

**Recommendation: JSONB payload, with the fields that drive branching or filtering promoted to typed columns on `Permohonan` itself** (because those are read/filtered outside the form that produced them — the dashboard, the state machine, and other forms all need them):

- `jenis_sambungan` (Activity #1 → read by SLA calc and stage-3/6 ownership for every later activity)
- `kebutuhan_tiang` (Activity #3b decision → gates whether Activity #6 is required)
- `nps_keputusan` (Activity #5 decision → gates whether the permohonan proceeds at all)
- `perlu_pdkb` (WO Vendor Konstruksi decision → gates Stage 4/5 PDKB requirements)

Everything else (RAB numbers, vendor names, GPS coordinates, DC test results, etc.) goes in `permohonan_activities.payload jsonb`, shaped per activity by the `dto` package (one request struct per form, matching `create_module.sh`'s DTO convention).

## Entities

### `Permohonan`
The aggregate root — one row per PBPD request.

| Field | Type | Notes |
|---|---|---|
| `ID` | `uuid.UUID` | PK |
| `NoPermohonan` | `varchar(30)` unique | e.g. `PBPD-2026-0142` — format `PBPD-{year}-{seq}` |
| `JenisPermohonan` | `varchar(30)` | `Pasang Baru (PB)` \| `Perubahan Daya (PD)` |
| `JenisSambungan` | `varchar(30)` | `JTR` \| `JTM/Gardu` \| `PLG TM <5 GWNG` \| `PLG TM >5 GWNG` — set at Activity #1, drives SLA + Stage 3/6 ownership |
| `UlpUnit` | `varchar(50)` | `ULP Taman` \| `ULP Karang Pilang` \| `ULP Menganti` — scopes ownership within `teknik`/`pelayanan-pelanggan` |
| `PelangganNama`, `PelangganAlamat`, `PelangganNoHp` | `varchar` | customer info captured at Activity #1 |
| `RequestDate` | `date` | the "H"/"G" reference date every SLA offset is computed from |
| `CurrentStage` | `smallint` | 1–7, the UI-level stage (not activity number) |
| `Status` | `varchar(20)` | `in_progress` \| `selesai` \| (see PRD.md §3 for the NPS-rejection open question — don't invent a `ditolak` terminal status without a product decision) |
| `KebutuhanTiang` | `nullable bool` | Activity #3b decision; null until decided |
| `NpsKeputusan` | `nullable varchar(20)` | `Disetujui` \| `Ditolak`; null until Activity #5 |
| `PerluPdkb` | `nullable bool` | decided alongside WO Vendor Konstruksi |
| `OwnerFnOverride` | `varchar(200) nullable` | comma-separated `fn` list narrowing a co-owned stage to one role for *this* permohonan — ports `data-owner-fn` from the mockup's dashboard rows (see `RBAC.md`) |
| `CreatedBy` | `uuid.UUID` (FK → User) | who opened it (always `pelayanan-pelanggan`) |
| `Timestamp` | embedded | `created_at`/`updated_at` |

### `PermohonanActivity`
One row per (permohonan, activity_number) — 17 rows per permohonan once fully seeded, created lazily as each stage is reached (don't pre-create all 17 at Activity #1; conditional activities like #6/WO PDKB/#14 may never exist for a given permohonan).

| Field | Type | Notes |
|---|---|---|
| `ID` | `uuid.UUID` | PK |
| `PermohonanID` | `uuid.UUID` FK | |
| `ActivityNumber` | `smallint` | 1–17, per `PRD.md` §3 table |
| `StageNumber` | `smallint` | 1–7, derived from `ActivityNumber` (keep the mapping as a Go constant map, not duplicated per row — mirrors `assets/rbac.js`'s stage/activity split) |
| `Status` | `varchar(20)` | `not_started` \| `in_progress` \| `done` \| `overdue` |
| `SlaDeadline` | `date` | computed at creation from `Permohonan.RequestDate` + `SLARule` offset for this activity + `Permohonan.JenisSambungan` |
| `Payload` | `jsonb` | form-specific fields, one shape per `ActivityNumber` (validated by that form's `dto`) |
| `CompletedBy` | `uuid.UUID` FK nullable | user who submitted the form |
| `CompletedAt` | `timestamp` nullable | |
| `Timestamp` | embedded | |

Unique constraint on `(permohonan_id, activity_number)`.

### `SLARule`
Seed/config table replacing the mockup's per-detail-page hardcoded SLA copies (`hifi-colabora/DEVELOPMENT.md` §9 flags this as a documented mockup limitation — the real backend should fix it, not carry it over). One row per (activity, connection type) from the 17×4 table in `PRD.md` §3.

| Field | Type | Notes |
|---|---|---|
| `ActivityNumber` | `smallint` | 1–17 |
| `JenisSambungan` | `varchar(30)` | JTR / JTM-Gardu / PLG TM <5 GWNG / PLG TM >5 GWNG |
| `OffsetDays` | `smallint` | the `H+n`/`G+n` value |
| `ReferencePoint` | `varchar(1)` | `H` or `G` per the source table |

Seed via `database/seeders/` following the existing `user_seed.go` pattern (JSON fixture in `seeders/json/`, loader in `seeders/seeds/`).

### `Document`
Evidence uploads — every activity requires at least one before it can be marked `done`.

| Field | Type | Notes |
|---|---|---|
| `ID` | `uuid.UUID` | PK |
| `PermohonanID` | `uuid.UUID` FK | |
| `ActivityNumber` | `smallint` | which activity this evidence belongs to |
| `FileName`, `FileUrl`, `MimeType`, `SizeBytes` | | actual storage: local disk or S3-compatible bucket behind `pkg/utils` — pick one before building, don't invent a third option mid-implementation |
| `UploadedBy` | `uuid.UUID` FK | |
| `Timestamp` | embedded | |

### `ActivityLog`
Audit trail — mirrors each detail page's "Log Aktivitas" panel; also the basis for any future SLA/bottleneck reporting.

| Field | Type | Notes |
|---|---|---|
| `ID` | `uuid.UUID` | PK |
| `PermohonanID` | `uuid.UUID` FK | |
| `ActivityNumber` | `smallint nullable` | null for permohonan-level events (e.g. creation) |
| `Actor` | `uuid.UUID` FK → User | |
| `Action` | `varchar(100)` | e.g. `activity_submitted`, `nps_approved`, `nps_rejected`, `permohonan_created` |
| `Detail` | `text nullable` | free-form note |
| `Timestamp` | embedded | `created_at` is the event time |

### `User` (extend existing entity)
`entities/user_entity.go` already has a free-form `Role varchar(50)` column — reuse it directly for the `fn` values (`teknik`, `perencanaan`, `nps`, `vendor-tiang`, etc., see `RBAC.md`) instead of adding a parallel field. Two additions needed:

| Field | Type | Notes |
|---|---|---|
| `Unit` | `varchar(50) nullable` | `ULP Taman` \| `ULP Karang Pilang` \| `ULP Menganti` \| `UP3` \| vendor company name — scopes ULP roles to their unit |
| `CanViewAll` | `bool default false` | the `super-user` flag (`COLABORA_ROLES`' `canViewAll`) — read-only cross-unit monitoring, never a write permission |

Add both via `make migrate-create name=add_unit_and_can_view_all_to_users` rather than editing the existing `20240101000000_create_users_table.go` migration in place.

## Migration order

1. Extend `users` (Unit, CanViewAll).
2. `sla_rules` (seed data, no FKs — build this first so `permohonan_activities` can reference it at insert time).
3. `permohonan` (references `users.id` via `CreatedBy`).
4. `permohonan_activities` (references `permohonan.id`, `users.id`).
5. `documents` (references `permohonan.id`).
6. `activity_logs` (references `permohonan.id`, `users.id`).

Each should be its own `make migrate-create name=create_x_table` call so `database/migration.go`'s `AutoMigrate` list and `database/entities/` stay in lockstep, per this repo's existing convention (see root `CLAUDE.md`).
