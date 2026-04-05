# Overview

## What Is OGC API - Connected Systems?

OGC API - Connected Systems is an API model for describing connected assets and the data or commands that move through them.

At a high level, it separates the domain into two concerns:

- Part 1: feature resources that describe assets and their context
- Part 2: dynamic resources that carry observations, commands, events, and history

The standard models systems as things that can produce data, receive commands, or both. Around those systems, it defines supporting resources such as deployments, procedures, sampling features, datastreams, control streams, and property definitions.

## Source Layout In This Repo

The schema source is organized into three families:

- `schemas/geojson/`: GeoJSON feature representations for Part 1 feature-style resources
- `schemas/sensorml/`: SensorML JSON representations for richer descriptive resources in Part 1
- `schemas/json/`: JSON schemas for Part 2 dynamic-data resources

The OpenAPI documents are split like this:

- Part 1 is consolidated in `schemas/openapi/connected-systems-Part 1 All.yaml`
- Part 2 is split by capability and resource in `schemas/openapi/*.yaml`

## Conceptual Model

The main mental model is:

- `System`: the asset, platform, sensor, actuator, or processing component
- `Deployment`: where and when a system is deployed
- `Procedure`: how a system behaves or what method or configuration it uses
- `Sampling Feature`: the sampled target or sampled portion of a larger feature of interest
- `Property`: the semantic definition of a thing that is observed, asserted, or controlled
- `Datastream`: a channel of observations produced by a system
- `Observation`: one measured or derived result in a datastream
- `Control Stream`: a channel for commands sent to a system
- `Command`: one command instance sent through a control stream
- `System Event`: a time-tagged event about a system
- `System History`: historical revisions of system descriptions over time

## Part 1: Feature Resources

Canonical Part 1 resource roots:

- `/systems`
- `/deployments`
- `/procedures`
- `/samplingFeatures`
- `/properties`

| Resource | Meaning | Supported Representations In This Repo | Media Types Seen In OpenAPI | Schema Files |
| --- | --- | --- | --- | --- |
| `System` | A data-producing and/or command-receiving asset; can also represent platforms, subsystems, sensors, actuators, or processing components | GeoJSON and SensorML | `application/geo+json`, `application/sml+json` | `schemas/geojson/system-bundled.json`, `schemas/sensorml/system-bundled.json` |
| `Deployment` | A description of how a system is deployed in place and time | GeoJSON and SensorML | `application/geo+json`, `application/sml+json` | `schemas/geojson/deployment-bundled.json`, `schemas/sensorml/deployment-bundled.json` |
| `Procedure` | A description of method, behavior, datasheet, or configuration used by a system | GeoJSON and SensorML | `application/geo+json`, `application/sml+json` | `schemas/geojson/procedure-bundled.json`, `schemas/sensorml/procedure-bundled.json` |
| `Sampling Feature` | The sampled entity or sampled portion that links a system to its ultimate feature of interest | GeoJSON only in this repo | `application/geo+json` | `schemas/geojson/samplingFeature-bundled.json` |
| `Property` | A derived or defined property used elsewhere in the API as the semantic definition of observed, asserted, or controlled values | SensorML-style JSON only in this repo | `application/sml+json` | `schemas/sensorml/property-bundled.json` |

## What GeoJSON Means Here

In this repo, the GeoJSON schemas are feature-oriented representations used when a resource is exposed as a spatial feature.

Common traits:

- GeoJSON `Feature` structure
- Spatial `geometry` and optional `bbox`
- Resource metadata carried inside `properties`
- Good fit when map, display, or spatial filtering matters

This is why `System`, `Deployment`, `Procedure`, and `Sampling Feature` all have GeoJSON variants.

## What SensorML Means Here

In this repo, SensorML JSON is the richer descriptive form used for asset and process metadata.

Common traits seen in the schemas:

- Rich identification metadata such as `uniqueId`, `label`, `description`, and `keywords`
- Asset description structures such as `identifiers`, `classifiers`, `contacts`, `documentation`, `characteristics`, and `capabilities`
- SWE Common-style embedded structures for formal data descriptions
- Better fit for detailed asset, procedure, and property semantics than a plain GeoJSON feature wrapper

This is why `System`, `Deployment`, `Procedure`, and `Property` appear in the SensorML schema family.

## Part 2: Dynamic Data Resources

Canonical Part 2 resource roots:

- `/datastreams`
- `/observations`
- `/controlstreams`
- `/commands`
- `/systemEvents`
- `/systems/{systemId}/history`

| Resource | Meaning | Representation In This Repo | Media Types Seen In OpenAPI | Schema Files |
| --- | --- | --- | --- | --- |
| `Datastream` | Metadata for a stream of observations produced by a system | JSON resource with nested schema definitions | `application/json` | `schemas/json/datastream-bundled.json` |
| `Observation` | An individual observation record belonging to a datastream | JSON | `application/json` | `schemas/json/observation-bundled.json` |
| `Control Stream` | Metadata for a stream or channel that accepts commands for a system | JSON resource with nested schema definitions | `application/json` | `schemas/json/controlStream-bundled.json` |
| `Command` | An individual command sent through a control stream | JSON | `application/json` | `schemas/json/command-bundled.json` |
| `System Event` | A time-tagged event describing something that happened to or around a system | JSON | `application/json` | `schemas/json/systemEvent-bundled.json` |
| `System History` | Historical access pattern for prior versions of system descriptions | Reuses system representations rather than a dedicated bundled schema file | `application/geo+json`, `application/sml+json` | No dedicated `systemHistory-bundled.json` file in this repo |

## Important Part 2 Schema Pattern

The most important design pattern in Part 2 is that stream resources carry schemas for their payloads:

- A `Datastream` contains a `schema` definition that explains what its `Observation.result` and optional `Observation.parameters` mean
- A `Control Stream` contains a `schema` definition that explains what its `Command.parameters` and related results mean

So the stream metadata defines the contract, and the individual observations or commands are instances that must conform to that contract.

## Resource-To-Schema Summary

### GeoJSON Family

- `System`
- `Deployment`
- `Procedure`
- `Sampling Feature`

### SensorML Family

- `System`
- `Deployment`
- `Procedure`
- `Property`

### JSON Dynamic-Data Family

- `Datastream`
- `Observation`
- `Control Stream`
- `Command`
- `System Event`
