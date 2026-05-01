# Dependency Graph

## Most Imported Files (change these carefully)

- `encoding/json` — imported by **71** files
- `net/http` — imported by **43** files
- `database/sql/driver` — imported by **21** files
- `math/rand` — imported by **6** files
- `net/url` — imported by **5** files
- `os/signal` — imported by **1** files
- `path/filepath` — imported by **1** files
- `net/http/httptest` — imported by **1** files
- `encoding/hex` — imported by **1** files

## Import Map (who imports what)

- `encoding/json` ← `e2e/collections_test.go`, `e2e/control_streams_test.go`, `e2e/datastreams_test.go`, `e2e/deployments_test.go`, `e2e/features_test.go` +66 more
- `net/http` ← `cmd/server/main.go`, `e2e/collections_test.go`, `e2e/control_streams_test.go`, `e2e/datastreams_test.go`, `e2e/deployments_test.go` +38 more
- `database/sql/driver` ← `internal/model/common_shared/capabilities.go`, `internal/model/common_shared/characteristics.go`, `internal/model/common_shared/codeList.go`, `internal/model/common_shared/configurationSettings.go`, `internal/model/common_shared/contacts.go` +16 more
- `math/rand` ← `internal/model/generators/generators_common_shared.go`, `internal/model/generators/generators_datastream.go`, `internal/model/generators/generators_deployment.go`, `internal/model/generators/generators_procedure.go`, `internal/model/generators/generators_sensorml_shared.go` +1 more
- `net/url` ← `internal/model/formaters/association_links.go`, `internal/model/formaters/formatter.go`, `internal/model/formaters/multi_format_serializer.go`, `internal/model/query_params/query_params.go`, `internal/model/query_params/query_params_test.go`
- `os/signal` ← `cmd/server/main.go`
- `path/filepath` ← `e2e/schema_validator.go`
- `net/http/httptest` ← `e2e/setup_test.go`
- `encoding/hex` ← `internal/model/common_shared/go_geom.go`
