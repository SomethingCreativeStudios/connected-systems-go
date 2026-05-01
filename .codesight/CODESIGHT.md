# connected-systems-go — AI Context Map

> **Stack:** chi | gorm | unknown | go

> 155 routes | 15 models | 0 components | 113 lib files | 0 env vars | 0 middleware | 38% test coverage
> **Token savings:** this file is ~10,200 tokens. Without it, AI exploration would cost ~122,800 tokens. **Saves ~112,500 tokens per conversation.**
> **Last scanned:** 2026-05-01 03:12 — re-run after significant changes

---

# Routes

## CRUD Resources

- **`/collections/{collectionId}/items`** GET | POST | PUT/:id | DELETE/:id → Item
- **``** GET/:id | PUT/:id | DELETE/:id
- **`/systems`** GET | POST | PUT/:id | DELETE/:id → System
- **`/events`** GET | POST | GET/:id | PUT/:id | DELETE/:id → Event
- **`/history`** GET | GET/:id | PUT/:id | DELETE/:id → History
- **`/datastreams`** GET | POST | PUT/:id | DELETE/:id → Datastream
- **`/controlstreams`** GET | POST | PUT/:id | DELETE/:id → Controlstream
- **`/commands`** GET | POST | PUT/:id | DELETE/:id → Command
- **`/observations`** GET | POST | PUT/:id | DELETE/:id → Observation
- **`/deployments`** GET | POST | PUT/:id | DELETE/:id → Deployment
- **`/procedures`** GET | POST | PUT/:id | DELETE/:id → Procedure
- **`/samplingFeatures`** GET | POST | PUT/:id | DELETE/:id → SamplingFeature
- **`/properties`** GET | POST | PUT/:id | DELETE/:id → Propertie
- **`/`** GET | POST | PUT/:id | DELETE/:id
- **`/collections`** GET | POST | GET/:id → Collection

## Other Routes

- `GET` `Location` params()
- `GET` `Content-Type` params()
- `GET` `Accept` params()
- `GET` `cascade` params() [db]
- `GET` `content-type` params() [db] ✓
- `GET` `/systems/subsystems` params() [auth, db]
- `POST` `/systems/subsystems` params() [auth, db]
- `GET` `/systems/deployments` params() [auth, db]
- `GET` `/systems/procedures` params() [auth, db]
- `GET` `/systems/samplingFeatures` params() [auth, db]
- `GET` `/systems/datastreams` params() [auth, db]
- `GET` `/systems/controlstreams` params() [auth, db]
- `GET` `/systems/events` params() [auth, db]
- `POST` `/systems/events` params() [auth, db]
- `GET` `/systems/history` params() [auth, db]
- `POST` `/systems/samplingFeatures` params() [auth, db]
- `POST` `/systems/datastreams` params() [auth, db]
- `POST` `/systems/controlstreams` params() [auth, db]
- `GET` `/{id}/subsystems` params(id) [auth, db]
- `POST` `/{id}/subsystems` params(id) [auth, db]
- `GET` `/{id}/deployments` params(id) [auth, db]
- `GET` `/{id}/procedures` params(id) [auth, db]
- `GET` `/{id}/samplingFeatures` params(id) [auth, db]
- `GET` `/{id}/datastreams` params(id) [auth, db]
- `GET` `/{id}/controlstreams` params(id) [auth, db]
- `GET` `/{id}/events` params(id) [auth, db]
- `POST` `/{id}/events` params(id) [auth, db]
- `GET` `/{id}/history` params(id) [auth, db]
- `POST` `/{id}/samplingFeatures` params(id) [auth, db]
- `POST` `/{id}/datastreams` params(id) [auth, db]
- `POST` `/{id}/controlstreams` params(id) [auth, db]
- `GET` `/systemEvents` params() [auth, db]
- `GET` `/datastreams/schema` params() [auth, db]
- `PUT` `/datastreams/schema` params() [auth, db]
- `GET` `/datastreams/observations` params() [auth, db]
- `POST` `/datastreams/observations` params() [auth, db]
- `GET` `/{dataStreamId}/schema` params(dataStreamId) [auth, db]
- `PUT` `/{dataStreamId}/schema` params(dataStreamId) [auth, db]
- `GET` `/{dataStreamId}/observations` params(dataStreamId) [auth, db]
- `POST` `/{dataStreamId}/observations` params(dataStreamId) [auth, db]
- `GET` `/controlstreams/schema` params() [auth, db]
- `PUT` `/controlstreams/schema` params() [auth, db]
- `GET` `/controlstreams/commands` params() [auth, db]
- `POST` `/controlstreams/commands` params() [auth, db]
- `GET` `/{controlStreamId}/schema` params(controlStreamId) [auth, db]
- `PUT` `/{controlStreamId}/schema` params(controlStreamId) [auth, db]
- `GET` `/{controlStreamId}/commands` params(controlStreamId) [auth, db]
- `POST` `/{controlStreamId}/commands` params(controlStreamId) [auth, db]
- `GET` `/deployments/subdeployments` params() [auth, db]
- `POST` `/deployments/subdeployments` params() [auth, db]
- `GET` `/{id}/subdeployments` params(id) [auth, db]
- `POST` `/{id}/subdeployments` params(id) [auth, db]
- `GET` `/conformance` params() [auth, db]
- `GET` `/subsystems` params() [auth, db] ✓
- `POST` `/subsystems` params() [auth, db] ✓
- `GET` `/schema` params() [auth, db] ✓
- `PUT` `/schema` params() [auth, db] ✓
- `GET` `/subdeployments` params() [auth, db] ✓
- `POST` `/subdeployments` params() [auth, db] ✓
- `GET` `/api` params() [auth, db]
- `GET` `recursive` params() [db]
- `GET` `controlStream` params() [db]
- `GET` `system` params() [db]
- `GET` `foi` params() [db]
- `GET` `currentStatus` params() [db]
- `GET` `controlledProperty` params() [db]
- `GET` `observedProperty` params() [db]
- `GET` `parent` params() [db]
- `GET` `bbox` params() [db]
- `GET` `datetime` params() [db]
- `GET` `dataStream` params() [db]
- `GET` `baseProperty` params() [db]
- `GET` `objectType` params() [db]
- `GET` `limit` params() [db] ✓
- `GET` `offset` params() [db] ✓
- `GET` `id` params() [db] ✓
- `GET` `q` params() [db]
- `GET` `eventType` params() [db]
- `GET` `keyword` params() [db]
- `GET` `procedure` params() [db]
- `GET` `geom` params() [db]

