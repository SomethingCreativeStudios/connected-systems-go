# connected-systems-go — AI Context Map

> **Stack:** go-net-http, chi | gorm | unknown | go

> 155 routes | 121 models | 0 components | 129 lib files | 14 env vars | 0 middleware | 301 import links
> **Token savings:** this file is ~15,100 tokens. Without it, AI exploration would cost ~172,300 tokens. **Saves ~157,200 tokens per conversation.**

---

# Routes

- `GET` `Location` params()
- `GET` `Content-Type` params()
- `GET` `Accept` params()
- `GET` `cascade` params() [db]
- `GET` `content-type` params() [db]
- `GET` `/collections/{collectionId}/items` params(collectionId) [auth, db]
- `POST` `/collections/{collectionId}/items` params(collectionId) [auth, db]
- `PUT` `/collections/{collectionId}/items` params(collectionId) [auth, db]
- `DELETE` `/collections/{collectionId}/items` params(collectionId) [auth, db]
- `GET` `/{featureId}` params(featureId) [auth, db]
- `PUT` `/{featureId}` params(featureId) [auth, db]
- `DELETE` `/{featureId}` params(featureId) [auth, db]
- `GET` `/systems` params() [auth, db]
- `POST` `/systems` params() [auth, db]
- `PUT` `/systems` params() [auth, db]
- `DELETE` `/systems` params() [auth, db]
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
- `GET` `/{id}` params(id) [auth, db]
- `PUT` `/{id}` params(id) [auth, db]
- `DELETE` `/{id}` params(id) [auth, db]
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
- `GET` `/events/{eventId}` params(eventId) [auth, db]
- `PUT` `/events/{eventId}` params(eventId) [auth, db]
- `DELETE` `/events/{eventId}` params(eventId) [auth, db]
- `GET` `/history/{revId}` params(revId) [auth, db]
- `PUT` `/history/{revId}` params(revId) [auth, db]
- `DELETE` `/history/{revId}` params(revId) [auth, db]
- `GET` `/systemEvents` params() [auth, db]
- `GET` `/datastreams` params() [auth, db]
- `PUT` `/datastreams` params() [auth, db]
- `DELETE` `/datastreams` params() [auth, db]
- `GET` `/datastreams/schema` params() [auth, db]
- `PUT` `/datastreams/schema` params() [auth, db]
- `GET` `/datastreams/observations` params() [auth, db]
- `POST` `/datastreams/observations` params() [auth, db]
- `GET` `/{dataStreamId}` params(dataStreamId) [auth, db]
- `PUT` `/{dataStreamId}` params(dataStreamId) [auth, db]
- `DELETE` `/{dataStreamId}` params(dataStreamId) [auth, db]
- `GET` `/{dataStreamId}/schema` params(dataStreamId) [auth, db]
- `PUT` `/{dataStreamId}/schema` params(dataStreamId) [auth, db]
- `GET` `/{dataStreamId}/observations` params(dataStreamId) [auth, db]
- `POST` `/{dataStreamId}/observations` params(dataStreamId) [auth, db]
- `GET` `/controlstreams` params() [auth, db]
- `PUT` `/controlstreams` params() [auth, db]
- `DELETE` `/controlstreams` params() [auth, db]
- `GET` `/controlstreams/schema` params() [auth, db]
- `PUT` `/controlstreams/schema` params() [auth, db]
- `GET` `/controlstreams/commands` params() [auth, db]
- `POST` `/controlstreams/commands` params() [auth, db]
- `GET` `/{controlStreamId}` params(controlStreamId) [auth, db]
- `PUT` `/{controlStreamId}` params(controlStreamId) [auth, db]
- `DELETE` `/{controlStreamId}` params(controlStreamId) [auth, db]
- `GET` `/{controlStreamId}/schema` params(controlStreamId) [auth, db]
- `PUT` `/{controlStreamId}/schema` params(controlStreamId) [auth, db]
- `GET` `/{controlStreamId}/commands` params(controlStreamId) [auth, db]
- `POST` `/{controlStreamId}/commands` params(controlStreamId) [auth, db]
- `GET` `/commands` params() [auth, db]
- `PUT` `/commands` params() [auth, db]
- `DELETE` `/commands` params() [auth, db]
- `GET` `/{cmdId}` params(cmdId) [auth, db]
- `PUT` `/{cmdId}` params(cmdId) [auth, db]
- `DELETE` `/{cmdId}` params(cmdId) [auth, db]
- `GET` `/observations` params() [auth, db]
- `PUT` `/observations` params() [auth, db]
- `DELETE` `/observations` params() [auth, db]
- `GET` `/{obsId}` params(obsId) [auth, db]
- `PUT` `/{obsId}` params(obsId) [auth, db]
- `DELETE` `/{obsId}` params(obsId) [auth, db]
- `GET` `/deployments` params() [auth, db]
- `POST` `/deployments` params() [auth, db]
- `PUT` `/deployments` params() [auth, db]
- `DELETE` `/deployments` params() [auth, db]
- `GET` `/deployments/subdeployments` params() [auth, db]
- `POST` `/deployments/subdeployments` params() [auth, db]
- `GET` `/{id}/subdeployments` params(id) [auth, db]
- `POST` `/{id}/subdeployments` params(id) [auth, db]
- `GET` `/procedures` params() [auth, db]
- `POST` `/procedures` params() [auth, db]
- `PUT` `/procedures` params() [auth, db]
- `DELETE` `/procedures` params() [auth, db]
- `GET` `/samplingFeatures` params() [auth, db]
- `PUT` `/samplingFeatures` params() [auth, db]
- `DELETE` `/samplingFeatures` params() [auth, db]
- `GET` `/properties` params() [auth, db]
- `POST` `/properties` params() [auth, db]
- `PUT` `/properties` params() [auth, db]
- `DELETE` `/properties` params() [auth, db]
- `GET` `/` params() [auth, db]
- `GET` `/conformance` params() [auth, db]
- `POST` `/collections` params() [auth, db]
- `GET` `/collections` params() [auth, db]
- `GET` `/collections/{collectionId}` params(collectionId) [auth, db]
- `POST` `/` params() [auth, db]
- `PUT` `/` params() [auth, db]
- `DELETE` `/` params() [auth, db]
- `GET` `/subsystems` params() [auth, db]
- `POST` `/subsystems` params() [auth, db]
- `GET` `/events` params() [auth, db]
- `POST` `/events` params() [auth, db]
- `GET` `/history` params() [auth, db]
- `POST` `/samplingFeatures` params() [auth, db]
- `POST` `/datastreams` params() [auth, db]
- `POST` `/controlstreams` params() [auth, db]
- `GET` `/schema` params() [auth, db]
- `PUT` `/schema` params() [auth, db]
- `POST` `/observations` params() [auth, db]
- `POST` `/commands` params() [auth, db]
- `GET` `/subdeployments` params() [auth, db]
- `POST` `/subdeployments` params() [auth, db]
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
- `GET` `limit` params() [db]
- `GET` `offset` params() [db]
- `GET` `id` params() [db]
- `GET` `q` params() [db]
- `GET` `eventType` params() [db]
- `GET` `keyword` params() [db]
- `GET` `procedure` params() [db]
- `GET` `geom` params() [db]

