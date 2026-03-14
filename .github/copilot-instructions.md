# Copilot / AI Agent Instructions — connected-systems-go

This short guide helps AI coding agents be productive in this repo. It focuses on the concrete, discoverable patterns, project layout, developer workflows, and examples the codebase uses.

## Big picture
- Entrypoint: `cmd/server/main.go` — loads config, opens GORM DB, runs `repository.AutoMigrate`, constructs repositories (`repository.NewRepositories`) and builds the Chi router via `api.NewRouter`.
- HTTP layer: `internal/api/router.go` — registers routes and builds the formatter collections (GeoJSON / SensorML). Handlers live alongside route builders in `internal/api`.
- Domain models: `internal/model/domains/` — GORM models for System, Deployment, Procedure, SamplingFeature, Property, Feature, Collection.
- Persistence: `internal/repository` — repository types wrap GORM and are instantiated by `NewRepositories`. `AutoMigrate` lists the models to migrate.
- Serialization: `internal/model/formaters/geojson_formatters` and `.../sensorml_formatters` — formatters implement serialization/deserialization for media types. Router registers formatters via `serializers.RegisterFormatterTyped` and sets a default with `RegisterFormatterTypedDefault`.

## Quickstart (how to run)
- Install deps: `go mod download`
- Copy config: `cp config.example.yaml config.yaml` (config uses `internal/config.Load()` — Viper-based; env vars override file values)
- DB: PostgreSQL + PostGIS required. Development helpers in repo include `docker-compose.yml` and `Makefile` targets referenced in README.
- Migrate & run:
  - `make migrate` (runs DB migrations)
  - `make run` (starts the server)
- Lint & build: `make lint`, `make build`

## Testing

Notes: most repository and end-to-end tests use Testcontainers to spin up a PostGIS database. Ensure Docker is running locally before running tests.

- Run all tests (unit + integration):
```bash
make test
# or
go test -v ./...
```

- Run tests with coverage and open report:
```bash
make test-coverage
# or
go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

- Run only repository tests (faster when iterating):
```bash
go test ./internal/repository -v
```

- Run a single test or package (useful during development):
```bash
# single test by name
go test ./internal/repository -run TestSystemRepository_Create -v

# single package
go test ./internal/api -v
```

- Integration / E2E tests: the `e2e` package (`e2e/setup_test.go`) also uses Testcontainers and an in-process HTTP server. Run them the same way as other tests; they will start a PostGIS container automatically.

Troubleshooting & tips:
- If tests fail due to inability to start containers, verify Docker daemon is running and your user can access it.
- To enable SQL logging for debugging, open the test file (e.g. `internal/repository/system_repository_test.go`) and in the `setupTestDB` call set `EnableLogging: true` when calling `testutil.OpenTestDB`.
- Tests use the `testutil` helpers in `internal/repository/testutil/postgis.go` which control the PostGIS image and migration order. Adjust migrations by changing the `Models` passed to `OpenTestDB` if you add new domain types.

## E2E Tests — How to build them

- **Canonical Example**: [e2e/procedures_test.go](e2e/procedures_test.go) demonstrates the preferred structure and patterns.
- **Structure**: write small, independent, top-level test functions for each abstract operation (Create, List, Get, Update, Delete). Example names: `TestDeployment_Create`, `TestDeployment_Get`.
- **Isolation**: each test should call `cleanupDB(t)` (or equivalent) at start, create its own fixtures, and not rely on order of execution.
- **Schema Validation**: centralize JSON/schema checks in helpers (see [e2e/test_helpers.go](e2e/test_helpers.go)). Use `GetSchemaValidator` and `validateAgainstSchema` to keep assertions consistent.
- **Fixtures & Helpers**: prefer small inline payload builders in tests and share only generic helpers (HTTP client, schema validator, geometry makers) via `e2e/test_helpers.go` or `internal/repository/testutil`.
- **Testcontainers & PostGIS**: tests use Testcontainers to start PostGIS automatically; ensure Docker is running locally before running E2E tests.
- **Run commands**: run e2e tests (zsh):

```zsh
# Ensure Docker is available
docker info

# Run only e2e tests
go test ./e2e -v

