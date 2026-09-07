# PRD.md — COLABORA Backend Product Requirements

COLABORA is a workflow-tracking backend for PLN Permohonan PB/PD. It persists process progress, evidence, SLA state, authorization, and audit history across ULP, UP3, and vendor roles.

## Source-of-truth hierarchy

1. [`hifi-colabora/workflow/jtr-jtm.html`](./hifi-colabora/workflow/jtr-jtm.html) and [`hifi-colabora/workflow/plg-tm.html`](./hifi-colabora/workflow/plg-tm.html) are the living source of truth for process order, dependencies, branches, and role ownership. Agents must inspect both before changing workflow behavior.
2. This PRD translates those diagrams into backend behavior. When the diagrams change, update this PRD and its companion documents before implementing code.
3. `hifi-colabora/DEVELOPMENT.md` carries supporting domain and SLA detail. The remaining hifi forms/detail pages are illustrative prototypes, may lag the workflow diagrams, and may be retired when detailed production forms are specified. Do not infer backend behavior from their filenames or hardcoded demo state.
4. `docs/*.yaml` describes APIs that are actually implemented. [`API_SPEC.md`](./API_SPEC.md) describes the target contract.

Companion documents: [`DATA_MODEL.md`](./DATA_MODEL.md), [`API_SPEC.md`](./API_SPEC.md), [`RBAC.md`](./RBAC.md), and [`ROADMAP.md`](./ROADMAP.md).

## 1. Product behavior

A `Permohonan` follows 17 numbered business activities grouped into 7 presentation stages. The activities are not a simple counter: the process contains decisions, optional work, parallel branches, and join gates. The backend must evaluate explicit workflow-node prerequisites rather than incrementing an activity or stage number.

The backend must:

1. Persist the request, applicable workflow nodes, evidence, SLA deadlines, and audit events.
2. Resolve authorization per workflow node, connection type, and ULP scope.
3. Expose every action currently available to the caller, including parallel actions.
4. Compute presentation stage and SLA state from node state.
5. Preserve terminal returned/completed requests for reporting and audit.

## 2. Roles and connection-type ownership

| Role (`fn`) | Responsibility |
|---|---|
| `pelayanan-pelanggan` | Creates JTR/JTM requests for its own ULP and completes closing activities |
| `teknik` | Surveys and prepares RAB for JTR/JTM in its own ULP; operates JTR/JTM networks |
| `nps` | Creates PLG TM requests, chooses their target ULP, and delegates or returns all requests after planning |
| `perencanaan` | Surveys and prepares RAB for PLG TM; issues applicable WO Vendor Tiang |
| `konstruksi` | Issues WO Vendor Konstruksi, PK Vendor, and conditional WO PDKB |
| `transaksi-energi` | Issues WO Vendor APP, reserves material, and completes APP assembly/tera |
| `jaringan` | Operates PLG TM networks |
| `pdkb` | Uploads conditional PDKB execution documentation |
| `vendor-tiang` | Installs poles when required |
| `vendor-konstruksi` | Executes network construction and PLG TM SR/APP work |
| `vendor-sr-app` | Executes JTR/JTM SR/APP work |
| `super-user` | Read-only cross-unit monitoring; never owns a write action |

ULP roles (`pelayanan-pelanggan`, `teknik`) may act only when `User.Unit == Permohonan.UlpUnit`. An NPS-created PLG TM request must explicitly provide a valid target `ulp_unit`; it is never inferred from the NPS user's UP3 unit.

## 3. Canonical workflow dependencies

Stages are visual groupings, not global barriers. A later-stage node may become available while another independent branch remains in an earlier stage. `CurrentStage` is therefore a derived dashboard projection only.

