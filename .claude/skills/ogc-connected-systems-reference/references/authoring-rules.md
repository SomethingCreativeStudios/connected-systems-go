# Authoring Rules

## Purpose

This is the compact creation checklist for this project.

Use it when deciding which fields clients should send, which fields should usually be omitted, and which fields the server should derive.

## Core Rule

Keep three layers separate:

- endpoint path chooses structural parentage
- payload chooses client-authored semantics
- server materializes derived links and operational summary fields

## Part 1 Authoring Rules

### System

Usually author:

- instance identity and descriptive metadata
- `systemKind@link` in GeoJSON or `typeOf` in SensorML
- instance-specific overrides that genuinely belong to the concrete asset

Usually do not author as stable payload truth:

- server navigation links
- inferred child-resource links

### Deployment

Usually author:

- deployment metadata
- `platform@link` or `platform`
- `deployedSystems@link` or `deployedSystems`
- deployment-specific configuration context

Usually do not author as an ad hoc search shortcut:

- derived FOI, property, or stream summaries that should instead come from deployed systems

### Sampling Feature

Usually author:

- the sampling-feature content itself
- `sampledFeature@link`

Do not author:

- parent system identity in the payload as if it can override the creation path

Structural rule:

- create sampling features under `/systems/{systemId}/samplingFeatures`

## Part 2 Container Authoring Rules

### Datastream

Usually author on create:

- `name`
- `description`
- `outputName`
- initial `schema`

Usually omit and let the server derive:

- `system@link`
- `procedure@link`
- `deployment@link`
- `samplingFeature@link`
- `featureOfInterest@link`
- `observedProperties`
- time extents and liveness summaries

Important rule:

- `outputName` is the client-controlled join back to the system and procedure output channel
- the stream schema should stay aligned with that channel's IOComponent shape

### ControlStream

Usually author on create:

- `name`
- `description`
- `inputName`
- initial `schema`

Usually omit and let the server derive:

- `system@link`
- `procedure@link`
- `deployment@link`
- `samplingFeature@link`
- `featureOfInterest@link`
- `controlledProperties`
- time extents and channel-state summaries

Important rule:

- `inputName` is the client-controlled join back to the system and procedure input channel
- the stream schema should stay aligned with that channel's IOComponent shape

## Part 2 Item Authoring Rules

### Observation

Usually author:

- `samplingFeature@id` when targeting a specific sample
- `procedure@link` only when item-level specialization is really needed
- `phenomenonTime`
- `resultTime`
- `parameters`
- `result` or `result@link`

Do not author:

- `id`
- `datastream@id`

Validation rule:

- `parameters` must satisfy the parent datastream `parametersSchema`
- `result` must satisfy the parent datastream `resultSchema`

### Command

Usually author:

- `samplingFeature@id` when targeting a specific sample
- `procedure@link` only when item-level specialization is really needed
- `parameters`

Do not author:

- `id`
- `controlstream@id`
- `issueTime`
- `executionTime`
- `sender`
- `currentStatus`

Validation rule:

- `parameters` must satisfy the parent control-stream `parametersSchema`

### CommandStatus

Usually author only from the processing side:

- `statusCode`
- `reportTime`
- progress or message fields
- partial results when appropriate

Do not treat as client-request payload:

- parent association
- server-issued identifiers

### CommandResult

Usually author only from the processing side:

- inline data
- observation links
- observation-set links
- datastream links
- external result links

Do not treat as free-form user payload:

- parent association
- server-issued identifiers

### SystemEvent

Usually author:

- semantic event content
- descriptive event metadata

Do not author as mutable parentage:

- parent system identity beyond the creation endpoint

## Recommended UI Defaults

For create forms, prefill or expose only the narrow client-owned fields.

Examples:

- datastream form: show `name`, `description`, `outputName`, `schema`
- control-stream form: show `name`, `description`, `inputName`, `schema`
- observation form: show `samplingFeature@id`, times, `parameters`, `result`
- command form: show `samplingFeature@id`, `parameters`

Usually hide or show as read-only:

- server links
- IDs
- summary extents
- operational status fields

## Discouraged Patterns

- trying to move a hierarchical resource by editing payload links
- authoring stream context links independently from the bound system channel
- using a Part 2 schema that does not match the selected Part 1 input or output shape
- treating server summary fields as source-of-truth input fields in the UI

## Fast Checklist

Before submitting a create or update payload, ask:

- is the parent chosen by the URL rather than the body?
- are we sending only client-owned fields?
- are we omitting server-derived links and summary fields?
- does the stream schema still match the selected system or procedure channel?
- are item payloads validated by the parent stream schema rather than by ad hoc UI rules?