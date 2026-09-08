# DATA_MODEL.md — COLABORA Target Entities

This is the data model for the dependency-based workflow in `PRD.md`. Phase 2 implements its persistence model; Phases 3 and 4 implement its entry/read and transition APIs through terminal closing.

## Design principles

- A stable workflow-node code is the persistence and authorization identity.
- The source activity number remains nullable display/SLA metadata. It cannot represent decision `3b` or supporting nodes such as PK Vendor and PDKB documentation by itself.
- Presentation stage is derived and must never determine transition order.
- Workflow status and SLA state are separate concerns.
- Branch fields used outside one form remain typed columns on `Permohonan`; other form data stays JSONB on the node record.

## `Permohonan`

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID | Primary key |
| `NoPermohonan` | varchar(30), unique | `PBPD-{year}-{sequence}` |
| `JenisPermohonan` | varchar(30) | Pasang Baru or Perubahan Daya |
| `JenisSambungan` | varchar(30) | JTR, JTM/Gardu, PLG TM <5 GWNG, or PLG TM >5 GWNG |
| `UlpUnit` | varchar(50) | Caller ULP for JTR/JTM; explicitly selected target ULP for NPS-created PLG TM |
| Customer fields | varchar | Name, address, and phone captured at Activity #1 |
| `RequestDate` | date | H/G SLA reference date |
| `CurrentStage` | smallint | Derived lowest stage containing an applicable unfinished node |
| `Status` | varchar(20) | `in_progress`, `completed`, or terminal `returned` |
| `KebutuhanTiang` | nullable boolean | Activity #3b branch decision |
| `NpsDelegationStatus` | nullable varchar(20) | `delegated` or terminal `returned` |
| `PerluPdkb` | nullable boolean | Decision recorded with WO Konstruksi |
| `CreatedBy` | UUID FK → users | Customer service for JTR/JTM; NPS for PLG TM |
| Timestamps | | Created/updated timestamps |

`OwnerFnOverride` is removed from the target model. Owners are derived from available workflow nodes.

## `PermohonanWorkflowNode`

One row per canonical workflow node for a request. All codes are listed in `PRD.md`; conditional nodes are retained and marked `skipped` when inapplicable.

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID | Primary key |
| `PermohonanID` | UUID FK | Unique together with `WorkflowNode` |
| `WorkflowNode` | varchar(50) | Stable code such as `rab_kko_kkf` or `pdkb_documentation` |
| `ActivityNumber` | nullable smallint | Source display/SLA activity; null for 3b/supporting nodes without their own number |
| `StageNumber` | smallint | Presentation grouping metadata |
| `Status` | varchar(20) | `locked`, `available`, `in_progress`, `completed`, or `skipped` |
| `SlaDeadline` | nullable date | Null when the node has no SLA rule |
| `Payload` | jsonb | Form-specific values validated at the DTO/service boundary |
| `CompletedBy` | nullable UUID FK → users | Completing actor |
| `CompletedAt` | nullable timestamp | Completion time |
| Timestamps | | Created/updated timestamps |

Create all canonical nodes when the request is created so locked, skipped, and available states are queryable consistently. Branch decisions change status; they do not delete conditional rows.

## `SLARule`

One row per source activity number and connection type where an SLA is defined:

- `ActivityNumber`, `ActivityName`, `JenisSambungan`, `OffsetDays`, and `ReferencePoint`.
- Unique constraint on `(activity_number, jenis_sambungan)`.
- Activity #5 and supporting workflow nodes have no rule and therefore a null node deadline.
- The SLA seed remains synchronized with the table in [`DEVELOPMENT.md`](./DEVELOPMENT.md).

## `Document`

Evidence remains stored privately in Garage and downloaded through a presigned URL.

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID | Primary key |
| `PermohonanID` | nullable UUID FK | Null during upload-first state |
| `FilePath` | varchar | Private object key, not a public URL |
| `UploadedBy` | UUID FK | Uploading actor |
| Timestamps | | Created/updated timestamps |

One file belongs to at most one `Permohonan`, but may evidence multiple nodes of that request through `DocumentEvidence`.

## `DocumentEvidence`

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID | Primary key |
| `DocumentID`, `PermohonanID` | UUID | Composite FK to the file's one request binding |
| `WorkflowNode` | varchar(50) | Composite FK to `PermohonanWorkflowNode` |
| `AttachedBy` | UUID FK → users | Actor who associated the evidence |
| Timestamps | | Created/updated timestamps |

`(DocumentID, WorkflowNode)` is unique. Attachment authorization evaluates the target `WorkflowNode`, not stage or activity number alone.

## `ActivityLog`

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID | Primary key |
| `PermohonanID` | UUID FK | Aggregate |
| `WorkflowNode` | nullable varchar(50) | Null for aggregate-only events |
| `ActivityNumber` | nullable smallint | Display/reporting metadata |
| `Actor` | UUID FK → users | Event actor |
| `Action` | varchar(100) | For example `node_completed`, `node_skipped`, `nps_delegated`, `nps_returned`, `pdkb_required`, or `pdkb_not_required` |
| `Detail` | nullable text | Human-readable context |
| Timestamps | | `created_at` is event time |

## Transition invariants

- Node state, aggregate projections, document attachment, and activity log are committed in one transaction.
- Completed/skipped nodes cannot be submitted again.
- A `returned` or `completed` aggregate has no available actions.
- `CurrentStage` and `available_actions` are recomputed after each transition.
- SLA state is calculated at read time from nullable deadline plus workflow completion state; `overdue` is not stored as workflow status.

## Migration strategy

Breaking changes are allowed. `20260908120000_refactor_workflow_nodes` adds the workflow-node fields, maps retained numbered history, removes legacy ownership/NPS naming, and creates evidence associations. A development database reset is acceptable; do not edit already-shipped migration files in place.