---

# Schema

### Command
- ControlStreamID: string (required, index)
- SamplingFeatureID: *string (index)
- ProcedureLink: *common_shared.Link
- IssueTime: *time.Time (index)
- ExecutionTime: *common_shared.TimeRange
- Sender: string
- CurrentStatus: CommandStatus (default)
- Parameters: json.RawMessage

### Base
- ID: string (pk)
- CreatedAt: time.Time
- UpdatedAt: time.Time

### CommonSSN
- UniqueIdentifier: UniqueID (unique)
- Name: string (required)
- Description: string

### ControlStream
- ValidTime: *common_shared.TimeRange
- Formats: common_shared.StringArray
- SystemLink: *common_shared.Link
- InputName: string
- ProcedureLink: *common_shared.Link
- DeploymentLink: *common_shared.Link
- FeatureOfInterest: *common_shared.Link
- SamplingFeatureLink: *common_shared.Link
- IssueTime: *common_shared.TimeRange
- ExecutionTime: *common_shared.TimeRange
- Live: *bool
- Async: *bool
- Links: common_shared.Links
- SystemID: *string (index)
- ProcedureID: *string (index)
- DeploymentID: *string (index)
- FeatureOfInterestID: *string (index)
- SamplingFeatureID: *string (index)
- Systems: []System
- _relations_: ControlledProperties: ControlStreamControlledProperties, Schema: ControlStreamSchema

### Datastream
- Name: string (required)
- Description: string
- ValidTime: *common_shared.TimeRange
- Formats: common_shared.StringArray
- SystemLink: *common_shared.Link
- OutputName: string
- ProcedureLink: *common_shared.Link
- DeploymentLink: *common_shared.Link
- FeatureOfInterest: *common_shared.Link
- SamplingFeatureLink: *common_shared.Link
- PhenomenonTime: *common_shared.TimeRange
- PhenomenonTimeInterval: *string
- ResultTime: *common_shared.TimeRange
- ResultTimeInterval: *string
- Type: string
- ResultType: *string
- Live: *bool
- Links: common_shared.Links
- SystemID: *string (index)
- ProcedureID: *string (index)
- DeploymentID: *string (index)
- FeatureOfInterestID: *string (index)
- SamplingFeatureID: *string (index)
- Systems: []System
- _relations_: ObservedProperties: DatastreamObservedProperties, Schema: DatastreamSchema

