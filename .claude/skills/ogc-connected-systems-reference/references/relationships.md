# Relationships

## Relationship Model

The most useful practical model in this repo is not just endpoint-based. It is a semantic graph.

## System And Procedure

The system-to-procedure relationship is represented differently depending on the schema family:

- In GeoJSON `System`, the procedure or model reference appears as `systemKind@link`
- In SensorML `System`, inheritance from a base process is expressed with `typeOf`

Practical interpretation for this project:

- `Procedure` is the generic definition of how a class of systems behaves
- It defines reusable structure such as outputs, inputs, parameters, contacts, documentation, capabilities, and other descriptive metadata
- It should be treated as the generic configuration or template for a system family
- A concrete `System` then points at that procedure and inherits most of its descriptive behavior from it

This is the key decoupling pattern:

- `Procedure` holds the reusable model-level description
- `System` holds the instance-level identity and contextual state
- This reduces duplication and makes mass creation and maintenance easier

In short, procedures describe the kind of thing, while systems describe the actual thing.

## Procedure Versus System Scope

A useful way to divide responsibilities is:

- `Procedure`: generic process description, structure, inputs, outputs, parameters, contacts, docs, capabilities, semantic model
- `System`: concrete deployed or managed asset instance, identity, validity, parent and child membership, and instance-specific overrides

Location is typically not the main concern of a procedure. Spatial and contextual placement belongs more naturally with systems, deployments, and sampling features.

## System And Sampling Feature

Sampling features are system-attached resources.

The OpenAPI makes this explicit for creation:

- Sampling features are created at `/systems/{systemId}/samplingFeatures`

The key semantic field is:

- `sampledFeature@link`

Practical interpretation for this project:

- `Sampling Feature` is the bridge from a system to the ultimate feature of interest
- `sampledFeature@link` acts as the feature-of-interest target from a search perspective

So when reasoning about FOI search:

- A system's FOI is discovered through its attached sampling features
- The effective FOI identifier is the identifier of the `sampledFeature@link` target

## FOI Search Mapping

The OpenAPI defines the `foi` query generically as selecting resources associated with a feature of interest.

For this project, the practical mapping should be:

- `System?foi=...`: match a system if any attached sampling feature has a `sampledFeature@link` matching that FOI ID or URI
- `Deployment?foi=...`: match a deployment if any deployed system has a sampling feature whose `sampledFeature@link` matches that FOI ID or URI
- `Datastream?foi=...` and `ControlStream?foi=...`: prefer the stream's attached `samplingFeature@link` and its resolved `sampledFeature@link` meaning, even if the stream also exposes `featureOfInterest@link`

That gives a consistent search model centered on sampling features as the FOI bridge.

## Deployment And Systems

Deployments describe systems in context.

In the schemas, deployments can carry:

- `deployedSystems@link` in GeoJSON and OpenAPI representations
- `deployedSystems` in SensorML-style descriptions
- optional `platform` and deployment-specific `configuration`

Practical interpretation:

- A deployment is not the generic definition of a system
- A deployment is the contextual packaging of one or more concrete systems in place and time
- Deployment-level search for FOI or observed and controlled properties should be evaluated through the systems it deploys

So for this project:

- `Deployment?observedProperty=...`: match if any deployed system has a datastream with that observed property
- `Deployment?controlledProperty=...`: match if any deployed system has a control stream with that controlled property

## System, Datastream, And Control Stream

Datastreams and control streams are the operational representation of a system's behavior.

Practical mapping:

- `Datastream` represents a system or procedure output channel
- `Control Stream` represents a system or procedure input channel

The schema fields that support this are:

- `Datastream.outputName`
- `ControlStream.inputName`

So the intended mapping is:

- A system's datastreams should be mappable to one of the outputs defined by its procedure or instance description
- A system's control streams should be mappable to one of the inputs defined by its procedure or instance description
- `Datastream.outputName` should identify the specific system output that feeds the stream
- `ControlStream.inputName` should identify the specific system input or command channel that the stream drives

This is the operational bridge between Part 1 descriptions and Part 2 runtime data.

## Part 2 Stream Inheritance Model

Part 2 breaks the simple Part 1 pattern where inline `...@link` fields are usually treated as client-owned semantic references.