---

# Schema

### collectionsResponse
- Links: common_shared.Links
- Collections: []*domains.Collection
- NumberMatched: int
- NumberReturned: int

### CommandCollectionResponse
- Items: []any
- Links: common_shared.Links

### ControlStreamCollectionResponse
- Items: []any
- Links: common_shared.Links

### DatastreamCollectionResponse
- Items: []any
- Links: common_shared.Links

### ObservationCollectionResponse
- Items: []any
- Links: common_shared.Links

### SystemEventCollectionResponse
- Items: []any
- Links: common_shared.Links

### CollectionMetadata
- ID: string
- Title: string
- Description: string
- Links: common_shared.Links
- ItemType: string
- FeatureType: string
- CRS: []string
- _relations_: Extent: Extent

### SpatialExtent
- Bbox: [][]float64
- CRS: string

### TemporalExtent
- Interval: [][]string
- TRS: string

### LandingPage
- Title: string
- Description: string
- Links: common_shared.Links

### ConformanceDeclaration
- ConformsTo: []string

### CapabilityGroup
- ID: string
- Label: string
- Description: string
- Definition: string
- Conditions: []ComponentWrapper
- Capabilities: []ComponentWrapper

### CharacteristicGroup
- ID: string
- Label: string
- Description: string
- Definition: string
- Conditions: []ComponentWrapper
- Characteristics: []ComponentWrapper

