# Workflow Alignment and Refactor Contract

## Status

The parent repository records `hifi-colabora` at `c580e04`, while the nested working tree is currently at `8497536`. The newer workflow makes PLG TM Activity 1 owned by NPS, Activity 2 owned by Perencanaan, and reframes the NPS action as delegation/return.

The two files in `hifi-colabora/workflow/` are the living source of truth and will continue to evolve even if the remaining hifi prototype is retired. Every implementation session must diff/re-read both diagrams before relying on this document.

## Locked decisions

- JTR/JTM creation: `pelayanan-pelanggan`; survey: `teknik`.
- PLG TM creation: `nps`, with required target `ulp_unit`; survey: `perencanaan`.
- Activity order is #3/3b before #4/#5 regardless of form filename ordering.
- NPS contract field is `nps_delegation_status` with `delegated|returned`.
- `returned` is terminal and retained for audit.
- WO Tiang, WO Konstruksi, and WO APP all wait for NPS delegation.
- Backend target is a workflow-node DAG; presentation stage is not the state machine.
- Breaking development schema/API changes and development-data reset are allowed.
- Runtime OpenAPI remains truthful to implemented code until the refactor lands.

## Known current-code gaps

- Create accepts only `pelayanan-pelanggan` and derives ULP only from the caller.
- Stage 1/2 ownership does not implement the PLG TM split.
- `OwnsActivity` delegates to coarse stage ownership and can authorize the wrong action within a parallel stage.
- `CurrentStage`, one `OwnerFnOverride`, and one `can_act` cannot represent simultaneous actions.
- `ActivityNumber` cannot identify 3b/supporting PDKB/PK nodes; Activity #5 has no SLA rule while deadlines are non-null.
- Documents attach by activity number instead of stable workflow node.

## Delivery sequence

1. Re-read/diff both living workflow diagrams and synchronize design docs if they changed again.
2. Add canonical node definitions and dependency/authorization evaluator with unit tests.
3. Migrate persistence to workflow-node identity, nullable display activity/deadline, separate workflow/SLA state, and new NPS naming.
4. Fix create/survey ownership and expose `available_actions`/node-based `scope=mine`.
5. Add write endpoints by dependency slice: planning; parallel WO/material; construction/PDKB; energize/SR-APP; closing.
6. Update runtime OpenAPI in the same changes that implement each target contract.
7. Add full branch/join/unit/terminal/read-authorization integration coverage.

Detailed behavior is normative in `PRD.md`; target persistence, API, and authorization are in `DATA_MODEL.md`, `API_SPEC.md`, and `RBAC.md`.
