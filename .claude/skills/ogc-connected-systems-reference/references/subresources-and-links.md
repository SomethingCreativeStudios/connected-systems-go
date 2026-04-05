# Sub-Resources And Links

## Sub-Resources And Reparenting

The standard treats subsystems and subdeployments as regular resources that are also exposed through parent-scoped subcollections.

The important split is:

- Creation happens through the parent-scoped collection
- Retrieval, replace, update, and delete happen at the canonical root resource URL

## Subsystems

Subsystems are just `System` resources with an additional parent association.

Normative shape from Part 1:

- Create at `/systems/{parentId}/subsystems`
- Canonical resource URL is still `/systems/{id}`
- Parent association is `parentSystem`
- Nested collection is `/systems/{parentId}/subsystems`

Important rule from the standard:

- There is no move or reparent operation for subsystems
- To move a subsystem, the client deletes it from its canonical URL and recreates it under the new parent

Practical interpretation for this project:

- Treat parentage as immutable for subsystem creation
- Parent is chosen at create time only
- Any reparenting request should be modeled as delete plus recreate
- PATCH and PUT on `/systems/{id}` should update the subsystem resource itself, not move it to another parent

## Subdeployments

Subdeployments follow the same structural pattern as subsystems, but for `Deployment` resources.

Normative shape from Part 1:

- Create at `/deployments/{parentId}/subdeployments`
- Canonical resource URL is still `/deployments/{id}`
- Parent association is `parentDeployment`
- Nested collection is `/deployments/{parentId}/subdeployments`

The standard explicitly says subdeployments are created as sub-resources of a parent deployment and then updated and deleted at their canonical URL.

Practical interpretation for this project:

- Use the same no-reparent rule as subsystems
- Parent deployment is chosen at create time
- If a subdeployment must move, delete it and recreate it under the new parent deployment
- Do not model parent reassignment as a PATCH or PUT concern

## Why This Fits The CS Model

This keeps the resource identity model clean:

- canonical URL identifies the resource instance
- parent-scoped POST chooses where that instance is born in the hierarchy
- hierarchical restructuring is an explicit delete and create event, not an in-place mutation of ancestry

That is consistent with the Connected Systems hierarchy model where the nested endpoints represent structural containment, not just a loose tag.

## Composition Versus Aggregation

The subsystem section of the standard makes an important distinction:

- Composition: use nested subsystem creation when the component is structurally part of the parent
- Aggregation: use deployments when components can be mounted or unmounted over time

So for this project:

- subsystem and subdeployment nesting should mean stable structural membership
- temporary mounting or mission-time grouping should be expressed through deployments, not reparenting

## Part 1 Association Links

The standard uses two different kinds of links in Part 1, and it is important not to blur them.

## 1. Server-Provided Navigation Links

These are the links the server returns in the top-level `links` array of the resource representation.

Their role is navigation and discoverability:

- they tell the client what related resource or collection endpoint to follow
- they usually point at canonical resources endpoints or nested resources endpoints
- for association links, the relation type is `ogc-rel:<associationName>`

Examples:

- `ogc-rel:subsystems`
- `ogc-rel:samplingFeatures`
- `ogc-rel:deployments`
- `ogc-rel:implementingSystems`
- `ogc-rel:parentSystem`
- `ogc-rel:parentDeployment`
- `ogc-rel:subdeployments`
- `ogc-rel:sampledFeature`
- `ogc-rel:sampleOf`
- `ogc-rel:datastreams`
- `ogc-rel:controlStreams`

These are distinct from generic web links such as `self`, `canonical`, `service-desc`, `conformance`, and `collections`.

Practical rule:

- If a link exists only so the client can traverse to related resources or related collections, treat it as server-provided

## 2. User-Provided Reference Links

These are association values embedded inside the resource payload itself, usually as `...@link` objects in GeoJSON or specific SensorML properties such as `typeOf`, `platform`, or `deployedSystems`.

Their role is semantic reference, not navigation:

- they express what other resource or feature this resource refers to
- they are part of the resource content
- they are the links a client usually supplies on create and update when declaring an association

Examples in this repo:

- `System` GeoJSON: `systemKind@link`
- `System` SensorML: `typeOf`
- `Deployment` GeoJSON: `platform@link`, `deployedSystems@link`
- `Deployment` SensorML: `platform`, `deployedSystems`
- `Sampling Feature` GeoJSON: `sampledFeature@link`

Practical rule:

- If a link expresses the actual data relationship carried by the resource body, treat it as user-provided or client-controlled input unless your server chooses to derive it

## Part 1 Association Tables And Link Ownership

The cleanest way to read the Part 1 association tables is to ask two questions for every association:

- Is this association represented as a navigable related-resource link in the top-level `links` array?
- Or is it represented inline as part of the resource body itself?

## System Associations

Abstract associations from the standard:

- `systemKind`
- `subsystems`
- `samplingFeatures`
- `deployments`
- `procedures`
- `datastreams`
- `controlstreams`
- `parentSystem` for subsystems

Recommended ownership model:

- User-provided reference: `systemKind`
- Server-provided navigation: `parentSystem`, `subsystems`, `samplingFeatures`, `deployments`, `procedures`, `datastreams`, `controlstreams`

Encoding details in this repo:

- GeoJSON encodes `systemKind` inline as `systemKind@link`
- SensorML encodes `systemKind` inline as `typeOf`
- the other associations are represented as links to related resource endpoints

Expected `ogc-rel:*` values for server-provided navigation links:

- `ogc-rel:parentSystem`
- `ogc-rel:subsystems`
- `ogc-rel:samplingFeatures`
- `ogc-rel:deployments`
- `ogc-rel:procedures`
- `ogc-rel:datastreams`
- `ogc-rel:controlStreams`

## Deployment Associations

Abstract associations from the standard:

- `platform`
- `deployedSystems`
- `subdeployments`
- `featuresOfInterest`
- `samplingFeatures`
- `datastreams`
- `controlstreams`
- `parentDeployment` for subdeployments

Recommended ownership model:

- User-provided references: `platform`, `deployedSystems`
- Server-provided navigation: `parentDeployment`, `subdeployments`, `featuresOfInterest`, `samplingFeatures`, `datastreams`, `controlstreams`

Encoding details in this repo:

- GeoJSON encodes `platform` inline as `platform@link`
- GeoJSON encodes `deployedSystems` inline as `deployedSystems@link`
- SensorML encodes `platform` inline as `platform`
- SensorML encodes `deployedSystems` inline as `deployedSystems`
- the other associations are represented as links to related resource endpoints

Expected `ogc-rel:*` values for server-provided navigation links:

- `ogc-rel:parentDeployment`
- `ogc-rel:subdeployments`
- `ogc-rel:samplingFeatures`
- `ogc-rel:featuresOfInterest`
- `ogc-rel:datastreams`
- `ogc-rel:controlStreams`

## Procedure Associations

Abstract associations from the standard:

- `implementingSystems`

Recommended ownership model:

- Server-provided navigation: `implementingSystems`

Reasoning:

- the procedure does not usually own the membership list in request payloads
- the server can derive this inverse association from systems that reference the procedure

Expected `ogc-rel:*` value:

- `ogc-rel:implementingSystems`

## Sampling Feature Associations

Abstract associations from the standard:

- `parentSystem`
- `sampledFeature`
- `sampleOf`
- `datastreams`
- `controlstreams`

Recommended ownership model:

- User-provided reference: `sampledFeature`
- Server-provided navigation: `parentSystem`, `sampleOf`, `datastreams`, `controlstreams`

Practical note:

- `sampledFeature` is the most important client-supplied semantic link because it ties the sampling feature to the ultimate feature of interest or to a higher-level sampling feature in a chain
- `parentSystem` should be treated as server-controlled because creation under `/systems/{systemId}/samplingFeatures` already establishes that association

Encoding details in this repo:

- GeoJSON encodes `sampledFeature` inline as `sampledFeature@link`
- the remaining associations are represented as top-level links

Expected `ogc-rel:*` values for server-provided navigation links:

- `ogc-rel:parentSystem`
- `ogc-rel:sampleOf`
- `ogc-rel:datastreams`
- `ogc-rel:controlStreams`

Expected inline client-provided reference:

- `sampledFeature@link`

## Property Associations

Part 1 property definitions are different from the feature resources above.

In the Part 1 abstract model, the Property resource table only defines attributes such as `baseProperty`, `objectType`, and `statistic`.

There is no Part 1 association table for Property comparable to the ones for System, Deployment, Procedure, or Sampling Feature.

Practical interpretation:

- treat property relationships as attribute-level semantic references, not as `ogc-rel:*` navigational associations in Part 1

## Practical Link Rules For This Project

### Server-Provided

The server should generate:

- `self` and `canonical`
- all collection navigation links
- all nested-resource navigation links
- all inverse and derived association links exposed through the top-level `links` array
- all `ogc-rel:*` relations used to advertise related resources or related resource collections

### User-Provided

The client should provide the semantic references that are part of the resource body, especially:

- `systemKind@link` or `typeOf`
- `platform@link` or `platform`
- `deployedSystems@link` or `deployedSystems`
- `sampledFeature@link`

## Part 2 Stream Associations

