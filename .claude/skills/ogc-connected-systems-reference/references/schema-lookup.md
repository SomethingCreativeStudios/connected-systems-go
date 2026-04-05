# Schema Lookup

## Purpose

This file copies the highest-signal JSON schema fragments into the skill so they are easier to inspect during future work.

It is intentionally not a full copy of the bundled schemas.

The source of truth remains:

- `schemas/json/datastream-bundled.json`
- `schemas/json/observation-bundled.json`
- `schemas/json/controlStream-bundled.json`
- `schemas/json/command-bundled.json`
- `schemas/json/systemEvent-bundled.json`
- `schemas/openapi/commands-only.yaml` for `CommandStatus` and `CommandResult` component schemas present in OpenAPI but not as standalone bundled JSON files in this repo

## Why Curated Copies Instead Of Full Bundles

The bundled schemas are large and deeply nested.

For agent lookup, the most useful parts are usually:

- the top-level operational fields
- read-only versus write-only markers
- the parent-stream schema contracts
- the fields that connect Part 1 semantics to Part 2 payloads

## Datastream Top-Level Fields

Copied from the local datastream JSON schema.

- `system@link`: link to the system producing the observations, marked `readOnly`
- `outputName`: name of the system output feeding this datastream
- `procedure@link`: procedure used to acquire observations, only provided if all observations share it
- `deployment@link`: deployment during which observations are or were collected, only provided if all observations share it
- `featureOfInterest@link`: ultimate feature of interest, only provided if all observations share it
- `samplingFeature@link`: sampling feature, only provided if all observations share it
- `observedProperties`: list of observed properties included in this datastream

Important project reading:

- `system@link` is explicitly read-only in the schema
- the other contextual links are phrased as conditional summary material, which fits the project's server-derived model for streams

## Datastream Schema Contract

Copied from the local datastream JSON schema's `schema` section.

Important keys:

- `obsFormat`
- `parametersSchema`
- `resultSchema`
- `resultLink`

Copied meaning:

- `parametersSchema`: record schema for observation `parameters`; if omitted, parameters are not included in the datastream
- `resultSchema`: schema for the observation `result`; this describes the observed properties and how they are structured
- `resultLink`: encoding information when the result is provided out-of-band via `result@link`

Operational use:

- nested observations must satisfy this parent schema contract
- after observations exist, incompatible schema changes may be rejected

Schema-family note:

- in the local bundle, the stream schema branch points back into the same SWE component family used by Part 1 SensorML IO definitions
- this is why a Part 1 output or parameter definition can often be reused almost directly as a datastream runtime schema

## Observation Top-Level Fields

Copied from the local observation JSON schema.

- `id`: local observation ID, `readOnly`
- `datastream@id`: local parent datastream ID, `readOnly`
- `samplingFeature@id`: local ID of the sampling feature targeted by the observation
- `procedure@link`: procedure or method used to make the observation
- `phenomenonTime`: time at which the result is a valid estimate of the property value
- `resultTime`: time at which the result was generated
- `parameters`: must be valid according to the parent datastream parameters schema
- `result`: must be valid according to the parent datastream result schema
- `result@link`: link to external result data when the result is out-of-band

Important project reading:

- the parent stream owns the main contract
- the observation can refine target sample and procedure where needed

## ControlStream Top-Level Fields

Copied from the local control-stream JSON schema.

- `system@link`: link to the system receiving the commands, marked `readOnly`
- `inputName`: name of the system control input receiving data from this control stream
- `procedure@link`: procedure used to execute commands, only provided if all commands share it
- `deployment@link`: deployment during which commands are or were received, only provided if all commands share it
- `featureOfInterest@link`: ultimate feature of interest, only provided if all commands share it
- `samplingFeature@link`: sampling feature, only provided if all commands share it
- `controlledProperties`: list of properties controllable through this stream

Important project reading:

- `system@link` is explicitly read-only in the schema
- the other contextual links are conditional summary material, which fits the project's server-derived model for streams

## ControlStream Schema Contract

Copied from the local control-stream JSON schema's `schema` section.

Important keys:

- `commandFormat`
- `parametersSchema`
- `resultSchema`
- `feasibilityResultSchema`

Copied meaning:

- `parametersSchema`: schema for command `parameters`
- `resultSchema`: schema for inline command results, if any
- `feasibilityResultSchema`: schema for feasibility results, if any

Operational use:

- nested commands must satisfy `parametersSchema`
- inline command results must satisfy `resultSchema`
- feasibility results must satisfy `feasibilityResultSchema`
- after commands exist, incompatible schema changes may be rejected

Schema-family note:

