# COLABORA Development Plan

This is the canonical implementation plan for the COLABORA backend. It records what is implemented, what remains, and the exit gate for each delivery phase. Detailed product behavior, persistence, API, and authorization contracts remain in `PRD.md`, `DATA_MODEL.md`, `API_SPEC.md`, and `RBAC.md`.

**Baseline verified:** 8 September 2026  
**Next phase:** Phase 4 — Activity endpoints

**Compatibility policy:** breaking development schema and API changes are allowed; runtime OpenAPI must continue to describe only behavior that is actually implemented.

## 1. Sources of truth

Use these sources in this order:

1. `hifi-colabora/workflow/jtr-jtm.html` and `hifi-colabora/workflow/plg-tm.html` define workflow order, prerequisites, branches, joins, and ownership. Re-read both before every workflow change and compare the nested checkout with the root gitlink.
2. `PRD.md`, `DATA_MODEL.md`, `API_SPEC.md`, and `RBAC.md` translate the diagrams into the target backend contract.
3. This document governs development status and delivery order.
4. `docs/*.yaml` and the Go code describe the current runtime. They must not advertise a planned contract early.
5. `DOCUMENT_WORKFLOW_REFERENCE.md` summarizes sensitive operational samples for payload and evidence design. Those samples are evidence references, not workflow authority.

The remaining hifi forms, detail pages, demo navigation, and client-side RBAC are illustrative. They may lag the diagrams and may be retired when production forms are defined.

## 2. Workflow invariants

- Activity numbers are display and SLA labels, not a state machine. Stable workflow-node codes are the persistence and authorization identity.
- Presentation stages are derived groupings. They must not block an independent branch or determine authorization.
- JTR/JTM requests start with `pelayanan-pelanggan` at the caller's ULP; PLG TM requests start with `nps` and require an explicitly selected target ULP.
- Activities #3 and #3b complete before NPS Activities #4–#5.
- NPS `delegated` unlocks the applicable WO branches; `returned` is terminal and remains auditable.
- Pole work and PDKB work are conditional. Construction, APP preparation, pole work, energization, and SR/APP work contain parallel branches and explicit join gates.
- Every transition, branch decision, skip, evidence attachment, and aggregate projection change is committed atomically with an activity log entry.
- Workflow status and SLA state are separate. A node without an SLA rule has no fabricated deadline.
- Write authorization is evaluated against the exact workflow node, connection type, role, unit scope, applicability, and current node state.

## 3. Current implementation baseline

### Implemented

- Gin/GORM application structure, PostgreSQL migrations and seeders, dependency injection, health endpoints, CORS, and Scalar documentation aggregation.
- JWT access/refresh authentication and users with COLABORA role and unit data.
- SLA-rule persistence and seed data for numbered activities.
- Connection-specific creation: matching-ULP `pelayanan-pelanggan` for JTR/JTM, and `nps` with a configured target ULP for PLG TM.
- Paginated list, node-based `scope=mine`, detail with all workflow nodes and caller-owned `available_actions`, plus activity-timeline and audit-log reads.
- Workflow-node persistence: creation initializes all 21 canonical nodes, with nullable display/SLA fields for decision and support nodes; logs carry the canonical node code.
- Exact-node role/unit ownership helpers, plus a compatibility middleware for remaining numbered routes.
- Private Garage/S3-compatible document upload, list, download, validation, and upload-first node-evidence helpers. One stored file may evidence multiple nodes of the same request.
- Runtime OpenAPI for the currently implemented auth, user, permohonan, document, and health endpoints.
- Phase 4 activity submissions through Sequence 3: survey/planning/NPS decisions, parallel Stage 4 preparation, conditional pole and PDKB branches, construction execution, and the Stage 6 join projections.

### Known gaps

- Phase 4 Sequences 4–5 activity-submission endpoints are not implemented yet.
- Document evidence attaches by workflow node, but list/download access has the same unresolved read-authorization/IDOR gap as permohonan detail.
- Generated permohonan test files under `modules/permohonan/tests` remain placeholders; meaningful workflow coverage lives beside the controller/service/repository and in RBAC tests.

### Verified baseline

`go test ./...` passes when Go uses a writable cache, for example:

```bash
GOCACHE=/tmp/colabora-go-build-cache go test ./...
```

Passing tests do not imply adequate permohonan workflow coverage because several generated test files currently contain only placeholder assertions.

## 4. Delivery phases

### Phase 0 — Foundation — Complete

The existing baseline above is accepted as the starting point. `PHASE5_DOCUMENTS.md` remains a historical design record for the already-built Garage document module; its filename does not define current phase status.

**Exit gate:** the application builds, the current test suite passes, migrations and seeders are registered, implemented endpoints appear in runtime OpenAPI, and the known legacy gaps are explicitly carried into the phases below.

### Phase 1 — Canonical workflow engine — Complete

Implemented in `pkg/workflow`: 21 canonical node definitions, nullable display/SLA references, deterministic evaluation, decision-controlled skips, validated start/completion transitions, terminal outcomes, and stage/action projections. `pkg/rbac/workflow_owner.go` provides exact-node role/unit authorization and caller-specific action metadata. These functions have no HTTP or persistence side effects; existing endpoints retain their legacy behavior until integration.

Verified with unit tests across all four connection types, all pole/PDKB combinations, opposite branch completion orders, prerequisite rejection, terminal return/completion, ownership matrices, unit boundaries, and detached input/output state. The living diagrams' conditional WO PDKB → PK Vendor dependency is reflected in both the registry and PRD.

Delivered one pure domain package that defines every canonical workflow node from the two living diagrams:

- stable code, display activity number where applicable, presentation stage, SLA activity reference, and connection-type owner;
- prerequisites and join conditions;
- pole and PDKB applicability rules;
- NPS delegated/returned terminal behavior;
- state evaluation for `locked`, `available`, `in_progress`, `completed`, and `skipped`;
- aggregate `in_progress`, `completed`, and `returned` status;
- derived `CurrentStage` and caller-specific available actions.

The evaluator must consume explicit state and decisions without querying HTTP or database concerns. Central authorization must resolve ownership from node definition and request context rather than stage ownership.

**Exit gate:** table-driven unit tests cover both connection families, every node, both conditional branches, NPS return, all parallel paths completed in either order, each join gate, ULP mismatch, and super-user read-only behavior.

### Phase 2 — Workflow-node persistence — Complete

- Add workflow-node identity to permohonan activity records and activity logs.
- Make display activity number and SLA deadline nullable.
- Store form-specific data as validated JSONB payload while keeping workflow-critical decisions typed.
- Replace `NpsKeputusan` with `NpsDelegationStatus` and remove `OwnerFnOverride` after projections no longer use it.
- Initialize every canonical node when a request is created; conditional nodes remain stored and become skipped rather than being deleted.
- Update document association from activity-number authorization to workflow-node authorization.
- Implement a new forward migration and rollback; do not rewrite shipped migrations. A development database reset is acceptable.

**Exit gate:** migration up/rollback succeeds on a clean development database; constraints prevent duplicate request/node rows; null SLA behavior is correct; all nodes initialize consistently; and legacy fields are no longer used by runtime code.

Decision recorded: one physical source file may evidence multiple workflow nodes within its one permohonan. `document_evidence` is the association table; it preserves one request binding per stored file and enforces a unique document/node association.

Implementation verification: `go test ./...` passes. On 8 September 2026, migration up → rollback → up succeeded against an isolated clean PostgreSQL database. The verification confirmed nullable activity/SLA metadata, required node identity, removal/restoration of legacy columns, duplicate request/node prevention, composite evidence foreign keys, and two node-evidence associations for one physical file in the same request.

### Phase 3 — Entry path and read model — Complete

Implemented connection-aware request creation, configured target-ULP validation for NPS-created PLG TM requests, canonical workflow-node/detail projections, caller-specific action links, node-derived SLA state, node-based `scope=mine`, and authenticated activity/log read endpoints. Runtime OpenAPI now describes those delivered contracts.

- Refactor creation authorization and request validation:
  - JTR/JTM: caller must be matching-ULP `pelayanan-pelanggan`; omit `ulp_unit` and derive it from the caller.
  - PLG TM: caller must be `nps`; require and validate a configured target `ulp_unit`.