### ComponentWrapper
- Type: string
- Definition: string
- Label: string
- ReferenceFrame: string
- AxisID: string
- LocalFrame: string
- Updatable: *bool
- Optional: *bool
- UOM: json.RawMessage
- Constraint: json.RawMessage
- NilValues: json.RawMessage
- Value: json.RawMessage
- Component: Component
- Raw: json.RawMessage

### BooleanComponent
- Type: string
- Definition: string
- Label: string
- Value: bool

### CountComponent
- Type: string
- Definition: string
- Label: string
- Value: int

### QuantityComponent
- Type: string
- Definition: string
- Label: string
- UOM: json.RawMessage
- Value: json.RawMessage

### TimeComponent
- Type: string
- Definition: string
- Label: string
- UOM: json.RawMessage
- Value: json.RawMessage

### CategoryComponent
- Type: string
- Definition: string
- Label: string
- Value: string

### TextComponent
- Type: string
- Definition: string
- Label: string
- Value: string

### CountRangeComponent
- Type: string
- Definition: string
- Label: string
- Value: []int

### QuantityRangeComponent
- Type: string
- Definition: string
- Label: string
- UOM: json.RawMessage
- Value: []json.RawMessage

### TimeRangeComponent
- Type: string
- Definition: string
- Label: string
- UOM: json.RawMessage
- Value: []json.RawMessage

### VectorComponent
- Type: string
- Definition: string
- Label: string
- ReferenceFrame: string
- LocalFrame: string
- Coordinates: json.RawMessage

### ArrayComponent
- Type: string
- Definition: string
- Label: string
- ElementCount: int
- Coordinates: json.RawMessage

### CodeList
- CodeSpace: string
- Value: string

### ConfigurationSettings
- SetValues: []SetValue
- SetArrayValues: []SetArrayValue
- SetModes: []SetMode
- SetConstraints: []Constraint
- SetStatus: []SetStatus

### SetValue
- Ref: string
- Value: interface

### SetArrayValue
- Ref: string
- Value: []interface

### SetMode
- Ref: string
- Value: string

### AllowedTokens
- Type: string
- Values: []string
- Pattern: string

### ValueItem
- Number: *float64
- String: *string

### AllowedValues
- Type: string
- Values: []ValueItem
- Intervals: [][]ValueItem
- SignificantFigures: *int

### AllowedTimes
- Type: string
- Values: []string
- Intervals: [][]string
- SignificantFigures: *int

### Constraint
- Type: string
- Ref: string
- _relations_: Tokens: AllowedTokens, Values: AllowedValues, Times: AllowedTimes

### SetStatus
- Ref: string
- Value: string

### ContactInfo
- Website: string
- HoursOfService: string
- ContactInstructions: string
- _relations_: Phone: Phone, Address: Address

### Phone
- Voice: string
- Facsimile: string

### Address
- DeliveryPoint: string
- City: string
- AdministrativeArea: string
- PostalCode: string
- Country: string
- ElectronicMailAddress: string

### ContactPersonOrg
- IndividualName: string
- OrganisationName: string
- PositionName: string
- Role: string
- _relations_: ContactInfo: ContactInfo

### ContactLink
- Role: string
- Name: string
- Link: Link

### ContactWrapper
- Raw: json.RawMessage
- _relations_: Person: ContactPersonOrg, LinkRef: ContactLink

### Document
- Role: string
- Name: string
- Description: string
- Link: Link

### Geometry
- Type: string
- Coordinates: interface

### HistoryTime
- Instant: *time.Time
- _relations_: Range: TimeRange

### HistoryEvent
- ID: string
- Label: string
- Description: string
- Definition: string
- Identifiers: []Term
- Classifiers: []Term
- Contacts: []ContactWrapper
- Documentation: Documents
- Time: HistoryTime
- Properties: []ComponentWrapper
- Configuration: json.RawMessage

### ObservablePropertyInline
- Type: string
- Definition: string
- Label: string

### IOItem
- Raw: json.RawMessage
- _relations_: Component: ComponentWrapper, Observable: ObservablePropertyInline

### JSONFeature
- ID: string

### LegalConstraint
- AccessConstraints: CodeLists
- UseConstraints: CodeLists
- OtherConstraints: Terms
- UserLimitations: *string

### Link
- Href: string
- Rel: string
- Type: string
- Title: string
- UID: *string

### Method
- Algorithm: string
- Description: string

### Point
- Type: string
- Coordinates: []float64

