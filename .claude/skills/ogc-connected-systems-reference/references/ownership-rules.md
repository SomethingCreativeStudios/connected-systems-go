# Ownership Rules

## Purpose

This reference captures who chooses, who derives, and who is allowed to mutate key Connected Systems fields in this project.

The goal is to keep three concerns separate:

- structural parentage set by the endpoint used for creation
- semantic references authored by the client
- derived or operational fields materialized by the server

## Core Rule

When a resource is created through a parent-scoped endpoint, that endpoint establishes the structural parent association.

For this project, that means parentage is not later reassigned by changing inline payload fields.

## Parent-Scoped Create Rules

Part 1 structural creates:

- `POST /systems/{parentId}/subsystems` creates a subsystem under a parent system
- `POST /deployments/{parentId}/subdeployments` creates a subdeployment under a parent deployment
- `POST /systems/{systemId}/samplingFeatures` creates a sampling feature under a parent system

Part 2 structural creates:

- `POST /systems/{systemId}/datastreams` creates a datastream under a system
- `POST /datastreams/{dataStreamId}/observations` creates observations under a datastream
- `POST /systems/{systemId}/controlstreams` creates a control stream under a system
- `POST /controlstreams/{controlStreamId}/commands` creates commands under a control stream
- `POST /commands/{cmdId}/status` creates command status resources under a command
- `POST /commands/{cmdId}/result` creates command result resources under a command
- `POST /systems/{systemId}/events` creates system events under a system

Normative Part 2 feasibility pattern:

- feasibility behaves like a command-like resource scoped under a control stream, with nested status and result resources

## Canonical Update And Delete Rules

Once created, the canonical resource URL owns replace, update, and delete operations.

Examples:

- subsystems are updated and deleted through `/systems/{id}`
- subdeployments are updated and deleted through `/deployments/{id}`
- datastreams are updated and deleted through `/datastreams/{id}`
- observations are updated and deleted through `/observations/{id}`
- control streams are updated and deleted through `/controlstreams/{id}`
- commands are updated and deleted through `/commands/{id}`
- system events are updated and deleted through `/systemEvents/{id}` in the standard, while the local split OpenAPI emphasizes the nested `/systems/{systemId}/events/{eventId}` form

Practical interpretation for this project:

- canonical URLs own lifecycle mutation
- parent-scoped URLs own creation and relationship establishment
- no PATCH or PUT should be interpreted as a move operation

## Part 1 Field Ownership

### System

Client-authored:

- `systemKind@link` in GeoJSON
- `typeOf` in SensorML
- normal descriptive metadata that belongs to the system instance

Server-derived or server-provided:

- parent-system links
- subsystem navigation links
- sampling-feature, deployment, datastream, and control-stream navigation links

### Deployment

Client-authored:

- `platform@link` or `platform`
- `deployedSystems@link` or `deployedSystems`
- deployment metadata and configuration content

Server-derived or server-provided:

- parent-deployment links
- subdeployment navigation links
- derived navigation to sampling features, features of interest, datastreams, and control streams

### Sampling Feature

Client-authored:

- `sampledFeature@link`

Server-derived or server-provided:

- parent-system association
- inverse `sampleOf` navigation
- datastream and control-stream navigation

Structural rule:

- the parent system comes from `POST /systems/{systemId}/samplingFeatures`
- the parent should not be edited by payload mutation later

## Part 2 Container Ownership

### Datastream

Client-authored or client-selected:

- `name`
- `description`
- `outputName`
- the initial `schema` payload at create time

Server-derived or server-maintained:

- `id`
- `system@link`
- `procedure@link` in this project
- `deployment@link` in this project
- `samplingFeature@link` in this project
- `featureOfInterest@link` in this project
- `observedProperties`
- `phenomenonTime`
- `resultTime`
- often `live`

Reasoning:

- `outputName` is the client-selected bind point to a system output
- once bound, the server can materialize the stream's broader context from the system, deployment, sampling feature, and nested observations

Schema rule:

- the datastream schema defines the contract that nested observations must satisfy
- once observations exist, schema-changing updates may be rejected with conflict semantics

### ControlStream

Client-authored or client-selected:

- `name`
- `description`
- `inputName`
- the initial `schema` payload at create time

Server-derived or server-maintained:

- `id`
- `system@link`
- `procedure@link` in this project
- `deployment@link` in this project
- `samplingFeature@link` in this project
- `featureOfInterest@link` in this project
- `controlledProperties`
- `issueTime`
- `executionTime`
- often `live`
- often `async`

Reasoning:

- `inputName` is the client-selected bind point to a system input
- once bound, the server can materialize the stream's broader context from the system, deployment, sampling feature, and nested commands

Schema rule:

- the control-stream schema defines the contract that nested commands and command results must satisfy
- once commands exist, schema-changing updates may be rejected with conflict semantics

## Part 2 Item Ownership

### Observation

Client-authored:

- `samplingFeature@id` when the observation targets a specific sample within the datastream context
- `procedure@link` when an observation needs a more specific procedure than the datastream-level context
- `phenomenonTime`
- `resultTime`
- `parameters`
- `result` or `result@link`

Server-derived or server-maintained:

- `id`
- `datastream@id`

Contract rule:

- `parameters` must validate against the parent datastream `parametersSchema`
- `result` must validate against the parent datastream `resultSchema`

### Command

Client-authored:

- `samplingFeature@id` when the command targets a specific sample within the control-stream context
- `procedure@link` when a command needs a more specific execution method than the control-stream-level context
- `parameters`

Server-derived or server-maintained:

- `id`
- `controlstream@id`
- `issueTime`
- `executionTime`
- `sender`
- `currentStatus`

Contract rule:

- `parameters` must validate against the parent control-stream `parametersSchema`

### CommandStatus

Client-authored when published by a processing component:

- `statusCode`
- `reportTime`
- progress metadata such as message or execution estimates

Server-derived or structural:

- `id`
- parent command association

### CommandResult

Client-authored when published by a processing component:

- inline result payloads
- observation or datastream references produced by command execution

Server-derived or structural:

- `id`
- parent command association

Contract rule:

- inline command result content must validate against the parent control-stream `resultSchema`
- feasibility result content must validate against the parent control-stream `feasibilityResultSchema`

### SystemEvent

Client-authored or publisher-authored:

- event metadata such as `type`, `eventTime`, `message`, and descriptive fields

Server-derived or structural:

- `id`
- parent system association

## Project-Level Strictness Rules

The project adds these stricter interpretations on top of the normative model:

- parentage is create-time only for hierarchical resources
- stream context links are server materialized from system and sampling-feature context
- `featureOfInterest@link` on streams is derived from `samplingFeature@link -> sampledFeature@link`
- stream property sets must stay consistent with the selected system channel and linked procedure definition

## Why This Split Matters

Without a clear ownership model, a client can try to mutate fields that really belong to a different layer of the model.

This reference keeps the layers stable:

- endpoint establishes structure
- payload establishes client-authored semantics where appropriate
- server materializes derived navigation and operational summary state