For datastreams and control streams, several inline association links are better treated as server-derived materializations of the stream's attached system context.

Practical interpretation for this project:

- a stream is attached to exactly one system
- the stream's `outputName` or `inputName` anchors it to a specific system output or input
- once that system-side attachment is known, the server can derive the related procedure, deployment, sampling feature, and ultimate feature-of-interest links for the stream

That means the stream is not acting like an independent semantic root. It is acting like a runtime view over one operational channel of a specific system.

## Datastream Relationship Rules

For this project, a `Datastream` should be interpreted as:

- produced by one system
- tied to one system output via `outputName`
- optionally scoped to one deployment context
- optionally scoped to one sampling feature context
- semantically aligned with the system's procedure

The most useful derivation chain is:

- `Datastream -> system@link`
- `system@link + outputName -> specific system output`
- `System -> Procedure`
- `Procedure -> matching output definition`
- `Datastream.samplingFeature@link -> SamplingFeature.sampledFeature@link -> Datastream.featureOfInterest@link`

So the stream-level links should be read as contextualized projections of the attached system and sampling-feature state, not as freely authored references.

## ControlStream Relationship Rules

For this project, a `ControlStream` should be interpreted as:

- received by one system
- tied to one system input via `inputName`
- optionally scoped to one deployment context
- optionally scoped to one sampling feature context
- semantically aligned with the system's procedure

The most useful derivation chain is:

- `ControlStream -> system@link`
- `system@link + inputName -> specific system input`
- `System -> Procedure`
- `Procedure -> matching input definition`
- `ControlStream.samplingFeature@link -> SamplingFeature.sampledFeature@link -> ControlStream.featureOfInterest@link`

As with datastreams, these links should be treated as server materialization of established context rather than independent client-owned references.

## Observed And Controlled Property Mapping

The stream-level property fields are:

- `Datastream.observedProperties`
- `ControlStream.controlledProperties`

Practical interpretation:

- Observed property membership lives at the datastream level
- Controlled property membership lives at the control stream level

But for this project, the stream property sets should also satisfy a stronger consistency rule than the minimum normative model:

- a datastream's `observedProperties` must match the observable properties of the system output identified by `outputName`
- that same set must also remain consistent with the corresponding output definition on the associated procedure
- a control stream's `controlledProperties` must match the controllable properties of the system input identified by `inputName`
- that same set must also remain consistent with the corresponding input definition on the associated procedure

So the stream property lists are not just arbitrary metadata tags. They are the runtime property contract of a concrete system channel.

The Part 2 standard also says the core stream extents and property summaries can be generated by the server from nested observations or commands. For this project, the better interpretation is:

- server-generated values still need to be semantically valid against the attached system channel
- observations cannot introduce properties that contradict the system output or procedure output bound to the datastream
- commands cannot introduce controlled properties that contradict the system input or procedure input bound to the control stream

So the query model should be:

- `System?observedProperty=...`: match if any datastream attached to that system exposes the requested property in `observedProperties`
- `System?controlledProperty=...`: match if any control stream attached to that system exposes the requested property in `controlledProperties`
- `Deployment?observedProperty=...`: match if any deployed system has such a datastream
- `Deployment?controlledProperty=...`: match if any deployed system has such a control stream

This is the cleanest way to interpret the Part 1 filter parameters against the Part 2 stream resources.

## Practical Search Traversal Summary

The most useful search traversals in this project are:

- `System -> Sampling Features -> sampledFeature@link` for FOI matching
- `System -> Datastreams -> observedProperties` for observed-property matching
- `System -> Control Streams -> controlledProperties` for controlled-property matching
- `Datastream -> samplingFeature@link -> sampledFeature@link` for stream-level FOI resolution
- `ControlStream -> samplingFeature@link -> sampledFeature@link` for stream-level FOI resolution
- `Datastream -> outputName -> System output -> Procedure output` for property consistency
- `ControlStream -> inputName -> System input -> Procedure input` for property consistency
- `Deployment -> Deployed Systems -> Sampling Features / Datastreams / Control Streams` for deployment-level matching

That means many of the Part 1 search filters are best understood as graph traversals across related resources rather than as fields directly stored on the Part 1 resource itself.