### SecurityConstraint
- Type: string
- Extra: map[string]interface

### Axis
- Name: string
- Description: string

### SpatialFrame
- ID: string
- Label: string
- Description: string
- Origin: string
- Axes: []Axis

### TemporalFrame
- ID: string
- Label: string
- Description: string
- Origin: string

### Term
- Definition: string
- Label: string
- CodeSpace: string
- Value: string

### TimeRange
- Start: *time.Time
- End: *time.Time

### Collection
- ID: string (pk)
- Title: string
- Description: string
- Links: common_shared.Links
- Extent: *common_shared.Extent
- ItemType: string (default)
- CRS: []string

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

### ControlStreamControlledProperty
- Definition: string
- Label: string
- Description: string

### ControlStreamSchema
- CommandFormat: string
- _relations_: ParametersSchema: DatastreamDataComponent, ResultSchema: DatastreamDataComponent, FeasibilityResultSchema: DatastreamDataComponent, RecordSchema: DatastreamDataComponent, Encoding: DatastreamEncoding

### Datastream
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

### DatastreamObservedProperty
- Definition: string
- Label: string
- Description: string

### DatastreamSchema
- ObsFormat: string
- Any: common_shared.Properties
- _relations_: ParametersSchema: DatastreamDataComponent, ResultSchema: DatastreamDataComponent, ResultLink: DatastreamResultLink, RecordSchema: DatastreamDataComponent, Encoding: DatastreamEncoding, MessageSchema: DatastreamMessageSchema

### DatastreamResultLink
- MediaType: string

### DatastreamMessageSchema
- Inline: *string
- Link: *common_shared.Link

### DatastreamEncoding
- Type: string
- CollapseWhiteSpaces: *bool
- DecimalSeparator: string
- TokenSeparator: string
- BlockSeparator: string
- RecordsAsArrays: *bool
- VectorsAsArrays: *bool
- ByteOrder: string
- ByteEncoding: string
- ByteLength: *int
- Members: []DatastreamBinaryMember
- Extensions: common_shared.Properties

### DatastreamBinaryMember
- Ref: string
- Compression: string
- Encryption: string
- DataType: string
- ByteLength: *int
- ByteOrder: string
- Extensions: common_shared.Properties

### DatastreamDataComponent
- ID: string
- Name: string
- Type: string
- Label: string
- Description: string
- Definition: string
- Updatable: *bool
- Optional: *bool
- ReferenceFrame: string
- LocalFrame: string
- AxisID: string
- CodeSpace: string
- NilValues: []DatastreamNilValue
- Value: json.RawMessage
- Fields: []DatastreamNamedComponent
- Coordinates: []DatastreamNamedComponent
- Items: []DatastreamNamedComponent
- Values: json.RawMessage
- SRS: string
- Extensions: common_shared.Properties
- _relations_: UOM: DatastreamUOM, Constraint: DatastreamConstraint, ElementCount: DatastreamElementCount, ElementType: DatastreamNamedComponent, Encoding: DatastreamEncoding, ChoiceValue: DatastreamDataComponent

### DatastreamNamedComponent
- Name: string

### DatastreamElementCount
- Fixed: *int
- _relations_: Component: DatastreamDataComponent

### DatastreamUOM
- Label: string
- Symbol: string
- Code: string
- Href: string

### DatastreamConstraint
- Type: string
- Values: json.RawMessage
- Intervals: json.RawMessage
- Pattern: string
- SignificantFigures: *int
- Extensions: common_shared.Properties

### DatastreamNilValue
- Reason: string
- Value: json.RawMessage

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

### DeploymentGeoJSONFeature
- Type: string
- ID: string
- Geometry: *common_shared.GoGeom
- Properties: DeploymentGeoJSONProperties
- Links: common_shared.Links

### DeploymentGeoJSONProperties
- UID: UniqueID
- Name: string
- Description: string
- FeatureType: string
- ValidTime: *common_shared.TimeRange
- Definition: string
- Platform: *common_shared.Link
- DeployedSystems: common_shared.Links

### DeployedSystemItem
- Name: string
- Description: string
- System: common_shared.Link
- Configuration: common_shared.ConfigurationSettings