### Deployment
- DeploymentType: string
- ValidTime: *common_shared.TimeRange
- Geometry: *common_shared.GoGeom
- ParentDeploymentID: *string (index)
- Lang: *string
- Keywords: common_shared.StringArray
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Characteristics: common_shared.CharacteristicGroups
- Capabilities: common_shared.CapabilityGroups
- Contacts: common_shared.ContactWrappers
- Documentation: common_shared.Documents
- History: common_shared.History
- SystemIds: *common_shared.StringArray
- DeployedSystems: DeployedSystemItems
- PlatformID: *string (index)
- Links: common_shared.Links
- _relations_: Platform: DeployedSystemItem

### DeploymentClosure
- AncestorID: string (required, index)
- DescendantID: string (required, index)
- Depth: int (required)

### Feature
- CollectionID: string (required, index)
- DateTime: *time.Time
- ValidTime: *common_shared.TimeRange
- Geometry: *common_shared.GoGeom
- Links: common_shared.Links
- Properties: common_shared.Properties

### Observation
- DatastreamID: string (required, index)
- SamplingFeatureID: *string (index)
- ProcedureLink: *common_shared.Link
- PhenomenonTime: *time.Time
- ResultTime: time.Time (required, index)
- Parameters: common_shared.Properties
- Result: json.RawMessage
- ResultLink: *common_shared.Link

### Procedure
- ProcedureType: string
- ProcessType: string
- Links: common_shared.Links
- Lang: *string
- Keywords: common_shared.StringArray
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Characteristics: common_shared.CharacteristicGroups
- Capabilities: common_shared.CapabilityGroups
- Contacts: common_shared.ContactWrappers
- Documentation: common_shared.Documents
- History: common_shared.History
- TypeOf: *common_shared.Link
- Configuration: json.RawMessage
- FeaturesOfInterest: common_shared.Links
- Inputs: common_shared.IOList
- Outputs: common_shared.IOList
- Parameters: common_shared.IOList
- Method: common_shared.Method
- Modes: json.RawMessage
- Components: json.RawMessage
- Connections: json.RawMessage
- AttachedTo: *common_shared.Link
- LocalReferenceFrames: []common_shared.SpatialFrame
- LocalTimeFrames: []common_shared.TemporalFrame
- ControlledProperties: []Property
- ObservedProperties: []Property
- Properties: common_shared.Properties
- ValidTime: *common_shared.TimeRange
- Systems: []System

### Property
- Definition: string
- PropertyType: string
- ObjectType: *string
- BaseProperty: *string
- Statistic: *string
- Qualifiers: common_shared.ComponentWrappers
- UnitOfMeasurement: *string
- Links: common_shared.Links
- Properties: common_shared.Properties

### SamplingFeature
- FeatureType: string
- ValidTime: *common_shared.TimeRange
- Geometry: *common_shared.GoGeom
- ParentSystemID: *string (index)
- ParentSystemUID: *string
- SampledFeatureID: *string (index)
- SampledFeatureUID: *string
- SampledFeatureLink: *common_shared.Link
- SampleOfIDs: *[]string
- SampleOfUIDs: *[]string
- SampleOf: *common_shared.Links
- Links: common_shared.Links
- Properties: common_shared.Properties

### System
- SystemType: string (required)
- AssetType: *string
- SMLType: *string
- ValidTime: *common_shared.TimeRange
- Geometry: *common_shared.GoGeom
- ParentSystemID: *string (index)
- SystemKindID: *string (index)
- Lang: *string
- Keywords: common_shared.StringArray
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Contacts: common_shared.ContactWrappers
- Documentation: common_shared.Documents
- History: common_shared.History
- TypeOf: *common_shared.Link
- Configuration: json.RawMessage
- FeaturesOfInterest: common_shared.Links
- Inputs: common_shared.IOList
- Outputs: common_shared.IOList
- Parameters: common_shared.IOList
- Modes: json.RawMessage
- AttachedTo: *common_shared.Link
- LocalReferenceFrames: []common_shared.SpatialFrame
- LocalTimeFrames: []common_shared.TemporalFrame
- Position: json.RawMessage
- Links: common_shared.Links
- SystemKind: Procedure (fk)
- Procedures: []Procedure
- Deployments: []Deployment
- SamplingFeatures: []SamplingFeature (fk)
- Datastreams: []Datastream
- Controlstreams: []ControlStream

