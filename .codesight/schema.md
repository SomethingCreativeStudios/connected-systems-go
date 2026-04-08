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