### DeploymentSensorMLFeature
- ID: string
- Type: string
- Label: string
- Description: string
- UniqueID: string
- Definition: string
- ValidTime: *common_shared.TimeRange
- Location: *common_shared.GoGeom
- DeployedSystems: []DeployedSystemItem
- Links: common_shared.Links
- Lang: *string
- Keywords: []string
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Characteristics: []common_shared.CharacteristicGroup
- Capabilities: []common_shared.CapabilityGroup
- Contacts: []common_shared.ContactWrapper
- Documentation: common_shared.Documents
- History: common_shared.History
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

### FeatureGeoJSONFeature
- Type: string
- ID: string
- Geometry: *common_shared.GoGeom
- Properties: map[string]interface
- Links: common_shared.Links

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

### ProcedureGeoJSONFeature
- Type: string
- ID: string
- Geometry: *common_shared.GoGeom
- Properties: ProcedureGeoJSONProperties
- Links: common_shared.Links

### ProcedureGeoJSONProperties
- UID: UniqueID
- Name: string
- Description: string
- FeatureType: string
- ValidTime: *common_shared.TimeRange

### ProcedureSensorMLFeature
- ID: string
- Type: string
- Label: string
- Description: string
- UniqueID: string
- Definition: string
- Lang: *string
- Keywords: []string
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Characteristics: []common_shared.CharacteristicGroup
- Capabilities: []common_shared.CapabilityGroup
- Contacts: []common_shared.ContactWrapper
- Documentation: common_shared.Documents
- History: common_shared.History
- TypeOf: *common_shared.Link
- Configuration: json.RawMessage
- FeaturesOfInterest: common_shared.Links
- Inputs: common_shared.IOList
- Outputs: common_shared.IOList
- Parameters: common_shared.IOList
- Modes: json.RawMessage
- Method: common_shared.Method
- Components: json.RawMessage
- Connections: json.RawMessage
- AttachedTo: *common_shared.Link
- LocalReferenceFrames: []common_shared.SpatialFrame
- LocalTimeFrames: []common_shared.TemporalFrame
- Position: json.RawMessage
- ValidTime: *common_shared.TimeRange
- Links: common_shared.Links

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

### PropertySensorMLFeature
- ID: string
- Label: string
- Description: string
- UniqueID: string
- BaseProperty: *string
- ObjectType: *string
- Statistic: *string
- Qualifiers: common_shared.ComponentWrappers
- Links: common_shared.Links

### PropertyGeoJSONFeature
- Type: string
- ID: string
- Geometry: interface
- Properties: PropertyGeoJSONProperties
- Links: common_shared.Links

### PropertyGeoJSONProperties
- UID: UniqueID
- Name: string
- Description: string
- Definition: string
- PropertyType: string
- BaseProperty: *string
- ObjectType: *string
- Statistic: *string
- Qualifiers: common_shared.ComponentWrappers
- UnitOfMeasurement: *string

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

### SamplingFeatureGeoJSONFeature
- Type: string
- ID: string
- Geometry: *common_shared.GoGeom
- Properties: SamplingFeatureGeoJSONProperties
- Links: common_shared.Links

### SamplingFeatureGeoJSONProperties
- UID: UniqueID
- Name: string
- Description: string
- FeatureType: string
- ValidTime: *common_shared.TimeRange
- SampledFeatureLink: *common_shared.Link

### SamplingFeatureSensorMLFeature
- ID: string
- Type: string
- Label: string
- Description: string
- UniqueID: string
- Definition: string
- ValidTime: *common_shared.TimeRange
- SampledFeatureLink: *common_shared.Link
- SampleOf: *common_shared.Links
- Links: common_shared.Links

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

### SystemGeoJSONFeature
- Type: string
- ID: string
- Geometry: *common_shared.GoGeom
- Properties: SystemGeoJSONProperties
- Links: common_shared.Links

### SystemGeoJSONProperties
- UID: UniqueID
- Name: string
- Description: string
- FeatureType: string
- AssetType: *string
- SMLType: *string
- ValidTime: *common_shared.TimeRange
- SystemKind: *common_shared.Link
- Lang: *string
- Keywords: []string
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- Contacts: []common_shared.ContactWrapper
- Documentation: common_shared.Documents
- History: common_shared.History
- Configuration: json.RawMessage
- FeaturesOfInterest: common_shared.Links
- Inputs: common_shared.IOList
- Outputs: common_shared.IOList
- Parameters: common_shared.IOList
- Modes: json.RawMessage
- LocalReferenceFrames: []common_shared.SpatialFrame
- LocalTimeFrames: []common_shared.TemporalFrame
- Position: json.RawMessage

