# COLABORA Backend

Go backend for PLN's COLABORA workflow tracker for Permohonan PB/PD. The service uses Gin, GORM/PostgreSQL, `samber/do` dependency injection, JWT authentication, Garage S3-compatible document storage, and Controller → Service → Repository feature modules.

## Workflow source of truth

Always inspect these living workflow diagrams before changing process behavior:

- [`hifi-colabora/workflow/jtr-jtm.html`](./hifi-colabora/workflow/jtr-jtm.html)
- [`hifi-colabora/workflow/plg-tm.html`](./hifi-colabora/workflow/plg-tm.html)

They define current order, dependencies, branches, and ownership. The rest of the hifi prototype is illustrative and may be retired when detailed production forms are available. Backend contracts must not depend on hifi filenames or hardcoded demo state.

The target backend workflow is a dependency graph: 17 numbered business activities plus supporting/decision nodes, grouped into 7 presentation stages. Stages 4–6 contain conditional and parallel branches, so progress cannot be implemented as a simple activity/stage increment.

## Implementation status

Implemented:

- JWT access/refresh authentication and user module;
- SLA rule entity and seed data;
- permohonan create/list/detail using the current pre-refactor stage model;
- initial RBAC helpers and middleware;
- activity log and document entities;
- private Garage-backed document upload/download module;
- Scalar/OpenAPI documentation for implemented endpoints.

The canonical development plan is [`DEVELOPMENT.md`](./DEVELOPMENT.md). Phase 1's pure workflow engine and exact-node authorization are implemented and tested. Next is workflow-node persistence, followed by service integration and activity endpoints in dependency order. Existing HTTP endpoints still use the legacy stage model. Breaking development schema/API changes are allowed.

## Documentation map

- [`PRD.md`](./PRD.md) — product behavior and canonical backend workflow dependencies.
- [`DATA_MODEL.md`](./DATA_MODEL.md) — target workflow-node persistence model.
- [`API_SPEC.md`](./API_SPEC.md) — target REST contract.
- [`RBAC.md`](./RBAC.md) — target per-node authorization.
- [`DEVELOPMENT.md`](./DEVELOPMENT.md) — canonical implementation status, phases, exit gates, and SLA reference.
- [`ROADMAP.md`](./ROADMAP.md) — concise phase overview linking to the canonical plan.
- [`PHASE5_DOCUMENTS.md`](./PHASE5_DOCUMENTS.md) — implemented evidence-storage design and known view-access gap.
- [`DOCUMENT_WORKFLOW_REFERENCE.md`](./DOCUMENT_WORKFLOW_REFERENCE.md) — findings from real operational samples and guidance for future activity forms/document modeling.
- [`DEPLOY.md`](./DEPLOY.md) — production deployment.
- [`AGENTS.md`](./AGENTS.md) / [`CLAUDE.md`](./CLAUDE.md) — repository guidance for coding agents.
- `docs/*.yaml` — runtime OpenAPI; these describe actual code rather than future design.

## Local development

Requirements: Go 1.20+, PostgreSQL 15+, and optionally Docker Compose.

```bash
cp .env.example .env
make dep
make migrate-seed
make run
```

With Docker:

```bash
make init-docker
make migrate-seed-docker
```

The API uses `GOLANG_PORT`; Scalar documentation is available at `/docs`.

## Common commands

```bash
make run
make build
make test-all
go test ./...

make migrate
make migrate-status
make migrate-rollback
make migrate-create name=create_x_table

make module name=product
```

Migration and seed commands can also be combined with the entrypoint CLI:

```bash
go run cmd/main.go --migrate:run --seed --run
```

## Architecture

Features live under `modules/<name>/` and normally contain controller, service, repository, DTO, validation, query, tests, and route registration. Dependencies are wired in `providers/core.go`; routes are registered from `cmd/main.go`. Services own transactions and workflow rules, while repositories accept the caller-provided `*gorm.DB` or transaction.

Use `pkg/utils.BuildResponseSuccess`/`BuildResponseFailed` for response envelopes. Keep workflow and role resolution centralized; do not duplicate ownership or transition checks in controllers.

The implemented workflow, RBAC, data model, API actions, evidence lifecycle, SLA,
and runtime components are visualized in the [architecture diagram catalog](./docs/architecture/README.md).
When the server is running, the same Mermaid sources are available at
`GET /docs/architecture`.

## Deployment

Production uses Docker Compose, GHCR, GitHub Actions, nginx, PostgreSQL, and Garage. See [`DEPLOY.md`](./DEPLOY.md) for setup, deployment, and rollback instructions.
