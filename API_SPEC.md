# API_SPEC.md — COLABORA Target REST Contract

This document describes the target API after the workflow-node refactor. Runtime OpenAPI under `docs/` continues to describe implemented code and must not claim this target is available early.

The API is not coupled to hifi form filenames. Detailed production forms may replace the prototype while retaining these workflow-node identities and dependencies.

## Read endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/permohonan` | Paginated dashboard list and `scope=mine` filtering |
| `GET` | `/api/permohonan/:id` | Aggregate detail, workflow nodes, and caller-specific available actions |
| `GET` | `/api/permohonan/:id/activities` | Full node timeline with display activity/stage/SLA metadata |
| `GET` | `/api/permohonan/:id/documents` | Attached evidence, optionally filtered by `workflow_node` |
| `GET` | `/api/permohonan/:id/logs` | Audit history |

List filters remain `ulp`, `jenis_sambungan`, `stage`, aggregate `status`, derived `sla`, `search`, and `scope=mine|all`. `scope=mine` returns requests where the caller owns at least one available node.

Detail/list items replace the legacy single `can_act` decision with:

```json
{
  "current_stage": 4,
  "status": "in_progress",
  "available_actions": [
    {
      "workflow_node": "wo_konstruksi",
      "activity_number": 7,
      "stage_number": 4,
      "method": "POST",
      "path": "/api/permohonan/{id}/wo-vendor/konstruksi"
    }
  ]
}
```

Only actions owned by the authenticated caller appear in `available_actions`. The full activity endpoint can still show locked/available nodes without granting permission.

## Create contract

`POST /api/permohonan` accepts customer/request fields plus `jenis_sambungan`.

- For JTR/JTM, caller must be `pelayanan-pelanggan`; request must omit `ulp_unit`, which is derived from the caller.
- For either PLG TM variant, caller must be `nps`; `ulp_unit` is required and must match a configured ULP.
- Other role/type combinations return 403. Invalid or missing PLG TM target ULP returns 400.
- Successful creation completes node `permohonan`, initializes all workflow nodes, and makes `survei` available to the correct role.

## Write endpoints

`POST /api/permohonan/:id/vendor-assignments` records a vendor account assignment
using `{"vendor_id":"<UUID>","vendor_role":"vendor-tiang|vendor-konstruksi|vendor-sr-app"}`.
It returns the aggregate with HTTP 200; an occupied role slot returns 409.
Assignment ownership, applicability and read scope are specified in `RBAC.md`.
No automatic vendor grants are created for existing requests.

All request-resource HTTP endpoints enforce the same read policy: matching ULP
for ULP roles, explicit assignment for vendors, cross-ULP operational access for
the listed UP3 roles, and read-only monitoring for super-user. Inaccessible IDs
return 404. List totals and pages include only visible requests, even with
`scope=all`.

| Method | Path | Workflow node(s) | Owner |
|---|---|---|---|
| `POST` | `/api/permohonan/:id/survei` | `survei` | `teknik` JTR/JTM; `perencanaan` PLG TM |
| `POST` | `/api/permohonan/:id/rab-kko-kkf` | `rab_kko_kkf`, `kebutuhan_tiang` | `teknik` JTR/JTM; `perencanaan` PLG TM |
| `POST` | `/api/permohonan/:id/permohonan-perluasan` | `permohonan_perluasan`, `nps_delegation` | `nps` |
| `POST` | `/api/permohonan/:id/wo-vendor/tiang` | `wo_tiang` | `perencanaan` |
| `POST` | `/api/permohonan/:id/wo-vendor/konstruksi` | `wo_konstruksi`; records `perlu_pdkb` | `konstruksi` |
| `POST` | `/api/permohonan/:id/wo-vendor/app` | `wo_app` | `transaksi-energi` |
| `POST` | `/api/permohonan/:id/reservasi-material` | `reservasi_material`, `tera_app` | `transaksi-energi` |
| `POST` | `/api/permohonan/:id/pk-vendor` | `pk_vendor` | `konstruksi` |
| `POST` | `/api/permohonan/:id/wo-pdkb` | `wo_pdkb` | `konstruksi` when PDKB required |
| `POST` | `/api/permohonan/:id/pelaksanaan-konstruksi` | `pemasangan_tiang` or `pelaksanaan_konstruksi`, selected explicitly in body | Matching vendor role |
| `POST` | `/api/permohonan/:id/pdkb-dokumentasi` | `pdkb_documentation` | `pdkb` when required |
| `POST` | `/api/permohonan/:id/energize-jaringan` | `energize_jaringan` | `teknik` JTR/JTM; `jaringan` PLG TM |
| `POST` | `/api/permohonan/:id/pemasangan-sr-app` | `pemasangan_sr_app` | `vendor-sr-app` JTR/JTM; `vendor-konstruksi` PLG TM |
| `POST` | `/api/permohonan/:id/closing` | `entri_mutasi_pdl`, `arsip_ail`, `selesai` | Matching ULP `pelayanan-pelanggan` |