- in the local bundle, these control-stream schema branches also reuse the same SWE component family used by Part 1 SensorML inputs and parameters
- this is why a Part 1 input definition can often be reused almost directly as a control-stream runtime schema

## Command Top-Level Fields

Copied from the local command JSON schema.

- `id`: local command ID, `readOnly`
- `controlstream@id`: local parent control-stream ID, `readOnly`
- `samplingFeature@id`: local ID of the sampling feature targeted by the command
- `procedure@link`: procedure or method used to process the command
- `issueTime`: if omitted on creation, the server sets it when the request is received; marked `readOnly`
- `executionTime`: execution period; marked `readOnly`
- `sender`: identifier of the person or entity who submitted the command; marked `readOnly`
- `currentStatus`: current command status; marked `readOnly`
- `parameters`: command payload, validated against the parent control-stream parameters schema

Important project reading:

- command payload ownership is narrow and focused on the actuation request itself
- runtime status and timing are server-side operational state

## Command Status Schema Hotspots

There is no standalone `schemas/json/commandStatus-bundled.json` file in this repo.

The local source of truth is the OpenAPI component schema in `schemas/openapi/commands-only.yaml`.

Important fields:

- `id`: local identifier of the status report, `readOnly`
- `command@id`: parent command identifier, `readOnly`
- `reportTime`: when the report was generated; `readOnly` in the OpenAPI schema
- `statusCode`: one of `PENDING`, `ACCEPTED`, `REJECTED`, `SCHEDULED`, `UPDATED`, `CANCELED`, `EXECUTING`, `FAILED`, `COMPLETED`
- `percentCompletion`: progress percentage from 0 to 100
- `executionTime`: scheduled or actual execution period depending on status
- `message`: human-readable status or error detail
- `results`: partial or complete command results available at the time of the report

Important project reading:

- status reports are operational state publications, not client-owned command payload fields
- `results` can be attached incrementally, so result publication does not have to wait for the terminal state

## Command Result Schema Hotspots

There is no standalone `schemas/json/commandResult-bundled.json` file in this repo.

The local source of truth is the OpenAPI component schema in `schemas/openapi/commands-only.yaml`.

Common fields:

- `id`: local identifier of the result resource, `readOnly`
- `command@id`: parent command identifier, `readOnly`

OpenAPI result variants:

- inline `data`
- `observation@link`
- `observationSet@link`
- `datastream@link` with optional `resultTime`
- `external@link`

Important project reading:

- this is a disjoint-union result model
- a command result is not just one JSON shape; it is a small family of allowed result carriers
- `datastream@link` is the bridge when command execution generates or appends observation streams rather than one-off inline data

## System Event Schema Hotspots

Copied from the local system-event JSON schema and OpenAPI wrapper.

Common event fields visible in the local bundle:

- `id`
- `label`
- `description`
- `definition`: semantic event type URI
- `identifiers`
- `classifiers`
- `contacts`

OpenAPI wrapper notes:

- the OpenAPI `systemEvent` schema wraps the base event model and marks `id` as read-only
- the response form also carries `links`
- collection form is `systemEventCollection` with `items` plus paging or related links

Important project reading:

- system events are structurally system-scoped when created
- event content is descriptive and publisher-authored, but parent system association is structural rather than payload-owned

## Bundle Availability Note

Part 2 resource coverage in local bundled JSON files is:

- bundled: datastream, observation, control stream, command, system event
- not bundled as standalone JSON files here: command status, command result, feasibility

So for status and result parsing, the skill should rely on OpenAPI component schemas unless dedicated bundled files are later added to the repo.

## Fast Lookup Table

Use this table when deciding where a field's meaning comes from.

- `outputName`: selected system output channel for a datastream
- `inputName`: selected system input channel for a control stream
- `observedProperties`: datastream property summary, tied to the selected output channel
- `controlledProperties`: control-stream property summary, tied to the selected input channel
- `parametersSchema`: parent-stream contract for item parameters
- `resultSchema`: parent-stream contract for observation results or inline command results
- `feasibilityResultSchema`: parent control-stream contract for feasibility result payloads
- `samplingFeature@id`: item-level local sample target
- `samplingFeature@link`: stream-level sample context
- `featureOfInterest@link`: ultimate FOI context derived through the sampling-feature chain
- `statusCode`: operational command state, published through status resources
- `results`: status-time result attachments, often partial
- `datastream@link` on a command result: pointer to result observations produced by command execution
- `definition` on a system event: semantic event type URI

## Practical Use

Use this reference when you need a quick answer to questions like:

- is this field client-authored or server-derived?
- which parent schema validates this payload?
- which field joins a Part 2 stream back to a Part 1 system channel?
- should this FOI reference be entered directly or derived through a sampling feature?