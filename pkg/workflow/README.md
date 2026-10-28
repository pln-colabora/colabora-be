# Workflow engine

This package is the pure domain engine used by the persisted workflow and HTTP
activity services. Definitions are documented in `docs/architecture/diagrams/`
and `docs/WORKFLOW_UPDATES_2026-10.md`. It imports only the standard library.

`Evaluate(Snapshot)` validates completed/in-progress history and derives all 20
active node states, aggregate status, the lowest unfinished stage, and actionable codes
in deterministic topological order. Missing nodes start locked. Persisted
available/locked values are projections and cannot override prerequisites. The
retired `pk_vendor` and `wo_app` codes are accepted only as historical input and
are never actionable.
Skipped values require a completed branch decision or the migrated priority
exemption for requests that already passed NPS.
Unknown connection types, nodes, statuses, missing completion decisions, and
impossible histories return `ErrInvalidState`.

`Transition(snapshot, code, InProgress|Completed)` returns a detached snapshot and
evaluation, leaving its input unchanged. Set decision values in the snapshot
before completing `kebutuhan_tiang`, `wo_konstruksi`, or `nps_delegation`.
Decisions take effect only when their owning node completes. A service can compose
ordered transitions for bundled endpoints whose nodes share one owner (RAB plus
pole decision, NPS submission, or closing) and persist them in one transaction.
Material reservation and APP tera are separate transitions because their owners differ.
After delegation, material reservation follows the construction WO, while APP tera
follows NPS delegation directly; there is no active APP WO. Construction execution requires the construction WO, completed
material reservation, and either completed WO PDKB or an explicit PDKB skip when
`perlu_pdkb=false`.
The service must prohibit revisions of completed decisions and validate evidence
and structured payloads. This package cannot validate those against a database.

Authorization lives in `pkg/rbac`: `OwnsWorkflowNode` checks role and ULP scope;
`AuthorizeWorkflowNode` also evaluates workflow state. `AvailableActions` filters
actionable codes by caller and returns definition metadata. HTTP routing and
bundled endpoint presentation belong to the service/DTO adapters.

On return, completed history and legitimate skips remain visible, downstream work
is locked, actions are empty, and the stage is 3. Completed requests use stage 7
with no actions. SLA references are nullable metadata; this package does not
calculate deadlines or infer rules for supporting nodes.

Priority verification belongs to Perencanaan for every connection type and gates
NPS expansion after RAB and the pole decision. SR/APP belongs to Transaksi Energi
and requires Tera only; closing joins SR/APP and network energize. Temporary and
permanent energize are service payload substatuses, not aggregate states.

Services implement PDKB date booking and completed survey/construction-WO evidence
revisions without reopening workflow nodes or changing completion/SLA facts.
Payment-based working-day deadlines live outside this evaluator, in
`pkg/businessday` and the persistence/service adapters.
