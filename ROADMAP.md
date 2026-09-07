# ROADMAP.md — COLABORA Build Order

The workflow diagrams under `hifi-colabora/workflow/` are continuously updated. Re-check them and synchronize `PRD.md`, `DATA_MODEL.md`, `API_SPEC.md`, and `RBAC.md` before starting any workflow phase.

## Current state

- Auth/user, SLA rules, core permohonan create/list/detail, initial stage-based RBAC helpers, audit entities, and the standalone document module exist.
- Activity-submission endpoints and the dependency evaluator do not exist yet.
- Current creation and RBAC behavior still reflects the older workflow; `docs/permohonan.yaml` intentionally documents that runtime until code is refactored.
- Breaking development schema/API changes are allowed; no compatibility layer is required.

## Phase 1 — canonical workflow package

- Define stable workflow-node codes, display activity numbers, stage metadata, connection-type owners, applicability conditions, prerequisites, and join gates in one package.
- Add a transition evaluator that derives locked/available/completed/skipped nodes, aggregate status, `CurrentStage`, and `available_actions`.
- Cover both workflow diagrams with table-driven unit tests before adding handlers.

## Phase 2 — persistence refactor

- Add `workflow_node` to activity records and make display activity number/SLA deadline nullable where appropriate.
- Separate workflow status from derived SLA status.
- Replace `NpsKeputusan` with `NpsDelegationStatus` (`delegated|returned`).
- Retain `CurrentStage` only as a derived dashboard projection and retire `OwnerFnOverride` after development data is reset/migrated.
- Attach documents to `workflow_node`; retain nullable display activity number for SLA/reporting compatibility.

## Phase 3 — entry path and read model

- Refactor create authorization: JTR/JTM by matching ULP customer service; PLG TM by NPS with required target ULP.
- Resolve survey owner as `teknik` for JTR/JTM and `perencanaan` for PLG TM.
- Return workflow nodes and `available_actions` from detail; implement `scope=mine` using available node ownership.
- Update `docs/permohonan.yaml` only when runtime behavior matches the new contract.

## Phase 4 — activity endpoints in dependency slices

1. Survey, RAB/pole decision, Permohonan Perluasan, and terminal/delegated NPS outcome.
2. Parallel WO branches, material reservation/tera, PK Vendor, and conditional WO PDKB.
3. Parallel pole/construction work and conditional PDKB documentation.
4. Parallel energize and SR/APP work.
5. Ordered PDL, AIL, and terminal completion.

Every endpoint must use the central evaluator, require evidence, append an audit event, and recompute projections in one transaction. Out-of-order or non-applicable submissions return 409.

## Phase 5 — view authorization and integration hardening

- Define one view policy for permohonan details and evidence documents, closing the current IDOR consistently.
- Add end-to-end tests for both connection families, conditional branches, parallel completion in either order, joins, terminal return, unit scoping, and super-user read-only behavior.
- Validate migration up/rollback, OpenAPI, and document attachment against workflow-node codes.

## Phase 6 — frontend handoff

Treat hifi forms/detail pages as disposable prototypes. Detailed production forms may replace them without changing workflow-node identity or backend dependencies. Continue using `hifi-colabora/workflow/` as the living workflow input even if the rest of `hifi-colabora` is retired.
