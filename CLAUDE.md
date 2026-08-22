# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Go backend built on the `go-gin-clean-starter` template (Gin + GORM + `samber/do` DI), implementing Controller → Service → Repository clean architecture with feature modules under `modules/`. Note: `hifi-colabora/` is a separate nested git repository (a static HTML high-fidelity prototype/mockup set) — it is not part of this Go codebase and has its own `CLAUDE.md`.

**This backend is being built to implement the COLABORA workflow tracked by that mockup.** Before starting feature work here, read the planning docs at repo root: [`PRD.md`](./PRD.md) (product spec, roles, 17-activity process, decision branches), [`DATA_MODEL.md`](./DATA_MODEL.md) (proposed entities), [`API_SPEC.md`](./API_SPEC.md) (endpoint-per-screen mapping), [`RBAC.md`](./RBAC.md) (authorization model, ported from `hifi-colabora/assets/rbac.js`), and [`ROADMAP.md`](./ROADMAP.md) (phased build order). These are living design docs, not yet-implemented fact — check the actual code state before assuming a phase is done.

## Commands

### Setup & running
```bash
make dep                # go mod tidy
make run                # go run cmd/main.go
make build               # build binary to ./main
make run-build           # build then run
```
No Docker: configure `.env` (DB_HOST/DB_USER/DB_PASS/DB_NAME/DB_PORT, etc.) then `make migrate-seed && make run`.
With Docker: `make init-docker` then `make migrate-seed-docker`.

### Tests
```bash
make test                                          # go test -v ./tests
make test-auth                                     # ./modules/auth/tests/...
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
- `query/` — filter/includable structs consumed by `github.com/Caknoooo/go-pagination` for list endpoints (see `modules/user/controller/user_controller.go` `GetAllUser`).
- `tests/` — one `_test.go` per layer, table-style with `testify/assert`.
- `routes.go` — `RegisterRoutes(server *gin.Engine, injector *do.Injector)`, resolves controllers/services via `do.MustInvoke`/`do.MustInvokeNamed`, groups routes under `/api/<name>`, applies `middlewares.Authenticate(jwtService)` per-route where auth is required.

`auth` and `user` modules are cross-wired: `authService` depends on `user`'s repository/DTOs directly (see `modules/auth/service/auth_service.go`), and `middlewares.Authenticate` depends on `auth/service.JWTService` and `user/dto` message constants. When adding a new module that needs the current user, follow the same pattern as `user_controller.go`'s `Me`: read `user_id` set by the auth middleware via `ctx.MustGet("user_id")`.

### Dependency injection (`samber/do`)
All singletons (DB connection, JWTService, repositories, services, controllers) are wired in `providers/core.go`'s `RegisterDependencies`, invoked once from `cmd/main.go`. Named bindings (`constants.DB`, `constants.JWTService`) are resolved with `do.MustInvokeNamed`; everything else uses plain `do.Provide`/`do.MustInvoke`. When adding a new module's dependencies, extend `RegisterDependencies` in the same style (build repo → build service → `do.Provide` the controller) rather than constructing dependencies inline elsewhere.

### Entry point flow (`cmd/main.go`)
1. Build the `do.Injector` and call `providers.RegisterDependencies`.
2. `args(injector)` parses CLI flags via `script.Commands`; if it returns false, the process exits without starting the server (used for one-off migrate/seed/script runs).
3. Otherwise build the Gin engine, apply `middlewares.CORSMiddleware()`, call each module's `RegisterRoutes`, then `run(server)` (serves static `/assets`, reads `GOLANG_PORT` env, binds `0.0.0.0` when `APP_ENV=localhost` else `:port`).
New modules must be added to both `providers.RegisterDependencies` (DI wiring) and `cmd/main.go` (`RegisterRoutes` call).

### Database layer
- `database/entities/` — GORM models; shared fields (`Timestamp`, `Authorization`) live in `entities/common.go`.
- `database/migration.go` — `Migrate(db)` runs `AutoMigrate` for all entities first, then runs the batch migration manager.
- `database/migrations/` — individual files register themselves via `init()` calling `database.RegisterMigration(name, upFn, downFn)`; naming convention `YYYYMMDDHHMMSS_description.go`. These are blank-imported (`_ "…/database/migrations"`) from `script/command.go` so `init()` runs.
- `database/manager.go` — the migration manager (batching, status, rollback, `Create` scaffolding).
- `database/seeders/` — JSON fixtures (`seeders/json/*.json`) loaded by seed functions in `seeders/seeds/`; `database/seeder.go` wires them together.

### Auth & middleware
JWT-based auth with access + refresh tokens (`modules/auth/service/jwt_service.go`, `auth_service.go`); refresh tokens persisted via `modules/auth/repository/refresh_token_repository.go`. `middlewares.Authenticate` expects `Authorization: Bearer <token>`, validates it, and sets `user_id`/`token` in the Gin context for downstream handlers. Role constants (`ENUM_ROLE_ADMIN`/`ENUM_ROLE_USER`) and shared config live in `pkg/constants/common.go`.

### Response & helper conventions
- Standard JSON envelope: `pkg/utils.BuildResponseSuccess(message, data)` / `BuildResponseFailed(message, errDetail, data)` — use these instead of hand-rolled response maps.
- Password hashing in `pkg/helpers/password.go`; AES helpers and email sending (with `pkg/utils/email-template/base_mail.html`) in `pkg/utils/`.
- Pagination for list endpoints uses `github.com/Caknoooo/go-pagination` (`PaginatedQueryWithIncludable`, `CalculatePagination`, `NewPaginatedResponse`), driven by a per-module `query.<Entity>Filter` that embeds pagination binding (`filter.BindPagination(ctx)`).

### Logs UI
Built-in query log viewer at `/logs` (served from `logs.html`), reading from `config/logs/query_log/`; filters by month, expandable entries.

### API docs (Scalar)
`GET /docs` serves a [Scalar](https://scalar.com/) API Reference UI (registered in `docs/routes.go`, wired in `cmd/main.go` alongside the module routes — no DI/DB dependency). One OpenAPI document per module — `docs/health.yaml`, `docs/auth.yaml`, `docs/user.yaml` — each self-contained (own `info`/`components`, not sharing `$ref`s across files) and served at `GET /docs/<module>.yaml`; the Scalar page's `sources` config lets you switch between them, and `hideModels: true` hides the schema-browser sidebar section (Scalar still resolves `$ref`s internally to render each endpoint's body). These files only document endpoints that actually exist in code today; when a new module/endpoint is added (e.g. `permohonan`), add its own `docs/<module>.yaml` plus one `server.StaticFile` line and one `sources` entry in `docs/routes.go` in the same change — don't let it drift, and don't lump it into an existing module's file. The Scalar CDN `<script>` is pinned to a specific version with a matching SRI `integrity` hash (not the unversioned `latest` tag) — bumping the Scalar version means updating the version in the URL and recomputing the hash together (see the comment in `docs/routes.go` for the exact command).