### SystemSensorMLFeature
- ID: string
- Type: string
- Label: string
- Description: string
- UniqueID: string
- ValidTime: *common_shared.TimeRange
- Lang: *string
- Keywords: []string
- Identifiers: common_shared.Terms
- Classifiers: common_shared.Terms
- SecurityConstraints: common_shared.SecurityConstraints
- LegalConstraints: common_shared.LegalConstraints
- Contacts: []common_shared.ContactWrapper
- Documentation: common_shared.Documents
- History: common_shared.History
- Definition: string
- TypeOf: *common_shared.Link
- Configuration: json.RawMessage
- FeaturesOfInterest: common_shared.Links
- Inputs: common_shared.IOList
- Outputs: common_shared.IOList
- Parameters: common_shared.IOList
- Modes: json.RawMessage
- Position: json.RawMessage
- AttachedTo: *common_shared.Link
- LocalReferenceFrames: []common_shared.SpatialFrame
- LocalTimeFrames: []common_shared.TemporalFrame
- Links: common_shared.Links

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

### AnyFeatureCollection
- Type: string
- Features: []any
- NumberMatched: *int
- NumberReturned: int
- Links: common_shared.Links

### FeatureQueryParams
- BBox: []float64
- CollectionID: string
- _relations_: DateTime: TimeFilter

### TimeFilter
- Start: *time.Time
- End: *time.Time

### link
- Href: string
- Rel: string
- Type: string
- Title: string

### systemProperties
- UID: string
- Name: string
- Description: string
- FeatureType: string
- AssetType: string
- Keywords: []string

### geoFeaturePayload
- Type: string
- Properties: systemProperties
- Geometry: map[string]any
- Links: []link

### systemsFeatureCollection
- Features: []struct
- ID: string
- Properties: struct
- UID: string

### controlStreamCollection
- Items: []struct
- ID: string
- UID: string
- Name: string

### controlStreamPayload
- UID: string
- Name: string
- Description: string
- SystemLink: *link
- InputName: string
- Type: string
- Formats: []string
- ControlledProperties: []map[string]string
- Schema: map[string]any
- Links: []map[string]interface

### datastreamCollection
- Items: []struct
- ID: string
- UID: string

### commandCollection
- Items: []commandResource

### commandResource
- ID: string
- ControlStreamID: string
- SamplingFeatureID: string
- ProcedureLink: *link
- IssueTime: string
- ExecutionTime: []string
- Sender: string
- CurrentStatus: string
- Parameters: json.RawMessage

### continuousMoveCommand
- PanVelocity: float64
- TiltVelocity: float64
- ZoomVelocity: float64
- TimeoutSec: *float64

### absoluteMoveCommand
- Pan: *float64
- Tilt: *float64
- Zoom: *float64
- Speed: *float64

### presetCommand
- Action: string
- PresetToken: string
- PresetName: string
- Speed: *float64

---

# Libraries

- `cs-api-client/src/client.ts`
  - class CsApiClient
  - interface ClientOptions
  - interface CollectionResponse
- `cs-api-client/src/codecs/deployment.ts`
  - function decodeDeploymentGeoJSON: (feature) => Deployment
  - function encodeDeploymentGeoJSON: (deployment) => GeoJsonFeature<DeploymentGeoJSONProperties>
  - function decodeDeploymentSensorML: (data) => Deployment
  - function encodeDeploymentSensorML: (deployment) => SensorMLDeployment
- `cs-api-client/src/codecs/procedure.ts`
  - function decodeProcedureGeoJSON: (feature) => Procedure
  - function encodeProcedureGeoJSON: (procedure) => GeoJsonFeature<ProcedureGeoJSONProperties>
  - function decodeProcedureSensorML: (data) => Procedure
  - function encodeProcedureSensorML: (procedure) => SensorMLProcedure
- `cs-api-client/src/codecs/property.ts`
  - function decodePropertyGeoJSON: (feature) => Property
  - function encodePropertyGeoJSON: (property) => GeoJsonFeature<PropertyGeoJSONProperties>
  - function decodePropertySensorML: (data) => Property
  - function encodePropertySensorML: (property) => SensorMLProperty
