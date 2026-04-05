# Navigation

## Practical Reading Of The Standard In This Repo

The shortest correct interpretation is:

- Part 1 tells you what the connected assets and supporting feature resources are
- Part 2 tells you what flows through those assets over time
- GeoJSON is the spatial feature representation
- SensorML is the richer descriptive representation for systems, processes, and properties
- JSON schemas in `schemas/json/` define the dynamic-data resources and their stream payload contracts

## Endpoint Flow And Navigation

There are two navigation styles in this API:

- Canonical resource roots such as `/systems`, `/deployments`, `/datastreams`, `/controlstreams`
- Parent-scoped subcollections such as `/systems/{systemId}/samplingFeatures` or `/controlstreams/{controlStreamId}/commands`

The canonical roots are the stable resource identities. The scoped endpoints are the practical traversal paths.

## Part 1 Discovery Flow

The broad discovery path is:

- Landing page: `/`
- Conformance: `/conformance`
- Generic collections: `/collections`, `/collections/{collectionId}`, `/collections/{collectionId}/items`
- Canonical Part 1 resources: `/systems`, `/deployments`, `/procedures`, `/samplingFeatures`, `/properties`

For actual Connected Systems navigation, the canonical Part 1 roots are usually more useful than generic collections.

## System-Centered Navigation Flow

The main resource graph in practice is system-centered:

- `/systems`
- `/systems/{systemId}`
- `/systems/{systemId}/subsystems`
- `/systems/{systemId}/deployments`
- `/systems/{systemId}/samplingFeatures`
- `/systems/{systemId}/datastreams`
- `/systems/{systemId}/controlstreams`
- `/systems/{systemId}/events`
- `/systems/{systemId}/history`

Practical reading:

- `System` is the anchor resource
- Descriptive context hangs off the system through procedures, deployments, and sampling features
- Operational context hangs off the system through datastreams and control streams
- Time-based context hangs off the system through events and history

## Procedure Navigation Flow

Procedures are canonical resources, but not exposed as subcollections under systems in this OpenAPI.

The navigation pattern is:

- Find the system
- Read the system's procedure reference
- Resolve the referenced procedure at `/procedures/{procedureId}` or equivalent canonical procedure URL

Canonical procedure endpoints:

- `/procedures`
- `/procedures/{procedureId}`

## Sampling Feature Navigation Flow

Sampling features have both a canonical root and a system-scoped creation and association path:

- `/samplingFeatures`
- `/samplingFeatures/{featureId}`
- `/systems/{systemId}/samplingFeatures`

Important practical rule from the OpenAPI:

- Sampling features are always created under `/systems/{systemId}/samplingFeatures`
- The canonical `/samplingFeatures` endpoint is for listing and searching all sampling features and resolving an individual one by ID

So the navigation flow is:

- Start from a system
- Traverse to `/systems/{systemId}/samplingFeatures`
- Resolve each sampling feature's `sampledFeature@link` as the ultimate feature of interest

## Deployment Navigation Flow

Deployment endpoints are:

- `/deployments`
- `/deployments/{deploymentId}`
- `/deployments/{deploymentId}/subdeployments`
- `/systems/{systemId}/deployments`

Important nuance:

- The API exposes deployments related to a system via `/systems/{systemId}/deployments`
- The deployment resource itself carries deployed-system relationships in the resource body via `deployedSystems` or `deployedSystems@link`
- There is no dedicated `/deployments/{deploymentId}/systems` endpoint in the OpenAPI shown here

So deployment-to-system traversal is partly endpoint-based and partly representation-based.

## Datastream Navigation Flow

Datastream endpoints are:

- `/datastreams`
- `/systems/{systemId}/datastreams`
- `/datastreams/{dataStreamId}`
- `/datastreams/{dataStreamId}/schema`
- `/datastreams/{dataStreamId}/observations`
- `/observations`
- `/observations/{obsId}`

Practical flow:

- Start from a system
- List its datastreams at `/systems/{systemId}/datastreams`
- Resolve one datastream at `/datastreams/{dataStreamId}`
- Read its payload contract at `/datastreams/{dataStreamId}/schema`
- Traverse instance data at `/datastreams/{dataStreamId}/observations`
- Resolve a single observation at `/observations/{obsId}`

This means `Datastream` is the metadata and contract, while `Observation` is the instance data.

## Control Stream Navigation Flow

Control stream endpoints are:

- `/controlstreams`
- `/systems/{systemId}/controlstreams`
- `/controlstreams/{controlStreamId}`
- `/controlstreams/{controlStreamId}/schema`
- `/controlstreams/{controlStreamId}/commands`
- `/commands`
- `/commands/{cmdId}`
- `/commands/{cmdId}/status`
- `/commands/{cmdId}/result`

Practical flow:

- Start from a system
- List its control streams at `/systems/{systemId}/controlstreams`
- Resolve one control stream at `/controlstreams/{controlStreamId}`
- Read its command contract at `/controlstreams/{controlStreamId}/schema`
- Traverse submitted commands at `/controlstreams/{controlStreamId}/commands`
- For a single command, inspect `/commands/{cmdId}`, `/commands/{cmdId}/status`, and `/commands/{cmdId}/result`

This means `Control Stream` is the command channel definition, while `Command` is the task or message instance moving through it.

## Events And History Flow

System events and history extend the system graph:

- `/systems/{systemId}/events`
- `/systems/{systemId}/events/{eventId}`
- `/systems/{systemId}/history`
- `/systems/{systemId}/history/{revId}`

Practical meaning:

- `events` are operational notifications about the system over time
- `history` is versioned descriptive state of the system itself