# Run a single e2e test
go test ./e2e -run TestDeployment_Create -v
```

- **Failure diagnosis**: when a test fails, run it in verbose mode and enable SQL logging by setting `EnableLogging: true` in the `setupTestDB`/`testutil.OpenTestDB` call in the test (or temporarily edit the test to enable logging).
- **Purposeful design**: aim for tests that exercise one API behaviour each, keep them fast(er) by limiting fixture size, and prefer asserting HTTP status + important fields over full-body equality (use schema validation helpers for payload contract checks).


## Conventions & notable patterns (do not change without reason)
- Router/Formatter pattern: `api.NewRouter` builds a `MultiFormatFormatterCollection<T>` per resource type and registers formatters for `application/geo+json` and `application/sml+json`. When adding new formats or formatters, follow the same registration pattern and set the default explicitly.
  - Example: `geojson_formatters.NewProcedureGeoJSONFormatter(repos)` then `serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)`.
- Default media types: GeoJSON is the default for most feature types; SensorML (`application/sml+json`) is the default for `Property` resources (see `buildPropertyFormatterCollection`). Keep defaults consistent with OGC mappings.
- DB model changes: update `internal/model/domains/*`, then add the type to `repository.AutoMigrate` and run `make migrate` / migrations. Repositories are provided by `internal/repository` and exposed via `Repositories` struct.
- Request/response format: Handlers accept GeoJSON Feature-like objects; the formatters translate between domain models and the HTTP payloads. Look at `internal/model/formaters/geojson_formatters/*` for examples of property mapping and `ProcedureGeoJSON*` types.
- Configuration: `internal/config/config.go` uses Viper; prefer `config.Load()` to obtain values and rely on env overrides when running CI or containers.

## How to add common things
- Add a new repository: implement `NewXRepository(db *gorm.DB)` under `internal/repository`, add it to the `Repositories` struct, include the model in `AutoMigrate`, and inject it into `api.NewRouter`/handlers.
- Add a new formatter/encoding: implement formatter in `internal/model/formaters/*`, register it in the appropriate `build*FormatterCollection` function in `internal/api/router.go`, and set the default if required.
- Add an API route/handler: create handler in `internal/api/*_handler.go`, then register routes in `router.go` following the canonical endpoints structure.

## Important files to inspect when making changes
- `cmd/server/main.go` — app lifecycle (logger, config, DB, router)
- `internal/api/router.go` — route registration and formatter wiring
- `internal/model/domains/` — domain models and constants (e.g. procedure types)
- `internal/repository/` — data access layer and migrations
- `internal/model/formaters/` — serialization logic for GeoJSON and SensorML
- `internal/config/config.go` — configuration loading (Viper)
- `Makefile`, `docker-compose.yml`, `migrations/` — development/test operations

## Gotchas & expectations for generated code
- OpenAPI generation: `getOpenAPISpec` in `internal/api/router.go` is a TODO — do not assume a full OpenAPI generator is present.
- Geometry handling: code uses a `common_shared.GoGeom` wrapper for spatial types; formatters expect geometry conversions (GeoJSON ↔ database). Reuse existing helpers in `internal/model/common_shared` rather than inventing new conversions.
- Procedure types & constants: some SOSA/SSN URIs are provided as constants in domain types (e.g. `internal/model/domains/procedure.go`). Prefer using those constants when setting `FeatureType` / `procedureType` values.

## Examples (short snippets)
- Register a GeoJSON formatter for a resource (see `router.go`):
```go
geoJSONFormatter := geojson_formatters.NewProcedureGeoJSONFormatter(repos)
serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)
serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")
```

- Create repositories and inject into router (from `cmd/server/main.go`):
```go
db, _ := gorm.Open(...)
if err := repository.AutoMigrate(db); err != nil { ... }
repos := repository.NewRepositories(db)
router := api.NewRouter(cfg, logger, repos)
```

## What to ask the maintainers (if unclear)
- Preferred DB migration tooling and CI steps for running integration tests.
- Any non-obvious conventions for `FeatureType` values or controlled vocabularies beyond the constants in `internal/model/domains`.

## Next steps / feedback
If this file misses specifics you want included (CI commands, exact Makefile targets, or preferred testing DB setup), tell me what to add and I will iterate.