The NPS request body uses `nps_delegation_status: delegated|returned`. `returned` terminally closes the workflow with aggregate status `returned`; it is not a draft/rework loop.

The WO Konstruksi request body requires the typed decision `perlu_pdkb: true|false`. Sequence 2 activities without another confirmed workflow-critical field accept optional notes and require evidence. The bundled reservasi/tera request exposes separate optional `reservation_notes` and `tera_notes`; detailed material and relay/OCR fields remain discovery items rather than inferred production contracts.

The construction-execution request body selects the exact node with `workflow_node: pemasangan_tiang|pelaksanaan_konstruksi`; the service then applies that node's vendor ownership and prerequisites. PDKB documentation uses its dedicated endpoint. Sequence 3 keeps uncovered production fields minimal—required evidence plus optional notes—until the business confirms the detailed forms.

The energize request requires `operation_result` plus document evidence for the operation BA and optional notes. SR/APP installation requires evidence plus optional notes while its detailed production form remains a discovery gate. These two Sequence 4 nodes are independent: each uses its own prerequisites and Activity #15 opens only after both complete.

Closing accepts the existing evidence request (`document_ids` required, `notes` optional). One submission records the completed closing package in dependency order: `entri_mutasi_pdl` → `arsip_ail` → `selesai`. The same documents and notes are associated with each node. All three transitions, attachments and audit events commit together; any failure rolls back the entire package. Both energize and SR/APP must already be complete, and the caller must be `pelayanan-pelanggan` in the target ULP for every connection type. Success returns `status: completed` and empty `available_actions`; duplicate or terminal-return submissions return 409. Detailed fields for #15–#17 and verification of document contents remain discovery items.

Every write endpoint:

1. authenticates the caller and authorizes the exact workflow node;
2. returns 409 when prerequisites are unmet, the node is skipped/completed, or the aggregate is terminal;
3. validates and attaches required `document_ids` to the workflow node;
4. writes payload, node state, projections, skips/unlocks, and an audit event in one transaction;
5. returns the updated aggregate with caller-specific `available_actions`.

## Documents

- `POST /api/documents` uploads a private, unattached file and returns its ID plus provenance. Optional `supersedes_document_id` creates the next revision only when the prior file is a same-type, unattached upload owned by the caller.
- Activity submissions attach `document_ids` to their exact `workflow_node`.
- `GET /api/permohonan/:id/documents?workflow_node=...` filters attached evidence.
- `GET /api/permohonan/:id/documents/:doc_id` returns/redirects to a short-lived download URL.

An attachment request must fail atomically if any document is missing, already attached, or the caller does not own the target node.
Superseded documents cannot be attached. Attached evidence cannot be revised or replaced through the current API because correction/reopen ownership remains a discovery gate. Uploads record detected MIME, measured size, SHA-256, original filename, `uploaded` source, `restricted` classification, scan status, and revision links. Configured ClamAV scanning is synchronous and fail-closed; without a configured scanner, status is explicitly `not_scanned`. Downloads redirect to private 15-minute presigned URLs. The operational `make cleanup-orphan-documents` command removes unattached uploads older than the configured TTL.

## Error semantics

- 400: malformed input, invalid enum, or invalid/missing PLG TM target ULP.
- 401: missing or invalid authentication.
- 403: caller does not own the requested action or violates ULP scope.
- 404: aggregate/document not found under the requested resource.
- 409: valid action type but wrong workflow state, unmet prerequisites, terminal aggregate, or non-applicable branch.

## Runtime OpenAPI policy

`docs/permohonan.yaml` documents the implemented Phase 3 create/read contract and all Phase 4 write endpoints through terminal completion.