| Node | Display activity | Owner | Prerequisites / applicability |
|---|---|---|---|
| `permohonan` | #1 Permohonan PB/PD | `pelayanan-pelanggan` for JTR/JTM; `nps` for PLG TM | Entry node |
| `survei` | #2 Survei | `teknik` for JTR/JTM; `perencanaan` for PLG TM | `permohonan` completed |
| `rab_kko_kkf` | #3 RAB/KKO/KKF | `teknik` for JTR/JTM; `perencanaan` for PLG TM | `survei` completed |
| `kebutuhan_tiang` | #3b decision | Same owner as #3 | Completed with the RAB submission |
| `permohonan_perluasan` | #4 Permohonan Perluasan | `nps` | RAB and pole decision completed |
| `nps_delegation` | #5 Delegasi Perintah Kerja NPS | `nps` | Activity #4 completed; outcome is `delegated` or `returned` |
| `wo_tiang` | #6 WO Vendor Tiang | `perencanaan` | NPS delegated and `kebutuhan_tiang = true`; otherwise skipped |
| `wo_konstruksi` | #7 WO Vendor Konstruksi | `konstruksi` | NPS delegated |
| `wo_app` | #8 WO Vendor APP | `transaksi-energi` | NPS delegated |
| `reservasi_material` | #9 Reservasi Material | `transaksi-energi` | WO APP completed |
| `tera_app` | #10 Perakitan dan Tera APP | `transaksi-energi` | Reservasi material completed |
| `pk_vendor` | Supporting workflow node | `konstruksi` | WO Konstruksi completed |
| `wo_pdkb` | Conditional supporting node | `konstruksi` | WO Konstruksi completed and `perlu_pdkb = true`; otherwise skipped |
| `pemasangan_tiang` | #11 Pemasangan Tiang | `vendor-tiang` | WO Tiang completed; skipped when poles are not required |
| `pelaksanaan_konstruksi` | #12 Pelaksanaan Konstruksi | `vendor-konstruksi` | WO Konstruksi and PK Vendor completed, plus WO PDKB when required |
| `pdkb_documentation` | Conditional supporting node | `pdkb` | Construction completed and `perlu_pdkb = true`; otherwise skipped |
| `energize_jaringan` | #13 Pengoperasian Jaringan | `teknik` for JTR/JTM; `jaringan` for PLG TM | Construction, applicable pole work, and applicable PDKB documentation completed |
| `pemasangan_sr_app` | #14 Pemasangan SR/APP | `vendor-sr-app` for JTR/JTM; `vendor-konstruksi` for PLG TM | Construction and APP tera completed |
| `entri_mutasi_pdl` | #15 Entri dan Mutasi PDL | `pelayanan-pelanggan` | Activities #13 and #14 completed |
| `arsip_ail` | #16 Arsip AIL / Updating DIJ | `pelayanan-pelanggan` | Activity #15 completed |
| `selesai` | #17 Selesai | `pelayanan-pelanggan` | Activity #16 completed; marks aggregate completed |

The filenames `forms/03-permohonan-perluasan.html` and `forms/04-rab-kko-kkf.html` do not represent execution order. Activity #3 always precedes Activities #4–5.

### Branch outcomes

- `kebutuhan_tiang = false` marks `wo_tiang` and `pemasangan_tiang` as skipped.
- `perlu_pdkb = false` marks `wo_pdkb` and `pdkb_documentation` as skipped.
- `nps_delegation_status = delegated` unlocks applicable Stage 4 work.
- `nps_delegation_status = returned` is terminal. The request remains stored with aggregate status `returned`; no downstream node becomes available.

## 4. Workflow and SLA state

Workflow-node status is one of `locked`, `available`, `in_progress`, `completed`, or `skipped`. SLA state is derived separately as `on_time`, `due_soon`, `overdue`, or `none`. A node without an SLA rule must not receive a fabricated deadline. The exact day offsets remain in `hifi-colabora/DEVELOPMENT.md` and the `sla_rules` seed.

Aggregate status is `in_progress`, `completed`, or `returned`. `CurrentStage` is the lowest presentation stage containing an applicable unfinished node; clients must use `available_actions` to render actionable work.

## 5. Evidence and audit

Every completion endpoint validates its required evidence server-side. Evidence attaches to a stable workflow-node code, with the display activity number retained as metadata where applicable. Each transition, branch decision, skip, terminal return, and completion is written to `ActivityLog` in the same transaction as the state change.

## 6. Hifi lifecycle

The hifi forms and detail pages validate concepts and presentation only. They are not permanent backend contracts and can be retired when detailed production form specifications arrive. The two files under `hifi-colabora/workflow/` remain the continuously updated workflow reference and must be reviewed at the start of every workflow-related task.
