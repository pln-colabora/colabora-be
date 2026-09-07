# AGENTS.md

This file provides guidance to coding agents working in this repository.

## Project overview

Go backend built on the `go-gin-clean-starter` template (Gin + GORM + `samber/do` DI), implementing Controller → Service → Repository clean architecture with feature modules under `modules/`. Note: `hifi-colabora/` is a separate nested git repository (a static HTML high-fidelity prototype/mockup set) — it is not part of this Go codebase and has its own `CLAUDE.md`.

**This backend implements the COLABORA workflow tracked by the living diagrams under `hifi-colabora/workflow/`.** Before starting feature work, read [`PRD.md`](./PRD.md), [`DATA_MODEL.md`](./DATA_MODEL.md), [`API_SPEC.md`](./API_SPEC.md), [`RBAC.md`](./RBAC.md), and [`ROADMAP.md`](./ROADMAP.md). The design documents describe the target refactor; `docs/*.yaml` and the code describe what is implemented today.

### Workflow source and nested-repository checks

- Before workflow work, compare the `hifi-colabora` commit checked out in the working tree with the gitlink recorded by this repository (`git ls-tree HEAD hifi-colabora` and `git -C hifi-colabora log`). Review the intervening commits rather than assuming the parent repository's docs already include them.
- Treat `hifi-colabora/workflow/jtr-jtm.html` and `hifi-colabora/workflow/plg-tm.html` as the continuously updated source of truth for order, dependencies, branches, and ownership. Read both at the beginning of every workflow-related task.
- Use `hifi-colabora/DEVELOPMENT.md` for supporting domain/SLA detail. Forms and detail pages are illustrative, may lag, and may be retired when detailed production forms arrive. Never infer backend order from form filenames or demo navigation.
- Activity numbers are labels from the source process, not a guarantee of simple sequential execution. Stages 4–6 contain conditional and parallel work. Do not advance by incrementing an activity number; evaluate the applicable prerequisites and completion gates.
- Keep connection-type ownership explicit: JTR/JTM and PLG TM can have different owners for the same numbered activity. Model authorization per workflow node, not per stage alone.

## Commands

### Setup & running

```bash
make dep                 # go mod tidy
make run                 # go run cmd/main.go
make build               # build binary to ./main
make run-build           # build then run
```

No Docker: configure `.env` (DB_HOST/DB_USER/DB_PASS/DB_NAME/DB_PORT, etc.) then `make migrate-seed && make run`.
With Docker: `make init-docker` then `make migrate-seed-docker`.

### Tests

```bash
make test                                           # go test -v ./tests
make test-auth                                      # ./modules/auth/tests/...
make test-user                                      # ./modules/user/tests/...
make test-all                                       # go test -v ./modules/.../tests/...
make test-coverage                                  # coverage.out + HTML report
go test -v ./modules/user/tests/... -run TestName   # single test
```

### Migrations (Laravel-style, batch-based)

```bash
make migrate                                # run pending migrations
make migrate-status                         # show status
make migrate-rollback                       # rollback last batch
make migrate-rollback-batch batch=<n>       # rollback a specific batch
make migrate-rollback-all                   # rollback everything
make migrate-create name=create_x_table     # scaffold a new migration
```

Equivalent `-docker` targets exist for all of the above (run inside the app container).
Note: `make migrate-create name=create_posts_table` (format `create_*_table`) auto-generates the entity in `database/entities/`, adds it to the new migration file, and appends it to the `AutoMigrate` list in `database/migration.go` — do the same three edits by hand if adding an entity/table manually.

### Module scaffolding

```bash
make module name=product   # or: ./create_module.sh product
```

Generates `modules/product/{controller,service,repository,dto,validation,tests,query}` with boilerplate wired for DI, plus `routes.go`. After generating, register the module's `RegisterRoutes` call in `cmd/main.go` and wire its providers into `providers/core.go`.

### Ad-hoc scripts / combined CLI

```bash
go run cmd/main.go --migrate:run --seed --run --script:example_script
```