- Complete the entry node, initialize the graph, and make the connection-specific survey node available in one transaction.
- Return workflow nodes and caller-specific `available_actions` from detail.
- Implement activities, documents, and logs read endpoints defined by `API_SPEC.md`.
- Make `scope=mine` query available nodes owned by the caller instead of `CurrentStage` and owner overrides.
- Derive stage and SLA projections without turning them back into transition state.

**Exit gate:** create/list/detail behavior matches `API_SPEC.md`; wrong role/type combinations return 403, invalid PLG TM target ULP returns 400, node-based task filtering works for parallel actions, and `docs/permohonan.yaml` is updated in the same change.

### Phase 4 — Activity endpoints — In progress (Sequences 1–3 complete)

Deliver write behavior in dependency slices so each slice is usable and tested before the next:

1. **Complete:** Survey, RAB/KKO/KKF with pole decision, Permohonan Perluasan, and NPS delegation/terminal return.
2. **Complete:** Parallel WO Tiang/Konstruksi/APP, material reservation and tera, PK Vendor, and conditional WO PDKB.
3. **Complete:** Conditional pole installation, construction, and PDKB documentation.
4. Parallel network energization and SR/APP installation.
5. Ordered PDL entry/mutation, AIL/DIJ archive, and terminal completion.

Every endpoint must authorize the exact node, reject unmet/non-applicable/completed/terminal state with 409, validate required structured data and evidence, attach documents, update decisions and projections, and append audit events within one service-owned transaction.

**Exit gate per slice:** service, repository, controller, validation, and authorization tests pass; duplicate and out-of-order submission is rejected; evidence failure rolls back all changes; both connection families follow the living diagrams; and runtime OpenAPI describes the delivered endpoints.

Detailed production payloads must be derived from `DOCUMENT_WORKFLOW_REFERENCE.md` and confirmed with business owners. For uncovered activities, implement only agreed workflow-critical fields and evidence requirements—do not infer a full production form from the hifi mockup.

Sequence 1 delivers the three authenticated write routes with exact node/ULP authorization, row-locked service transactions, required evidence attachment, validated workflow-critical fields, JSONB payloads, projections, node completion, and audit logs. Bundled RAB/pole and expansion/NPS nodes complete in dependency order within one transaction. Tests cover JTR/JTM and PLG TM ownership, duplicate and out-of-order rejection, delegated and terminal-return outcomes, repository persistence, controller error mapping, and rollback when evidence attachment fails. The RAB workbook is retained as evidence; calculation parity remains intentionally out of scope pending authoritative formulas and catalogue ownership.

Sequence 2 delivers the six Stage 4 routes for the three parallel WO branches, bundled material reservation/tera, conditional WO PDKB, and PK Vendor. WO Konstruksi records the typed `perlu_pdkb` decision: the false branch audits and persists both PDKB skips before opening PK Vendor, while the true branch gates PK Vendor on WO PDKB. Tests cover both connection families, parallel completion in opposite orders, both pole/PDKB branches, exact owners, terminal/out-of-order/duplicate rejection, bundled evidence rollback, and audit events for derived skips. Payloads remain intentionally minimal where production fields are still discovery gates.

Sequence 3 delivers the explicit vendor-execution selector for pole installation versus construction, plus conditional PDKB documentation. Exact-node authorization prevents vendor roles from submitting one another's work. Tests cover both connection families, pole and construction completion in opposite orders, inapplicable/out-of-order/duplicate rejection, PDKB required and skipped paths, evidence failure, and both Stage 6 join projections. Activity #11/#12 and PDKB payloads remain evidence-first with optional notes because their detailed production forms are still discovery gates.

### Phase 5 — Evidence and access hardening — Queued

- Define one read policy for permohonan detail, activity history, logs, and documents, including ULP participants, UP3 roles, vendors, and cross-unit read-only super-user access.
- Apply that policy consistently to list/detail/document reads so closing the document IDOR does not leave customer data exposed elsewhere.
- Make document attachment atomic with node completion and reject missing, already-attached, unauthorized, or mismatched documents.
- Finalize required provenance fields, MIME/size enforcement, private object access, short-lived downloads, malware-scanning integration point, revision/supersession behavior, and orphan-upload cleanup.

