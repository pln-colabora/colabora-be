# Phase 5 — documents (evidence uploads)

> Historical implementation note: this filename reflects the delivery context in which the Garage document module was built. Current phase status and future delivery gates are governed by [`DEVELOPMENT.md`](./DEVELOPMENT.md).

## Context

Every workflow completion requires the responsible role to upload its specified evidence before the permohonan can advance (`PRD.md` §5; `DATA_MODEL.md`'s `Document` entity; `API_SPEC.md`'s "Documents" section).

Two things ROADMAP.md assumes aren't true, discovered during investigation:
- **Phase 4 (the 13+ activity-submission endpoints) doesn't exist in code at all** — only Phase 1–3 are built (`permohonan` create/list/detail, RBAC middleware unused by any route). ROADMAP.md wanted the document-module-vs-folded decision made "based on how Phase 4 shaped `permohonan`'s repository," which isn't available. **Decision: standalone `document` module**, matching `API_SPEC.md`'s already-written dedicated endpoints (`POST/GET /api/permohonan/:id/documents`, `GET .../documents/:doc_id`) and keeping the storage-client concern out of `permohonan`'s service.
- **There was no working upload plumbing to build on.** `modules/user`'s `Image *multipart.FileHeader` field, the `ImageUrl` column, and `pkg/utils/file.go`'s `UploadFile()` (local-disk only) were all dead scaffolding from the starter template — never wired to anything. This phase introduces the entire pipeline from scratch; nothing here is repurposed from that code.

**Storage backend: Garage** — the app runs on a 2GB/2vCPU/40GB VPS, and Garage's ~512MB RAM footprint fits that better than SeaweedFS's ~1GB. Trade-off accepted knowingly: Garage's own docs advise against replication-factor-1 in production (no redundancy) — but on a single VPS, SeaweedFS would be single-node too, and this stack already has a single Postgres instance with no documented replication story, so this doesn't change the app's actual risk posture, it's consistent with it. Mitigate via regular backups of Garage's data directory, same as you'd back up Postgres. Both Garage and SeaweedFS speak the S3 API, so the Go client code isn't Garage-specific — swapping backends later is a config change (`GARAGE_S3_ENDPOINT`/`GARAGE_REGION`), not a rewrite.

**Security-motivated deviation from `DATA_MODEL.md`'s literal field name**: `Document.FileUrl` stores the storage **object key**, not a public URL — the bucket stays private, and `GET .../documents/:doc_id` generates a short-lived (15 min) presigned URL per request. Evidence documents (customer PII, addresses, phone numbers) shouldn't be reachable via a stored static link.

## What was built