Flags: `--migrate`/`--migrate:run`, `--migrate:status`, `--migrate:rollback [batch]`, `--migrate:rollback:all`, `--migrate:create:<name>`, `--seed`, `--script:<name>` (defined in `script/script.go`), `--run` (keep server alive after the above). Any of these flags short-circuits normal server startup unless `--run` is also passed (see `args()` in `cmd/main.go`).

## Architecture

### Module layout

Each feature lives in `modules/<name>/` with a fixed shape:

- `controller/` — parses/validates HTTP request, calls service, shapes `utils.BuildResponseSuccess`/`BuildResponseFailed` JSON.
- `service/` — business logic, orchestrates repositories, takes `context.Context`.
- `repository/` — GORM queries, takes `*gorm.DB` (or a passed-in tx) explicitly per call rather than storing it once, so callers can pass a transaction.
- `dto/` — request/response structs plus `MESSAGE_*` string constants and sentinel `Err*` errors for that module.
- `validation/` — wraps `go-playground/validator` for request-level checks beyond struct tags.
- `query/` — filter/includable structs consumed by `github.com/Caknoooo/go-pagination` for list endpoints.
- `tests/` — one `_test.go` per layer, table-style with `testify/assert`.
- `routes.go` — registers `/api/<name>` routes and applies authentication per route where required.

`auth` and `user` are cross-wired: `authService` depends on `user` repository/DTO types, and `middlewares.Authenticate` depends on the JWT service and user message constants. For the current user, read the `user_id` set by auth middleware via `ctx.MustGet("user_id")`.

### Dependency injection (`samber/do`)

All singletons (DB connection, JWT service, repositories, services, controllers) are wired in `providers/core.go`'s `RegisterDependencies`, invoked once from `cmd/main.go`. Named bindings use `do.MustInvokeNamed`; other bindings use `do.Provide`/`do.MustInvoke`. Add module dependencies there rather than constructing them inline elsewhere.

### Entry point flow (`cmd/main.go`)

1. Build the `do.Injector` and call `providers.RegisterDependencies`.
2. `args(injector)` parses CLI flags; if it returns false, the process exits without starting the server.
3. Otherwise build Gin, apply CORS, register each module, and run the server.

New modules must be added to both `providers.RegisterDependencies` and `cmd/main.go`.

### Database layer

- `database/entities/` — GORM models; shared fields live in `entities/common.go`.
- `database/migration.go` — runs `AutoMigrate` for all entities, then the batch migration manager.
- `database/migrations/` — individual files self-register through `init()` and `database.RegisterMigration`.
- `database/manager.go` — migration batching, status, rollback, and scaffolding.
- `database/seeders/` — JSON fixtures and seed functions.

### Auth & middleware

JWT auth uses access and refresh tokens; refresh tokens are persisted. `middlewares.Authenticate` expects `Authorization: Bearer <token>`, validates it, and sets `user_id`/`token` in Gin context. COLABORA functional-role constants and activity ownership live in `pkg/rbac`; keep authorization logic centralized there.

### Response & helper conventions

- Use `pkg/utils.BuildResponseSuccess` / `BuildResponseFailed` rather than hand-written envelopes.
- Pagination uses `github.com/Caknoooo/go-pagination` and per-module filters.
- Services own business transactions and state transitions; repositories perform persistence using the DB/transaction passed by the caller.

### API docs (Scalar)

`GET /docs` serves Scalar API Reference. Keep one self-contained OpenAPI document per module and register it in both `docs/routes.go` and `docs/openapi_merge.go`. Runtime OpenAPI describes implemented code; future contracts stay in `API_SPEC.md` until they land. The Scalar CDN script is version-pinned with matching SRI; update the URL and hash together.

## Working-tree safety

The root repository and `hifi-colabora/` have separate Git histories. Preserve unrelated user changes in both. Do not commit, reset, checkout, or move the nested repository unless explicitly asked. A changed `hifi-colabora` gitlink can represent an intentional prototype update, not an ordinary modified directory.
