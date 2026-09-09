# COLABORA Architecture Diagrams

These diagrams describe the implemented backend state verified on 9 September 2026.
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

The two workflow diagrams must stay synchronized with the living diagrams under
`hifi-colabora/workflow/` and with `pkg/workflow`. Runtime routes and entities take
precedence for diagrams that describe API and persistence behavior.