### SystemEvent
- SystemID: string (required, index)
- Definition: string
- Label: string (required)
- Description: string
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- Contacts: common_shared.ContactWrappers
- Documentation: common_shared.Documents
- Time: common_shared.HistoryTime
- Properties: common_shared.ComponentWrappers
- Configuration: json.RawMessage
- Links: common_shared.Links
- TimeStart: *time.Time (index)
- TimeEnd: *time.Time (index)

### SystemHistoryRevision
- SystemID: string (required, index)
- Snapshot: json.RawMessage (required)
- ValidTime: *common_shared.TimeRange

---

# Libraries

- `e2e/schema_validator.go`
  - function GetSchemaValidator: () *SchemaValidator
  - function NewSchemaValidator: () *SchemaValidator
  - class SchemaValidator
- `internal/api/collection_handler.go` — function NewCollectionHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.CollectionRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Collection]) *CollectionHandler, class CollectionHandler
- `internal/api/collections_handler.go` — function NewCollectionsHandler: (cfg *config.Config, logger *zap.Logger) *CollectionsHandler, class CollectionsHandler
- `internal/api/command_handler.go`
  - function NewCommandHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.CommandRepository, controlStreamRepo *repository.ControlStreamRepository) *CommandHandler
  - class CommandCollectionResponse
  - class CommandHandler
- `internal/api/conformance_handler.go` — function NewConformanceHandler: (cfg *config.Config, logger *zap.Logger) *ConformanceHandler, class ConformanceHandler
- `internal/api/control_stream_handler.go`
  - function NewControlStreamHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.ControlStreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.ControlStream]) *ControlStreamHandler
  - class ControlStreamCollectionResponse
  - class ControlStreamHandler
- `internal/api/datastream_handler.go`
  - function NewDatastreamHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.DatastreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Datastream]) *DatastreamHandler
  - class DatastreamCollectionResponse
  - class DatastreamHandler
- `internal/api/deployment_handler.go` — function NewDeploymentHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.DeploymentRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Deployment]) *DeploymentHandler, class DeploymentHandler
- `internal/api/feature_handler.go` — function NewFeatureHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.FeatureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Feature]) *FeatureHandler, class FeatureHandler
- `internal/api/landing_handler.go` — function NewLandingHandler: (cfg *config.Config, logger *zap.Logger) *LandingHandler, class LandingHandler
- `internal/api/observation_handler.go`
  - function NewObservationHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.ObservationRepository, datastreamRepo *repository.DatastreamRepository) *ObservationHandler
  - class ObservationCollectionResponse
  - class ObservationHandler
- `internal/api/procedure_handler.go` — function NewProcedureHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.ProcedureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Procedure]) *ProcedureHandler, class ProcedureHandler
- `internal/api/property_handler.go` — function NewPropertyHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.PropertyRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Property]) *PropertyHandler, class PropertyHandler
- `internal/api/router.go` — function NewRouter: (cfg *config.Config, logger *zap.Logger, repos *repository.Repositories) http.Handler
- `internal/api/sampling_feature_handler.go` — function NewSamplingFeatureHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.SamplingFeatureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]) *SamplingFeatureHandler, class SamplingFeatureHandler
- `internal/api/system_event_handler.go`
  - function NewSystemEventHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.SystemEventRepository, systemRepo *repository.SystemRepository) *SystemEventHandler
  - class SystemEventCollectionResponse
  - class SystemEventHandler
- `internal/api/system_handler.go` — function NewSystemHandler: (cfg *config.Config, logger *zap.Logger, repo *repository.SystemRepository, historyRepo *repository.SystemHistoryRepository, fc *formaters.MultiFormatFormatterCollection[*domains.System], deploymentRepo *repository.DeploymentRepository, deploymentFC *formaters.MultiFormatFormatterCollection[*domains.Deployment], procedureRepo *repository.ProcedureRepository, procedureFC *formaters.MultiFormatFormatterCollection[*domains.Procedure]) *SystemHandler, class SystemHandler
- `internal/config/config.go`
  - function Load: () (*Config, error)
  - class Config
  - class ServerConfig
  - class DatabaseConfig
  - class APIConfig
- `internal/model/collection_metadata.go`
  - class CollectionMetadata
  - class Extent
  - class SpatialExtent
  - class TemporalExtent
  - class LandingPage
  - class ConformanceDeclaration
