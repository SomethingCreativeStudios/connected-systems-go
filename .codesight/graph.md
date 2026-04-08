# Dependency Graph

## Most Imported Files (change these carefully)

- `encoding/json` — imported by **72** files
- `net/http` — imported by **44** files
- `database/sql/driver` — imported by **21** files
- `cs-api-viewer/src/schema-components/utils.ts` — imported by **12** files
- `cs-api-client/src/types/common.ts` — imported by **10** files
- `cs-api-viewer/src/app/types.ts` — imported by **9** files
- `cs-api-client/src/codecs/wire-types.ts` — imported by **7** files
- `cs-api-client/src/types/resources.ts` — imported by **7** files
- `net/url` — imported by **6** files
- `math/rand` — imported by **6** files
- `cs-api-client/src/codecs/utils.ts` — imported by **5** files
- `cs-api-viewer/src/schema-components/schema-context.ts` — imported by **5** files
- `cs-api-viewer/src/app/shared.ts` — imported by **5** files
- `cs-api-viewer/src/app/constants.ts` — imported by **4** files
- `cs-api-client/src/content-types.ts` — imported by **3** files
- `cs-api-viewer/src/schema-components/geometry-editor/geometry-editor.vue` — imported by **3** files
- `cs-api-viewer/src/schema-components/types.ts` — imported by **3** files
- `os/signal` — imported by **2** files
- `cs-api-client/src/http.ts` — imported by **2** files
- `cs-api-client/src/errors.ts` — imported by **2** files

## Import Map (who imports what)

- `encoding/json` ← `e2e/collections_test.go`, `e2e/control_streams_test.go`, `e2e/datastreams_test.go`, `e2e/deployments_test.go`, `e2e/features_test.go` +67 more
- `net/http` ← `cmd/server/main.go`, `e2e/collections_test.go`, `e2e/control_streams_test.go`, `e2e/datastreams_test.go`, `e2e/deployments_test.go` +39 more
- `database/sql/driver` ← `internal/model/common_shared/capabilities.go`, `internal/model/common_shared/characteristics.go`, `internal/model/common_shared/codeList.go`, `internal/model/common_shared/configurationSettings.go`, `internal/model/common_shared/contacts.go` +16 more
- `cs-api-viewer/src/schema-components/utils.ts` ← `cs-api-viewer/src/app/use-association-graph.ts`, `cs-api-viewer/src/schema-components/fields/characteristic-capability-field.tsx`, `cs-api-viewer/src/schema-components/fields/component-field.tsx`, `cs-api-viewer/src/schema-components/fields/constraints-field.tsx`, `cs-api-viewer/src/schema-components/fields/contacts-field.tsx` +7 more
- `cs-api-client/src/types/common.ts` ← `cs-api-client/src/client.ts`, `cs-api-client/src/codecs/deployment.ts`, `cs-api-client/src/codecs/procedure.ts`, `cs-api-client/src/codecs/property.ts`, `cs-api-client/src/codecs/sampling-feature.ts` +5 more
- `cs-api-viewer/src/app/types.ts` ← `cs-api-viewer/src/app/association-helpers.ts`, `cs-api-viewer/src/app/collection-data.ts`, `cs-api-viewer/src/app/constants.ts`, `cs-api-viewer/src/app/schema-summary.ts`, `cs-api-viewer/src/app/url-state.ts` +4 more
- `cs-api-client/src/codecs/wire-types.ts` ← `cs-api-client/src/client.ts`, `cs-api-client/src/codecs/deployment.ts`, `cs-api-client/src/codecs/index.ts`, `cs-api-client/src/codecs/procedure.ts`, `cs-api-client/src/codecs/property.ts` +2 more
- `cs-api-client/src/types/resources.ts` ← `cs-api-client/src/codecs/deployment.ts`, `cs-api-client/src/codecs/procedure.ts`, `cs-api-client/src/codecs/property.ts`, `cs-api-client/src/codecs/sampling-feature.ts`, `cs-api-client/src/codecs/system.ts` +2 more
- `net/url` ← `internal/model/formaters/association_links.go`, `internal/model/formaters/formatter.go`, `internal/model/formaters/multi_format_serializer.go`, `internal/model/query_params/query_params.go`, `internal/model/query_params/query_params_test.go` +1 more
- `math/rand` ← `internal/model/generators/generators_common_shared.go`, `internal/model/generators/generators_datastream.go`, `internal/model/generators/generators_deployment.go`, `internal/model/generators/generators_procedure.go`, `internal/model/generators/generators_sensorml_shared.go` +1 more
