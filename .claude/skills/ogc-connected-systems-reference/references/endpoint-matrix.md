# Endpoint Matrix

## Purpose

This is a compact map of where resources are created, where they are retrieved canonically, and where replace or delete operations happen.

Use it as a quick lookup when the full OpenAPI files feel too wide.

## Reading Guide

- `Create` means the structural POST target
- `Canonical GET` means the stable single-resource URL
- `Update/Delete` means the URL used for replace or delete after creation
- `Nested Read` means the parent-scoped collection or item path commonly used for traversal
- `Schema` means a parent schema endpoint for stream resources when applicable

## Part 1 Matrix

| Resource | Create | Canonical GET | Update/Delete | Nested Read | Schema |
| --- | --- | --- | --- | --- | --- |
| `System` | `/systems` | `/systems/{systemId}` | `/systems/{systemId}` | `/systems/{systemId}/subsystems` | none |
| `Subsystem` | `/systems/{systemId}/subsystems` | `/systems/{subsystemId}` | `/systems/{subsystemId}` | `/systems/{systemId}/subsystems` | none |
| `Deployment` | `/deployments` | `/deployments/{deploymentId}` | `/deployments/{deploymentId}` | `/systems/{systemId}/deployments` | none |
| `Subdeployment` | `/deployments/{deploymentId}/subdeployments` | `/deployments/{subdeploymentId}` | `/deployments/{subdeploymentId}` | `/deployments/{deploymentId}/subdeployments` | none |
| `Procedure` | `/procedures` | `/procedures/{procedureId}` | `/procedures/{procedureId}` | inverse traversal from systems | none |
| `SamplingFeature` | `/systems/{systemId}/samplingFeatures` | `/samplingFeatures/{featureId}` | `/samplingFeatures/{featureId}` | `/systems/{systemId}/samplingFeatures` | none |
| `Property` | `/properties` | `/properties/{propId}` | `/properties/{propId}` | none | none |

## Part 1 Structural Notes

- subsystems, subdeployments, and sampling features are structurally parent-scoped at create time
- canonical URLs own later lifecycle operations
- this project treats parent reassignment as delete-plus-recreate, not in-place mutation

## Part 2 Matrix

| Resource | Create | Canonical GET | Update/Delete | Nested Read | Schema |
| --- | --- | --- | --- | --- | --- |
| `Datastream` | `/systems/{systemId}/datastreams` | `/datastreams/{dataStreamId}` | `/datastreams/{dataStreamId}` | `/systems/{systemId}/datastreams`, `/deployments/{deploymentId}/datastreams` in the standard | `/datastreams/{dataStreamId}/schema` |
| `Observation` | `/datastreams/{dataStreamId}/observations` | `/observations/{obsId}` | `/observations/{obsId}` | `/datastreams/{dataStreamId}/observations` | parent datastream schema |
| `ControlStream` | `/systems/{systemId}/controlstreams` | `/controlstreams/{controlStreamId}` | `/controlstreams/{controlStreamId}` | `/systems/{systemId}/controlstreams`, `/deployments/{deploymentId}/controlstreams` in the standard | `/controlstreams/{controlStreamId}/schema` |
| `Command` | `/controlstreams/{controlStreamId}/commands` | `/commands/{cmdId}` | `/commands/{cmdId}` | `/controlstreams/{controlStreamId}/commands` | parent control-stream schema |
| `CommandStatus` | `/commands/{cmdId}/status` | implementation commonly uses `/commands/{cmdId}/status/{statusId}` | `/commands/{cmdId}/status/{statusId}` | `/commands/{cmdId}/status` | none |
| `CommandResult` | `/commands/{cmdId}/result` | implementation commonly uses `/commands/{cmdId}/result/{resultId}` | `/commands/{cmdId}/result/{resultId}` | `/commands/{cmdId}/result` | parent control-stream `resultSchema` or `feasibilityResultSchema` |
| `SystemEvent` | `/systems/{systemId}/events` | standard root is `/systemEvents/{eventId}`; local split OpenAPI emphasizes nested system path | `/systems/{systemId}/events/{eventId}` and standard root form | `/systems/{systemId}/events`, `/systemEvents` | none |

## Part 2 Association Endpoints

The Part 2 standard also defines related-resource endpoints for stream context:

- `/datastreams/{id}/samplingFeatures`
- `/datastreams/{id}/featuresOfInterest`
- `/controlstreams/{id}/samplingFeatures`
- `/controlstreams/{id}/featuresOfInterest`

These are important because they show the standard treats stream context as navigable server associations, even when inline `...@link` fields also exist.

## Part 2 Contract Notes

- observations validate against the parent datastream schema
- commands validate against the parent control-stream schema
- inline command results validate against the parent control-stream result schema
- schema-changing stream updates may be rejected once child items exist
- stream delete operations can require cascade semantics when child items exist

## Feasibility Note

The Part 2 standard defines feasibility resources as command-like resources scoped under a control stream, with nested status and result resources.

However, this workspace's local OpenAPI split files do not currently expose dedicated `/feasibility` paths as first-class route definitions the way they expose datastreams, commands, and events.

Practical implication:

- treat feasibility as normatively part of Part 2
- but use the standard text rather than the local split OpenAPI as the main source of truth for feasibility endpoint details in this repo

## Quick Rules

- if the resource is hierarchical, create through the parent and mutate through the canonical item URL
- if the resource is a stream child, validate payloads against the parent stream contract
- if the resource is a stream container, look for a `/schema` endpoint before trying to understand its item payloads