- `internal/model/common_shared/bounding-box.go` — class BoundingBox
- `internal/model/common_shared/capabilities.go` — class CapabilityGroup
- `internal/model/common_shared/characteristics.go`
  - class CharacteristicGroup
  - class ComponentWrapper
  - class BooleanComponent
  - class CountComponent
  - class QuantityComponent
  - class TimeComponent
  - _...8 more_
- `internal/model/common_shared/codeList.go` — class CodeList
- `internal/model/common_shared/configurationSettings.go`
  - class ConfigurationSettings
  - class SetValue
  - class SetArrayValue
  - class SetMode
  - class AllowedTokens
  - class ValueItem
  - _...4 more_
- `internal/model/common_shared/contacts.go`
  - class ContactInfo
  - class Phone
  - class Address
  - class ContactPersonOrg
  - class ContactLink
  - class ContactWrapper
- `internal/model/common_shared/documents.go` — class Document
- `internal/model/common_shared/extent.go` — class Extent
- `internal/model/common_shared/geometry.go` — class Geometry
- `internal/model/common_shared/go_geom.go` — function WKBHexToWKT: (hexStr string) (string, error), class GoGeom
- `internal/model/common_shared/history.go` — class HistoryTime, class HistoryEvent
- `internal/model/common_shared/io.go` — class ObservablePropertyInline, class IOItem
- `internal/model/common_shared/json_feature.go` — class JSONFeature
- `internal/model/common_shared/legalConstraint.go` — class LegalConstraint
- `internal/model/common_shared/links.go`
  - function CanonicalRel: (rel string) string
  - function OGCRel: (rel string) string
  - function RelEquals: (actual, expected string) bool
  - function StripAssociationLinks: (links Links) Links
  - class Link
- `internal/model/common_shared/method.go` — class Method
- `internal/model/common_shared/point.go` — class Point
- `internal/model/common_shared/securityConstraint.go` — class SecurityConstraint
- `internal/model/common_shared/spatial_temporal.go`
  - class Axis
  - class SpatialFrame
  - class TemporalFrame
- `internal/model/common_shared/terms.go` — class Term
- `internal/model/common_shared/time_range.go`
  - function ToTimeRange: (timeValue string) TimeRange
  - function ToTimeRangeFromSlice: (parts []string) TimeRange
  - function ParseTimeRange: (value interface{}) TimeRange
  - class TimeRange
- `internal/model/domains/collection.go`
  - function NewCollection: (id, title, description string, links []common_shared.Link, extent *common_shared.Extent, itemType string, crs []string) *Collection
  - class Collection
  - class CollectionGeoJSONFeature
- `internal/model/domains/command.go` — class Command
- `internal/model/domains/common.go` — class Base, class CommonSSN
- `internal/model/domains/control_stream.go`
  - class ControlStream
  - class ControlStreamControlledProperty
  - class ControlStreamSchema
- `internal/model/domains/datastream.go`
  - class Datastream
  - class DatastreamObservedProperty
  - class DatastreamSchema
  - class DatastreamResultLink
  - class DatastreamMessageSchema
  - class DatastreamEncoding
  - _...7 more_
- `internal/model/domains/deployment.go`
  - class Deployment
  - class DeploymentGeoJSONFeature
  - class DeploymentGeoJSONProperties
  - class DeployedSystemItem
  - class DeploymentSensorMLFeature
- `internal/model/domains/deployment_closure.go` — class DeploymentClosure
- `internal/model/domains/feature.go` — class Feature, class FeatureGeoJSONFeature
- `internal/model/domains/observation.go` — class Observation
- `internal/model/domains/procedure.go`
  - class Procedure
  - class ProcedureGeoJSONFeature
  - class ProcedureGeoJSONProperties
  - class ProcedureSensorMLFeature
- `internal/model/domains/property.go`
  - class Property
  - class PropertySensorMLFeature
  - class PropertyGeoJSONFeature
  - class PropertyGeoJSONProperties
- `internal/model/domains/sampling_feature.go`
  - class SamplingFeature
  - class SamplingFeatureGeoJSONFeature
  - class SamplingFeatureGeoJSONProperties
  - class SamplingFeatureSensorMLFeature
- `internal/model/domains/system.go`
  - class System
  - class SystemGeoJSONFeature
  - class SystemGeoJSONProperties
  - class SystemSensorMLFeature