**Exit gate:** authorization matrix tests cover every role and unit boundary; guessed IDs cannot expose aggregate or document data; failed attachment leaves both workflow and documents unchanged; and security-sensitive audit events contain no PII or secret URLs.

### Phase 6 — Integration readiness and frontend handoff — Queued

- Add end-to-end lifecycle tests for JTR/JTM and both PLG TM variants, including pole/PDKB combinations, parallel completion in either order, joins, terminal return, and final completion.
- Validate migration up/rollback, seeding, pagination/filtering, SLA calculation, Garage configuration, and merged OpenAPI.
- Publish a frontend handoff based on stable workflow-node codes and `available_actions`, not hifi filenames or stage-owner approximations.
- Document deployment prerequisites, observability for failed transitions/storage, backup expectations, and rollback procedure.

**Exit gate:** automated tests exercise every canonical path, the merged OpenAPI validates and matches runtime behavior, security gates pass, a clean environment can migrate/seed/run, and the frontend can drive actions exclusively from server responses.

## 5. Canonical SLA offsets

Offsets are calendar days from the request date. `H` and `G` are retained as source-process reference labels. Activity #5, #3b, PK Vendor, WO PDKB, and PDKB documentation have no independent SLA rule unless the business adds one.

| # | Activity | JTR | JTM/Gardu | PLG TM <5 GWNG | PLG TM >5 GWNG |
|---|---|---|---|---|---|
| 1 | Permohonan PB/PD | H | H | G | G |
| 2 | Survei Perluasan Jaringan | H+2 | H+2 | G+2 | G+2 |
| 3 | Perhitungan RAB, KKO & KKF | H+2 | H+2 | G+2 | G+2 |
| 4 | Permohonan Perluasan | H+2 | H+2 | H | H |
| 6 | WO Vendor Tiang | H+2 | H+2 | H+1 | H+1 |
| 7 | WO Vendor Konstruksi | H+2 | H+2 | H+1 | H+1 |
| 8 | WO Vendor APP | H+2 | H+2 | H+1 | H+1 |
| 9 | Reservasi Material | H+2 | H+2 | H+2 | H+2 |
| 10 | Perakitan dan Tera APP | H+3 | H+3 | H+2 | H+2 |
| 11 | Pelaksanaan Pemasangan Tiang | H+6 | H+8 | H+8 | H+12 |
| 12 | Pelaksanaan Konstruksi | H+8 | H+12 | H+19 | H+49 |
| 13 | Pengoperasian Jaringan Listrik | H+9 | H+13 | H+20 | H+50 |
| 14 | Pemasangan SR/APP dan Penyalaan | H+10 | H+14 | H+20 | H+50 |
| 15 | Entri dan Mutasi PDL | H+10 | H+10 | H+20 | H+50 |
| 16 | Arsip AIL / Updating DIJ | H+10 | H+14 | H+20 | H+50 |
| 17 | Selesai | H+10 | H+14 | H+20 | H+50 |

Keep this table synchronized with `database/seeders/json/sla_rules.json`. The current runtime treats “due soon” as two calendar days; retain that behavior until the business confirms a different threshold.

## 6. Discovery gates

These decisions must be confirmed before their dependent implementation is considered complete:

- exact required evidence and structured fields for Activities #8, #11, #12, #14–#17, and the PDKB branch;
- whether relay/OCR evidence belongs entirely to `tera_app` or gates another node;
- correction/revision behavior for completed nodes and superseded documents;
- authoritative RAB calculation formulas, reference catalogues, versions, review, and generated-output ownership;
- exact participant/view-access matrix and retention policy;
- business definition of SLA reference labels and the `due_soon` threshold.

Until confirmed, preserve original operational artefacts as private evidence, use synthetic data in tests and documentation, and do not claim calculation or form parity.

## 7. Definition of done

A phase is complete only when its exit gate passes and:

- code follows Controller → Service → Repository boundaries and services own transactions;
- authorization and transition logic remain centralized;
- migrations include rollback and do not modify shipped migration history;
- tests cover success, authorization, invalid state, rollback, and relevant branch/join behavior;
- runtime OpenAPI changes together with implemented behavior;
- no real customer, pricing, location, signature, or network data enters source, fixtures, logs, or examples;
- this document's phase status and current baseline are updated in the same change.
