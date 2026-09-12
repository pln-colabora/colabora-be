# Workflow engine

This package is the pure Phase 1 domain engine. Definitions follow both living
diagrams in `hifi-colabora/workflow/`. It imports only the standard library.

`Evaluate(Snapshot)` validates completed/in-progress history and derives all 20
active node states, aggregate status, the lowest unfinished stage, and actionable codes
in deterministic topological order. Missing nodes start locked. Persisted
available/locked values are projections and cannot override prerequisites. The
retired `pk_vendor` code is accepted only as historical input and is never
actionable.
Skipped values are accepted only when backed by a completed branch decision.
Unknown connection types, nodes, statuses, missing completion decisions, and
impossible histories return `ErrInvalidState`.

`Transition(snapshot, code, InProgress|Completed)` returns a detached snapshot and
evaluation, leaving its input unchanged. Set decision values in the snapshot
before completing `kebutuhan_tiang`, `wo_konstruksi`, or `nps_delegation`.
Decisions take effect only when their owning node completes. A service can compose
ordered transitions for bundled endpoints (RAB plus pole decision, NPS submission,
reservation plus tera, or closing) and persist them in one transaction.
The service must prohibit revisions of completed decisions and validate evidence
and structured payloads. This package cannot validate those against a database.

Authorization lives in `pkg/rbac`: `OwnsWorkflowNode` checks role and ULP scope;
`AuthorizeWorkflowNode` also evaluates workflow state. `AvailableActions` filters
actionable codes by caller and returns definition metadata. HTTP routing and
bundled endpoint presentation belong to the future service/DTO adapters.

On return, completed history and legitimate skips remain visible, downstream work
is locked, actions are empty, and the stage is 3. Completed requests use stage 7
with no actions. SLA references are nullable metadata; this package does not
calculate deadlines or infer rules for supporting nodes.

The engine is not yet connected to legacy persistence, services, or HTTP routes.