Part 2 defines datastream and control stream association tables that look similar to Part 1 associations, but they should not be interpreted the same way in this project.

Normative association sets from Part 2 are:

- `Datastream`: `system`, `observations`, `procedure`, `deployment`, `samplingFeatures`, `featuresOfInterest`
- `ControlStream`: `system`, `commands`, `procedure`, `deployment`, `samplingFeatures`, `featuresOfInterest`

The standard also requires nested association endpoints for locally hosted sampling features and ultimate features of interest:

- `/datastreams/{id}/samplingFeatures`
- `/datastreams/{id}/featuresOfInterest`
- `/controlstreams/{id}/samplingFeatures`
- `/controlstreams/{id}/featuresOfInterest`

So Part 2 clearly supports server navigation over these associations even when the JSON encoding also exposes inline `...@link` fields.

## Part 2 Inline Links Versus Derived Context

The JSON schemas in this repo expose inline fields such as:

- `system@link`
- `procedure@link`
- `deployment@link`
- `samplingFeature@link`
- `featureOfInterest@link`

However, the schema descriptions for these fields already indicate derivation behavior:

- `system@link` is read-only on both datastreams and control streams
- `procedure@link`, `deployment@link`, `samplingFeature@link`, and `featureOfInterest@link` are only provided when all nested observations or commands share the same corresponding association

For this project, the practical ownership split should be stricter than a generic JSON reading.

### Datastream Ownership Model

Treat these as server-provided on a datastream:

- `system@link`
- `procedure@link`
- `deployment@link`
- `samplingFeature@link`
- `featureOfInterest@link`

Reasoning:

- the stream is structurally attached to a system
- `outputName` selects the operational channel on that system
- procedure and deployment context come from that attached system context
- `samplingFeature@link` is the stream's attached sample context
- `featureOfInterest@link` is derived from the sampling feature's `sampledFeature@link`

Treat these as client-authored or client-selected on a datastream:

- `name`
- `description`
- `outputName`
- schema-defining content supplied at creation time

Treat these as server-maintained summaries on a datastream:

- `observedProperties`
- `phenomenonTime`
- `resultTime`
- often `live`

That aligns with the Part 2 rule that several datastream summary properties can be automatically generated from nested observations.

### ControlStream Ownership Model

Treat these as server-provided on a control stream:

- `system@link`
- `procedure@link`
- `deployment@link`
- `samplingFeature@link`
- `featureOfInterest@link`

Reasoning:

- the stream is structurally attached to a system
- `inputName` selects the operational input channel on that system
- procedure and deployment context come from that attached system context
- `samplingFeature@link` is the stream's attached actuation target sample context
- `featureOfInterest@link` is derived from the sampling feature's `sampledFeature@link`

Treat these as client-authored or client-selected on a control stream:

- `name`
- `description`
- `inputName`
- schema-defining content supplied at creation time

Treat these as server-maintained summaries on a control stream:

- `controlledProperties`
- `issueTime`
- `executionTime`
- often `live`
- often `async`

That aligns with the Part 2 model where the control stream is a container view over commands received on a single system channel.

## Feature-Of-Interest Derivation Rule For Streams

The cleanest project rule for stream FOI materialization is:

- `samplingFeature@link` is the immediate stream-level feature context
- `featureOfInterest@link` is the ultimate feature of interest reached through that sampling feature
- in practice, `featureOfInterest@link` should be derived from `samplingFeature@link -> sampledFeature@link`

So if the sampling feature changes, the stream's ultimate FOI link should be regenerated by the server rather than treated as a separate user-managed reference.

This keeps stream FOI semantics aligned with the broader Part 1 rule that sampling features are the bridge from systems to ultimate features of interest.

## Part 2 Practical Rules For This Project

When implementing datastreams and control streams, use this ownership model:

- client chooses the system-scoped channel identity with `outputName` or `inputName`
- server materializes stream association links from the attached system and sampling-feature context
- server derives `featureOfInterest@link` from the linked sampling feature's `sampledFeature@link`
- server exposes nested association endpoints for sampling features and features of interest
- server rejects stream states whose property sets conflict with the attached system channel or the linked procedure definition

This is the main place where Part 2 intentionally bends the simpler Part 1 intuition that inline `...@link` fields are usually client-owned semantic references.

### Parent Links

For this project, parentage should not be user-editable through inline payload mutation.

That means:

- subsystem parent comes from POST target `/systems/{parentId}/subsystems`
- subdeployment parent comes from POST target `/deployments/{parentId}/subdeployments`
- sampling-feature parent comes from POST target `/systems/{systemId}/samplingFeatures`
- parent navigation links returned later are server materialization of that structural relationship