- `cs-api-client/src/codecs/sampling-feature.ts`
  - function decodeSamplingFeatureGeoJSON: (feature) => SamplingFeature
  - function encodeSamplingFeatureGeoJSON: (sf) => GeoJsonFeature<SamplingFeatureGeoJSONProperties>
  - function decodeSamplingFeatureSensorML: (data) => SamplingFeature
  - function encodeSamplingFeatureSensorML: (sf) => SensorMLSamplingFeature
- `cs-api-client/src/codecs/system.ts`
  - function decodeSystemGeoJSON: (feature) => System
  - function encodeSystemGeoJSON: (system) => GeoJsonFeature<SystemGeoJSONProperties>
  - function decodeSystemSensorML: (data) => System
  - function encodeSystemSensorML: (system) => SensorMLSystem
- `cs-api-client/src/codecs/utils.ts` — function omitEmpty: (obj) => T
- `cs-api-client/src/content-types.ts`
  - function normalizeContentType: (value) => string
  - function isGeoJSONContentType: (value) => boolean
  - function isSensorMLContentType: (value) => boolean
  - function isJSONContentType: (value) => boolean
  - type ContentType
  - const CONTENT_TYPES
- `cs-api-client/src/errors.ts` — class CsApiError
- `cs-api-client/src/http.ts`
  - function setAuthHeader: (value) => void
  - function getAuthHeader: () => string
  - class HttpClient
  - interface HttpClientOptions
  - interface RequestOptions
  - interface HttpResponse
- `cs-api-client/src/util.ts`
  - function toOptionalString: (value) => string | undefined
  - function deepClone: (value) => T
  - function ensureArray: (value) => T[]
  - interface JsonObject
  - type JsonPrimitive
  - type JsonValue
- `cs-api-viewer/src/schema-components/geometry-editor/composable/useGeometry.ts`
  - function defaultCoordsForType: (type) => unknown
  - function coordsEqual: (a, b) => boolean
  - function mapHint: (type) => string | null
  - function useGeometry: (props) => void
  - interface UseGeometryProps
  - const GEOMETRY_TYPES
- `cs-api-viewer/src/schema-components/required-fields.ts` — function checkRequiredFields: (resourceKey, data) => boolean
- `cs-api-viewer/src/schema-components/schema-context.ts`
  - function apiFetch: (url, init?) => Promise<Response>
  - const schemaApiBase
  - const schemaCurrentResourceId
  - const schemaNavigateTo
  - const schemaOpenFeatureTab
  - const schemaBuildSystemFromProcedure
  - _...1 more_
- `cs-api-viewer/src/schema-components/utils.ts`
  - function defaultArrayItem: (value, path) => unknown
  - function normalizeKey: (key) => string
  - function matchesKey: (normalizedKey, expected) => boolean
  - function humanizeKey: (key) => string
  - function formatPrimitive: (value) => string
  - function readString: (value) => string | undefined
  - _...16 more_
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
- `ptz-test/webapp/src/api.js` — function createApi: (baseUrl) => void

---

# Config

## Environment Variables

- `CS_API_BASE_URL` **required** — ptz-test/main.go
- `ONVIF_PASSWORD` **required** — ptz-test/main.go
- `ONVIF_PROFILE_TOKEN` **required** — ptz-test/main.go
- `ONVIF_SNAPSHOT_PROFILE_TOKEN` **required** — ptz-test/main.go
- `ONVIF_USERNAME` **required** — ptz-test/main.go
- `ONVIF_XADDR` **required** — ptz-test/main.go
- `PTZ_DISCOVER` **required** — ptz-test/main.go
- `PTZ_DISCOVER_INTERFACE` **required** — ptz-test/main.go
- `PTZ_POLL_INTERVAL_SEC` **required** — ptz-test/main.go
- `PTZ_REDISCOVER_INTERVAL_SEC` **required** — ptz-test/main.go
- `PTZ_SNAPSHOT_INTERVAL_MS` **required** — ptz-test/main.go
- `PTZ_SNAPSHOT_RTSP_URI` **required** — ptz-test/main.go
- `PTZ_SNAPSHOT_SOURCE` **required** — ptz-test/main.go
- `VITE_API_PROXY_TARGET` **required** — cs-api-viewer/vite.config.ts

## Config Files

- `Dockerfile`
- `cs-api-viewer/vite.config.ts`
- `docker-compose.yml`
- `go.mod`
- `ptz-test/webapp/vite.config.js`

---

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

---

_Generated by [codesight](https://github.com/Houseofmvps/codesight) — see your codebase clearly_