- `internal/model/domains/system_event.go` — class SystemEvent
- `internal/model/domains/system_history_revision.go` — class SystemHistoryRevision
- `internal/model/formaters/association_links.go`
  - function SetAssociationLinksBaseURL: (baseURL string)
  - function GeoJSONSystemAssociationLinks: (links common_shared.Links) common_shared.Links
  - function DeploymentAssociationLinks: (links common_shared.Links) common_shared.Links
  - function SamplingFeatureGeoJSONAssociationLinks: (links common_shared.Links) common_shared.Links
  - function AppendGeoJSONSystemAssociationLinks: (system *domains.System) common_shared.Links
  - function AppendSensorMLSystemAssociationLinks: (system *domains.System) common_shared.Links
  - _...7 more_
- `internal/model/formaters/geojson_formatters/collection_geojson.go` — function NewFeatureCollectionGeoJSONFormatter: (repos *repository.Repositories) *FeatureCollectionGeoJSONFormatter, class FeatureCollectionGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/deployment_geojson.go` — function NewDeploymentGeoJSONFormatter: (repos *repository.Repositories) *DeploymentGeoJSONFormatter, class DeploymentGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/feature_geojson.go` — function NewFeatureGeoJSONFormatter: (repos *repository.Repositories) *FeatureGeoJSONFormatter, class FeatureGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/procedure_geojson.go` — function NewProcedureGeoJSONFormatter: (repos *repository.Repositories) *ProcedureGeoJSONFormatter, class ProcedureGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/property_geojson.go` — function NewPropertyGeoJSONFormatter: (repos *repository.Repositories) *PropertyGeoJSONFormatter, class PropertyGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/sampling_feature_geojson.go` — function NewSamplingFeatureGeoJSONFormatter: (repos *repository.Repositories) *SamplingFeatureGeoJSONFormatter, class SamplingFeatureGeoJSONFormatter
- `internal/model/formaters/geojson_formatters/system_geojson.go` — function NewSystemGeoJSONFormatter: (repos *repository.Repositories) *SystemGeoJSONFormatter, class SystemGeoJSONFormatter
- `internal/model/formaters/json_formatters/control_stream_json.go` — function NewControlStreamJSONFormatter: () *ControlStreamJSONFormatter, class ControlStreamJSONFormatter
- `internal/model/formaters/json_formatters/datastream_json.go` — function NewDatastreamJSONFormatter: () *DatastreamJSONFormatter, class DatastreamJSONFormatter
- `internal/model/formaters/multi_format_serializer.go` — class AnyFeatureCollection
- `internal/model/formaters/sensorml_formatters/deployment_sensorml.go` — function NewDeploymentSensorMLFormatter: (repos *repository.Repositories) *DeploymentSensorMLFormatter, class DeploymentSensorMLFormatter
- `internal/model/formaters/sensorml_formatters/procedure_sensorml.go` — function NewProcedureSensorMLFormatter: (repos *repository.Repositories) *ProcedureSensorMLFormatter, class ProcedureSensorMLFormatter
- `internal/model/formaters/sensorml_formatters/property_sensorml.go` — function NewPropertySensorMLFormatter: (repos *repository.Repositories) *PropertySensorMLFormatter, class PropertySensorMLFormatter
- `internal/model/formaters/sensorml_formatters/sampling_feature_sensorml.go` — function NewSamplingFeatureSensorMLFormatter: (repos *repository.Repositories) *SamplingFeatureSensorMLFormatter, class SamplingFeatureSensorMLFormatter
- `internal/model/formaters/sensorml_formatters/system_sensorml.go` — function NewSystemSensorMLFormatter: (repos *repository.Repositories) *SystemSensorMLFormatter, class SystemSensorMLFormatter
- `internal/model/generators/generators_collection.go` — function FakeCollection: () domains.Collection
- `internal/model/generators/generators_common_shared.go`
  - function FakeTerms: () common_shared.Terms
  - function FakeLink: () common_shared.Link
  - function FakeLinks: () common_shared.Links
  - function FakeContactInfo: () *common_shared.ContactInfo
  - function FakeContactPersonOrg: () *common_shared.ContactPersonOrg
  - function FakeContactLink: () *common_shared.ContactLink
  - _...8 more_
- `internal/model/generators/generators_common_shared_more.go`
  - function FakeDocument: () common_shared.Document
  - function FakeDocuments: () common_shared.Documents
  - function FakeHistoryEvent: () common_shared.HistoryEvent
  - function FakeHistory: () common_shared.History
  - function FakeProperties: () common_shared.Properties
  - function FakeSecurityConstraints: () common_shared.SecurityConstraints
  - _...8 more_
- `internal/model/generators/generators_datastream.go`
  - function FakeDatastreamJSONScalarSchema: () *domains.DatastreamSchema
  - function FakeDatastreamJSONRecordSchema: () *domains.DatastreamSchema
  - function FakeDatastreamSWEJSONSchema: () *domains.DatastreamSchema
  - function FakeDatastreamSWECsvSchema: () *domains.DatastreamSchema
  - function FakeDatastreamProtobufSchema: () *domains.DatastreamSchema
  - function FakeDatastreamOtherFormatSchema: () *domains.DatastreamSchema
  - _...9 more_
- `internal/model/generators/generators_deployment.go`
  - function FakeSetValue: () common_shared.SetValue
  - function FakeSetArrayValue: () common_shared.SetArrayValue
  - function FakeSetMode: () common_shared.SetMode
  - function FakeAllowedTokens: () common_shared.AllowedTokens
  - function FakeAllowedValues: () common_shared.AllowedValues
  - function FakeConstraint: () common_shared.Constraint
  - _...27 more_
- `internal/model/generators/generators_observation.go`
  - function FakeObservationForDatastream: (ds domains.Datastream) domains.Observation
  - function FakeObservationWithResultLink: (ds domains.Datastream) domains.Observation
  - function FakeObservationListForDatastream: (ds domains.Datastream, n int) []domains.Observation
- `internal/model/generators/generators_procedure.go`
  - function FakeProcedureMinimal: () domains.Procedure
  - function FakeProcedureObserving: () domains.Procedure
  - function FakeProcedureSampling: () domains.Procedure
  - function FakeProcedureActuating: () domains.Procedure
  - function FakeProcedureSensorDatasheet: () domains.Procedure
  - function FakeProcedureActuatorDatasheet: () domains.Procedure
  - _...8 more_
- `internal/model/generators/generators_property.go` — function FakeProperty: () domains.Property
- `internal/model/generators/generators_sampling_feature.go` — function FakeSamplingFeature: () domains.SamplingFeature
- `internal/model/generators/generators_sensorml_shared.go`
  - function FakeTerm: () common_shared.Term
  - function FakeIdentifierTerm: () common_shared.Term
  - function FakeClassifierTerm: () common_shared.Term
  - function FakeIdentifiers: (count int) common_shared.Terms
  - function FakeClassifiers: (count int) common_shared.Terms
  - function FakePhone: () *common_shared.Phone
  - _...44 more_
- `internal/model/generators/generators_system.go`
  - function FakeSystemMinimal: () domains.System
  - function FakeSystemSensor: () domains.System
  - function FakeSystemActuator: () domains.System
  - function FakeSystemSampler: () domains.System
  - function FakeSystemPlatform: () domains.System
  - function FakeSystemPhysicalSystem: () domains.System
  - _...5 more_
- `internal/model/query_params/collection_query_params.go` — class CollectionQueryParams
- `internal/model/query_params/command_query_params.go` — class CommandsQueryParams
- `internal/model/query_params/control_stream_query_params.go` — class ControlStreamsQueryParams
- `internal/model/query_params/datastream_query_params.go` — class DatastreamsQueryParams
- `internal/model/query_params/deployment_query_params.go` — class DeploymentsQueryParams
- `internal/model/query_params/feature_query_params.go` — class FeatureQueryParams, class TimeFilter
- `internal/model/query_params/observation_query_params.go` — class ObservationsQueryParams
- `internal/model/query_params/procedure_query_params.go` — class ProceduresQueryParams
- `internal/model/query_params/property_query_params.go` — class PropertiesQueryParams
- `internal/model/query_params/query_params.go` — class QueryParams
- `internal/model/query_params/sampling_feature_query_params.go` — class SamplingFeatureQueryParams
- `internal/model/query_params/system_event_query_params.go` — class SystemEventsQueryParams
- `internal/model/query_params/system_history_query_params.go` — class SystemHistoryQueryParams
- `internal/model/query_params/system_query_params.go` — class SystemQueryParams
- `internal/repository/closure.go` — function EnsureClosureSupport: (db *gorm.DB, table, idCol, parentCol, closureTable string) error, function EnsureDeleteReparentSupport: (db *gorm.DB, table, idCol, parentCol string) error
- `internal/repository/collection_repository.go` — function NewCollectionRepository: (db *gorm.DB) *CollectionRepository, class CollectionRepository
- `internal/repository/command_repository.go` — function NewCommandRepository: (db *gorm.DB) *CommandRepository, class CommandRepository
- `internal/repository/control_stream_repository.go` — function NewControlStreamRepository: (db *gorm.DB) *ControlStreamRepository, class ControlStreamRepository
- `internal/repository/datastream_repository.go` — function NewDatastreamRepository: (db *gorm.DB) *DatastreamRepository, class DatastreamRepository
- `internal/repository/deployment_repository.go` — function NewDeploymentRepository: (db *gorm.DB) *DeploymentRepository, class DeploymentRepository
- `internal/repository/feature_repository.go` — function NewFeatureRepository: (db *gorm.DB) *FeatureRepository, class FeatureRepository
- `internal/repository/observation_repository.go` — function NewObservationRepository: (db *gorm.DB) *ObservationRepository, class ObservationRepository
- `internal/repository/procedure_repository.go` — function NewProcedureRepository: (db *gorm.DB) *ProcedureRepository, class ProcedureRepository
- `internal/repository/property_repository.go` — function NewPropertyRepository: (db *gorm.DB) *PropertyRepository, class PropertyRepository
- `internal/repository/repository.go`
  - function NewRepositories: (db *gorm.DB) *Repositories
  - function AutoMigrate: (db *gorm.DB) error
  - class Repositories
- `internal/repository/repository_shared/repository.go`
  - function NewRepositories: (db *gorm.DB) *Repositories
  - function AutoMigrate: (db *gorm.DB) error
  - class Repositories
- `internal/repository/sampling_feature_repository.go` — function NewSamplingFeatureRepository: (db *gorm.DB) *SamplingFeatureRepository, class SamplingFeatureRepository
- `internal/repository/system_event_repository.go` — function NewSystemEventRepository: (db *gorm.DB) *SystemEventRepository, class SystemEventRepository
- `internal/repository/system_history_repository.go` — function NewSystemHistoryRepository: (db *gorm.DB) *SystemHistoryRepository, class SystemHistoryRepository
- `internal/repository/system_repository.go` — function NewSystemRepository: (db *gorm.DB) *SystemRepository, class SystemRepository
- `internal/repository/testutil/postgis.go`
  - function StartPostGISContainer: (ctx context.Context, t *testing.T) *PostGISContainer
  - function OpenTestDB: (t *testing.T, dsn string, opts OpenTestDBOptions) *gorm.DB
  - function DefaultSystemModels: () []interface
  - function AllModels: () []interface
  - function PtrTime: (t time.Time) *time.Time
  - function PtrStr: (s string) *string
  - _...13 more_

---

# Config

## Config Files

- `Dockerfile`
- `docker-compose.yml`
- `go.mod`

---

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

---

# Test Coverage

> **38%** of routes and models are covered by tests
> 31 test files found

## Covered Routes

- GET:content-type
- GET:/systems
- POST:/systems
- PUT:/systems
- DELETE:/systems
- GET:/datastreams
- PUT:/datastreams
- DELETE:/datastreams
- GET:/controlstreams
- PUT:/controlstreams
- DELETE:/controlstreams
- GET:/commands
- PUT:/commands
- DELETE:/commands
- GET:/observations
- PUT:/observations
- DELETE:/observations
- GET:/deployments
- POST:/deployments
- PUT:/deployments
- DELETE:/deployments
- GET:/procedures
- POST:/procedures
- PUT:/procedures
- DELETE:/procedures
- GET:/properties
- POST:/properties
- PUT:/properties
- DELETE:/properties
- GET:/
- POST:/collections
- GET:/collections
- POST:/
- PUT:/
- DELETE:/
- GET:/subsystems
- POST:/subsystems
- GET:/events
- POST:/events
- GET:/history
- POST:/datastreams
- POST:/controlstreams
- GET:/schema
- PUT:/schema
- POST:/observations
- POST:/commands
- GET:/subdeployments
- POST:/subdeployments
- GET:limit
- GET:offset
- GET:id

## Covered Models

- Command
- Base
- CommonSSN
- ControlStream
- Datastream
- Deployment
- Feature
- Observation
- Procedure
- Property
- SamplingFeature
- System
- SystemEvent
- SystemHistoryRevision

---

_Generated by [codesight](https://github.com/Houseofmvps/codesight) — see your codebase clearly_