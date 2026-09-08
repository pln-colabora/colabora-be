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

Every write endpoint:

1. authenticates the caller and authorizes the exact workflow node;
2. returns 409 when prerequisites are unmet, the node is skipped/completed, or the aggregate is terminal;
3. validates and attaches required `document_ids` to the workflow node;
4. writes payload, node state, projections, skips/unlocks, and an audit event in one transaction;
5. returns the updated aggregate with caller-specific `available_actions`.

## Documents

- `POST /api/documents` uploads a private, unattached file and returns its ID.
- Activity submissions attach `document_ids` to their exact `workflow_node`.
- `GET /api/permohonan/:id/documents?workflow_node=...` filters attached evidence.
- `GET /api/permohonan/:id/documents/:doc_id` returns/redirects to a short-lived download URL.

An attachment request must fail atomically if any document is missing, already attached, or the caller does not own the target node.

## Error semantics

- 400: malformed input, invalid enum, or invalid/missing PLG TM target ULP.
- 401: missing or invalid authentication.
- 403: caller does not own the requested action or violates ULP scope.
- 404: aggregate/document not found under the requested resource.
- 409: valid action type but wrong workflow state, unmet prerequisites, terminal aggregate, or non-applicable branch.

## Runtime OpenAPI policy

`docs/permohonan.yaml` documents the implemented Phase 3 create and read contract. Phase 4 write endpoints remain target-only here until their runtime handlers are delivered.
