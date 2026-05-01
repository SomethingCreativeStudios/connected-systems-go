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
