# OGC API - Connected Systems (Go)

Go implementation of OGC API - Connected Systems, including:

- Part 1: Feature Resources
- Part 2: Dynamic Data

## Overview

This API provides metadata and dynamic data endpoints for connected systems such as sensors, actuators, platforms, and procedures.

Current implementation includes canonical Connected Systems resources plus dynamic data resources (datastreams, observations, control streams, commands, system events, and system history).

## Implemented Resource Types

Part 1 resources:

- Systems
- Deployments
- Procedures
- Sampling Features
- Properties
- Features (via OGC API - Features collection items)
- Collections

Part 2 resources:

- Datastreams
- Observations
- Control Streams
- Commands
- System Events
- System History

## Conformance

The conformance declaration is available at `GET /conformance`.

Implemented conformance URIs include:

- OGC API - Common: core, landing page, JSON, collections
- OGC API - Features: core, GeoJSON
- OGC API - Connected Systems Part 1: api-common, system, subsystem, deployment, procedure, sampling feature, property, advanced-filtering, GeoJSON
- OGC API - Connected Systems Part 2: api-common, datastream, observation, controlstream, command, system-event, system-history, JSON, create-replace-delete

## API Endpoints

Core:

- `GET /` - Landing page
- `GET /conformance` - Conformance declaration
- `GET /api` - Minimal OpenAPI metadata document

Collections and features:

- `POST /collections`
- `GET /collections`
- `GET /collections/{collectionId}`
- `GET /collections/{collectionId}/items`
- `POST /collections/{collectionId}/items`
- `GET /collections/{collectionId}/items/{featureId}`
- `PUT /collections/{collectionId}/items/{featureId}`
- `DELETE /collections/{collectionId}/items/{featureId}`

Systems and related resources:

- `GET /systems`
- `POST /systems`
- `GET /systems/{id}`
- `PUT /systems/{id}`
- `DELETE /systems/{id}`
- `GET /systems/{id}/subsystems`
- `POST /systems/{id}/subsystems`
- `GET /systems/{id}/deployments`
- `GET /systems/{id}/samplingFeatures`
- `POST /systems/{id}/samplingFeatures`
- `GET /systems/{id}/datastreams`
- `POST /systems/{id}/datastreams`
- `GET /systems/{id}/controlstreams`
- `POST /systems/{id}/controlstreams`
- `GET /systems/{id}/events`
- `POST /systems/{id}/events`
- `GET /systems/{id}/events/{eventId}`
- `PUT /systems/{id}/events/{eventId}`
- `DELETE /systems/{id}/events/{eventId}`
- `GET /systems/{id}/history`
- `GET /systems/{id}/history/{revId}`
- `PUT /systems/{id}/history/{revId}`
- `DELETE /systems/{id}/history/{revId}`

Deployments:

- `GET /deployments`
- `POST /deployments`
- `GET /deployments/{id}`
- `PUT /deployments/{id}`
- `DELETE /deployments/{id}`
- `GET /deployments/{id}/subdeployments`
- `POST /deployments/{id}/subdeployments`

Procedures:

- `GET /procedures`
- `POST /procedures`
- `GET /procedures/{id}`
- `PUT /procedures/{id}`
- `DELETE /procedures/{id}`

Sampling Features:

- `GET /samplingFeatures`
- `GET /samplingFeatures/{id}`
- `PUT /samplingFeatures/{id}`
- `DELETE /samplingFeatures/{id}`

Properties:

- `GET /properties`
- `POST /properties`
- `GET /properties/{id}`
- `PUT /properties/{id}`
- `DELETE /properties/{id}`

Part 2 dynamic data endpoints:

- `GET /datastreams`
- `GET /datastreams/{dataStreamId}`
- `PUT /datastreams/{dataStreamId}`
- `DELETE /datastreams/{dataStreamId}`
- `GET /datastreams/{dataStreamId}/schema`
- `PUT /datastreams/{dataStreamId}/schema`
- `GET /datastreams/{dataStreamId}/observations`
- `POST /datastreams/{dataStreamId}/observations`
- `GET /observations`
- `GET /observations/{obsId}`
- `PUT /observations/{obsId}`
- `DELETE /observations/{obsId}`
- `GET /controlstreams`
- `GET /controlstreams/{controlStreamId}`
- `PUT /controlstreams/{controlStreamId}`
- `DELETE /controlstreams/{controlStreamId}`
- `GET /controlstreams/{controlStreamId}/schema`
- `PUT /controlstreams/{controlStreamId}/schema`
- `GET /controlstreams/{controlStreamId}/commands`
- `POST /controlstreams/{controlStreamId}/commands`
- `GET /commands`
- `GET /commands/{cmdId}`
- `PUT /commands/{cmdId}`
- `DELETE /commands/{cmdId}`
- `GET /systemEvents`

## Content Types

- Part 1 resources primarily support `application/geo+json`
- Properties default to `application/sml+json`
- Part 2 resources use `application/json`

## Query Parameters

Common query parameters across list endpoints:

- `id` - Filter by resource ID or UID
- `q` - Full-text search
- `limit` - Page size
- `offset` - Page offset

Examples of resource-specific filters currently implemented:

- `parent`, `procedure` on systems
- `parent` on deployments
- `system`, `foi`, `observedProperty`, `phenomenonTime`, `resultTime` on datastreams
- `datastream`, `featureOfInterest`, `phenomenonTime`, `resultTime` on observations
- `controlstream`, `status`, `sender`, `issueTime` on commands

## Getting Started

Prerequisites:

- Go 1.24+
- PostgreSQL with PostGIS
- Docker (recommended for local database/test workflows)

Run locally:

```bash
go mod download
cp config.example.yaml config.yaml
make run
```

Build and test:

```bash
make build
make test
make test-coverage
```

## Project Layout

```text
connected-systems-go/
├── cmd/server/               # Server entrypoint
├── internal/api/             # HTTP handlers and router
├── internal/model/           # Domain models, formatters, query params
├── internal/repository/      # GORM repositories and persistence logic
├── internal/config/          # Configuration loading
├── e2e/                      # End-to-end and conformance-oriented tests
├── examples/                 # Example payloads
├── docker-compose.yml        # Local services
└── Makefile                  # Build/test/run commands
```


## References

- OGC API - Connected Systems Part 1: https://docs.ogc.org/is/23-001/23-001.html
- OGC API - Connected Systems Part 2: https://docs.ogc.org/is/24-008/24-008.html
- OGC API - Features: https://ogcapi.ogc.org/features/
- W3C SOSA/SSN: https://www.w3.org/TR/vocab-ssn/
