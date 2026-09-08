# RBAC.md — Backend Authorization Model

Authorization is evaluated per workflow node. Stage ownership from the hifi prototype is useful for display, but is too coarse for write authorization because several roles can own different parallel nodes in one stage.

The living ownership source is `hifi-colabora/workflow/jtr-jtm.html` plus `hifi-colabora/workflow/plg-tm.html`. Update this document whenever those diagrams change.

## Roles

`pelayanan-pelanggan`, `teknik`, `perencanaan`, `konstruksi`, `transaksi-energi`, `jaringan`, `nps`, `pdkb`, `vendor-tiang`, `vendor-konstruksi`, `vendor-sr-app`, and `super-user`.

`super-user` has cross-unit read access and no workflow write access.

## Node ownership

| Node(s) | JTR/JTM owner | PLG TM owner |
|---|---|---|
| `permohonan` | `pelayanan-pelanggan` | `nps` |
| `survei` | `teknik` | `perencanaan` |
| `rab_kko_kkf`, `kebutuhan_tiang` | `teknik` | `perencanaan` |
| `permohonan_perluasan`, `nps_delegation` | `nps` | `nps` |
| `wo_tiang` | `perencanaan` | `perencanaan` |
| `wo_konstruksi`, `pk_vendor`, `wo_pdkb` | `konstruksi` | `konstruksi` |
| `wo_app`, `reservasi_material`, `tera_app` | `transaksi-energi` | `transaksi-energi` |
| `pemasangan_tiang` | `vendor-tiang` | `vendor-tiang` |
| `pelaksanaan_konstruksi` | `vendor-konstruksi` | `vendor-konstruksi` |
| `pdkb_documentation` | `pdkb` | `pdkb` |
| `energize_jaringan` | `teknik` | `jaringan` |
| `pemasangan_sr_app` | `vendor-sr-app` | `vendor-konstruksi` |
| `entri_mutasi_pdl`, `arsip_ail`, `selesai` | `pelayanan-pelanggan` | `pelayanan-pelanggan` |

## Authorization rules

A write is allowed only when all conditions hold:

1. The workflow node exists and is currently `available` or `in_progress`.
2. The caller's role matches the node owner resolved for `JenisSambungan`.
3. For ULP-scoped roles (`pelayanan-pelanggan`, `teknik`), caller unit equals `Permohonan.UlpUnit`.
4. Conditional applicability is true; skipped/locked/completed nodes reject writes.
5. The aggregate is `in_progress`; completed and returned requests are immutable.

Activity 1 is special because the aggregate does not yet exist:

- JTR/JTM creation requires `pelayanan-pelanggan`; `ulp_unit` is taken from the caller.
- PLG TM creation requires `nps`; a valid target `ulp_unit` is required in the request.

## Implementation target

`OwnsWorkflowNode`, `AuthorizeWorkflowNode`, and `AvailableActions` in `pkg/rbac/workflow_owner.go` accept domain values and evaluate `pkg/workflow` snapshots. Read projections use them for node-based task filtering and caller-specific actions. Implemented Phase 4 write endpoints use `AuthorizeWorkflowNode` inside the service-owned transaction so ownership, applicability, prerequisites, and aggregate state are checked against the locked aggregate.

- Use the central `OwnsWorkflowNode(user, permohonan, nodeCode)` policy; the numbered activity middleware remains compatibility-only.
- `RequireWorkflowNodeOwner(nodeCode)` returns 403 for the wrong owner and 409 when the node is owned by the role but is not actionable.
- `OwnerFnOverride` is retired as an authorization source. Available owners derive from node state.
- `scope=mine` means at least one available or resumable node is owned by the caller, rather than matching `CurrentStage`.
- Detail and list responses expose `available_actions`; a single `can_act` boolean cannot represent parallel work.

Read authorization remains broader than write ownership. The existing document/permohonan read-access gap remains tracked in `PHASE5_DOCUMENTS.md` and must be resolved consistently across both modules.
