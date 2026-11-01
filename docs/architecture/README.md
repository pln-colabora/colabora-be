# COLABORA Architecture Diagrams

These diagrams describe the implemented `dev` backend state verified on 26 October 2026.
They describe repository behavior; deployment of the matching backend and migrations
is required before assuming that an online service supports this flow.
Workflow-node codes are the canonical transition and authorization identities;
activity numbers and stages are display metadata.

When the application is running, open [`/docs/architecture`](http://localhost:8888/docs/architecture)
using the configured `GOLANG_PORT`. GitHub and Mermaid-capable Markdown previews can
render each source file directly.

| Diagram | Perspective |
|---|---|
| [Workflow JTR/JTM](diagrams/01-workflow-jtr-jtm.md) | Canonical dependency graph and JTR/JTM ownership |
| [Workflow PLG TM](diagrams/02-workflow-plg-tm.md) | Canonical dependency graph and PLG TM ownership |
| [Workflow state](diagrams/03-workflow-state.md) | Node and aggregate state lifecycles |
| [RBAC ownership](diagrams/04-rbac-ownership.md) | Write ownership and read scopes |
| [Activity submission](diagrams/05-activity-submission.md) | Atomic write endpoint sequence and rollback |
| [Data model](diagrams/06-data-model.md) | Persisted entity relationships |
| [Components](diagrams/07-components.md) | Runtime modules and infrastructure boundaries |
| [API action map](diagrams/08-api-action-map.md) | Write endpoints, nodes, owners, and evidence |
| [Evidence lifecycle](diagrams/09-evidence-lifecycle.md) | Upload, scan, revision, attachment, and cleanup |
| [SLA and critical path](diagrams/10-sla-critical-path.md) | Dependency path and independent SLA references |

The current workflow includes Perencanaan priority verification before NPS,
payment-based working-day SLA, survey/construction-WO evidence revisions, atomic
PDKB date booking, Transaksi Energi ownership of SR/APP, and temporary/permanent
network energize. WO APP is historical only. See
[October workflow decisions](../WORKFLOW_UPDATES_2026-10.md) for agreed details.

Keep the two workflow diagrams synchronized with `pkg/workflow/definition.go`,
`pkg/rbac`, activity routes/services, and the frontend `colabora-flow-end-to-end.md`.
The earlier `hifi-colabora/workflow/` prototype is absent from this checkout;
it was not modified or treated as current evidence. Runtime routes and entities
take precedence for API and persistence diagrams.