- `database/entities/document_entity.go` + `database/migrations/20260824010000_create_documents_table.go` — the `Document` entity/table.
- `modules/document/` — standalone module: `dto`, `validation` (MIME allow-list: pdf/jpg/jpeg/png, 10MB max), `repository`, `service`, `controller`, `routes.go`, plus `storage/` (a small `Client` interface + an S3-backed implementation against `github.com/aws/aws-sdk-go-v2/service/s3`, so the service is unit-testable without a live Garage instance).
- `config/storage.go` (`SetUpStorageClient`) + `pkg/constants.Storage` — wired into `providers/core.go` the same way `constants.DB` is.
- `docker/garage/garage.toml`, a `garage` service in `docker-compose.yml` (S3 port published to host, for on-host `make run`) and `docker-compose.prod.yml` (internal `app-network` only), and matching `GARAGE_*` vars in `.env.example`/`.env`.
- `make container-garage` — shells into the container for the one-time bootstrap below (note: the Garage image has no shell — `docker exec` the `/garage` binary directly per the bootstrap commands, don't try to `sh` in first).

### Upload-first / attach-later design

The document lifecycle is two-phase, not "upload = attach":

1. `POST /api/documents` (top-level, no permohonan in the path) — uploads a raw file standalone. `Document.PermohonanID`/`ActivityNumber` are **nullable** and left `NULL` here. No RBAC check happens at this step — there's no permohonan/activity context yet to check ownership against.
2. `DocumentService.AttachToActivity(ctx, userId, permohonanId, activityNumber, documentIds []string)` — the currently implemented method links uploaded documents to one permohonan/activity and authorizes through the legacy stage-derived `rbac.OwnsActivity(...)`. The `permohonan_id IS NULL` guard prevents attaching an already-owned document.

The workflow refactor replaces this integration contract with workflow-node attachment and `OwnsWorkflowNode(...)`. `activity_number` remains optional reporting/SLA metadata; it is not sufficient authorization identity for 3b, PK Vendor, or PDKB supporting nodes. See `DATA_MODEL.md` and `API_SPEC.md` for the target shape.

**Not wired to any HTTP endpoint yet** — `AttachToActivity` is built ahead of its caller, same pattern as `HasEvidence`, for Phase 4's activity-submission endpoints (`document_ids: []` in their request body) to call once they exist.

We considered a genuinely generic `documents(id, type, file_path)` table + `document_permohonan` pivot for this (polymorphic attachment) and rejected it after a dedicated stress-test: a document is still only ever attached to **one** permohonan+activity, ever — just possibly later than upload rather than at upload. That's 1:N, not N:M. Nullable columns directly on `documents` give the identical two-phase UX (upload → get an id → reference it later) without a join table, without an extra transaction per attach, and without needing a uniqueness constraint to fake 1:1 behavior on a structurally N:M table.

Read routes, nested under the permohonan resource, all behind `middlewares.Authenticate`:
- `GET /api/permohonan/:id/documents` — list, optional `?activity_number=` filter. Only ever returns attached documents (nullable columns naturally exclude unattached rows).
- `GET /api/permohonan/:id/documents/:doc_id` — 302 redirect to a 15-minute presigned URL. 404s if the document doesn't exist, isn't attached to any permohonan yet, or is attached to a *different* permohonan than the one in the URL.

`Document` intentionally has no `FileName`/`MimeType`/`SizeBytes` columns — MIME/size validation runs against the incoming `multipart.FileHeader` at upload time without persisting it; the original filename is embedded in the storage key (`documents/{uuid}/{filename}`) for readability; file size can be fetched from storage on demand if ever needed.

## Garage bootstrap (one-time, after first `docker compose up -d`)

Not idempotent — run once per fresh `garage-meta`/`garage-data` volume, not on every deploy. Verified against Garage's own docs (quick-start + real-world deployment cookbook):

```sh
make up   # or: docker compose up -d
make container-garage
# now inside the garage container:
/garage node id                                   # copy the printed node ID
/garage layout assign -z dc1 -c 1G <node-id>
/garage layout apply --version 1
/garage key create colabora-app-key               # copy the printed Key ID + Secret Key
/garage bucket create colabora-documents           # matches GARAGE_BUCKET in .env
/garage bucket allow --read --write --key colabora-app-key colabora-documents
```

Paste the printed Key ID / Secret Key into `.env`'s `GARAGE_ACCESS_KEY` / `GARAGE_SECRET_KEY`, then `make run` (or restart the `app` container in prod).

## Verified against source docs

`dxflrs/garage:v2.3.0` image, `garage.toml` shape (`metadata_dir`/`data_dir`/`db_engine`/`replication_factor`/`rpc_bind_addr`/`rpc_public_addr`/`[s3_api]`), default ports (3900 S3 API, 3901 RPC, 3903 admin), and the bootstrap command sequence were cross-checked against two independent fetches of Garage's official quick-start and real-world-deployment docs. `GARAGE_RPC_SECRET` as an env var (rather than hardcoded in `garage.toml`) is explicitly confirmed by Garage's configuration reference. One claim from an initial fetch — a `--single-node --default-bucket` auto-bootstrap flag — was **not** corroborated by the official configuration reference and was dropped in favor of the documented manual bootstrap above; don't rely on that flag existing.

Also done: `docs/document.yaml` (registered in `docs/routes.go` and `docs/openapi_merge.go`, same self-contained-per-module convention as `permohonan.yaml`), and tests — `modules/document/validation/document_validation_test.go` (table-driven, real logic) and `modules/document/service/document_service_test.go` (hand-rolled fake repos + fake `storage.Client`, same pattern `middlewares/authorization_test.go` established, since this repo has no mocking library).

## Not done in this phase (deliberately out of scope)

- Phase 4's activity-submission endpoints don't exist yet, so nothing currently calls `HasEvidence` (evidence-gate check) or `AttachToActivity` (upload→attach) — both are built and ready for whenever those endpoints land.
- The Garage bootstrap above hasn't been run against a real deployment as part of this change — `GARAGE_ACCESS_KEY`/`GARAGE_SECRET_KEY` in `.env` are still placeholders. Run it before relying on uploads working.

## Known limitation: document read access isn't role/unit-scoped

`DocumentService.List`/`Download` allow any authenticated user to view/download any permohonan's evidence files — no ownership or unit check. This is a deliberate decision to **match**, not fix, `permohonan_service.go`'s existing `GetById` (Phase 2, predates this phase), which already lets any authenticated user read a permohonan's full detail (including `PelangganNama`/`PelangganAlamat`/`PelangganNoHp`) with zero role/unit restriction.

Flagged by automated security review as an IDOR; consciously not patched here because:
- Fixing documents alone wouldn't close the actual exposure — the same customer PII is already readable via `GetById`.
- The target `RBAC.md` now separates read access from workflow-node write ownership, but the exact participant/view matrix still needs to be implemented consistently for permohonan and documents.
- The obvious-looking shortcuts are wrong: `UlpUnit` matching would incorrectly lock out non-ULP-scoped roles (`konstruksi`, `transaksi-energi`, `nps`, all `vendor-*` — see `ulpScopedRoles` in `pkg/rbac/stage_owners.go`), and `rbac.OwnsStage` would incorrectly lock out `super-user` (RBAC.md's read-only cross-unit monitoring role — `CanViewAll` isn't even implemented on `User` yet) and anyone reviewing a now-completed earlier stage.

**Follow-up**: define a real "view" authorization model (in `RBAC.md`) and apply it to `GetById` and the `document` endpoints together, not just one of them.
