# OGC API - Connected Systems (Go)

Go implementation of OGC API - Connected Systems, including:

- Part 1: Feature Resources
- Part 2: Dynamic Data
- MQTT publish/subscribe: Resource Data Messages and Resource Events

## Overview

For a guided explanation of the server, start with [Architecture and resource flow](docs/architecture/README.md): diagrams, a source-reading tour, all registered endpoints, serialization, MQTT, and a verified local walkthrough.

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
- `GET /asyncapi` - Enabled Pub/Sub MQTT channels and event types

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

## MQTT Publish/Subscribe

Part 3 uses the [pinned current draft](docs/standards/23-003r0.html). Run the API
with its embedded Mochi endpoint: it validates client creates before distribution.
HTTP, MQTT and WebSockets run in one image; no external broker plugin is needed. `GET /asyncapi` describes the enabled capabilities in AsyncAPI
2.6 and is linked from the landing page. `/conformance` includes draft-derived,
provisional Part 3 class identifiers; the draft's advertisement conventions remain
unfinished. See the [implementation and evidence ledger](docs/part3-mqtt.md).

Topics start with the stable node ID and follow complete ownership hierarchies:

- Resource Event: `node1/systems/{systemId}/datastreams/{datastreamId}`
- Observation batch: `node1/systems/{systemId}/datastreams/{datastreamId}/observations`
- Observation data: `node1/systems/{systemId}/datastreams/{datastreamId}/observations:data`
- System data: `node1/systems:data`; SensorML: `node1/systems:data/sml-json`
- Subsystem data: `node1/systems/{systemId}/subsystems:data`

Identifiers use canonical percent encoding of every byte outside URI unreserved
characters, including literal `%`. Events use `org.ogc.api.csapi.<resource>.<operation>`
and absolute HTTP `source`, `subject`, and `resourceparent` identities. There are
no additional-topic copies. Observation and command lifecycle notifications use
UTC aligned one-minute batches by default; batching requires Resource Events.

Clients can create systems/subsystems, datastreams, controlstreams, observations,
commands, command statuses/results and System Events. Each payload is one native
resource object. MQTT 5 requires matching Content Type. MQTT 3.1.1 derives the
representation from the topic. Systems default to GeoJSON and also offer SensorML;
Part 2 uses JSON. Unsupported format topics are rejected.

MQTT 5 clients can supply one `idempotency-key` User Property (1–128 UTF-8 bytes).
A matching retry returns the original committed result without another resource,
history revision, lifecycle event or batch contribution. The durable key is scoped
to the API root and authenticated username, survives resource deletion, and does
not expire. Client ID is excluded. Anonymous local clients share a principal and
should use globally unique keys. Different input with the same key is rejected.
Keyless messages are separate creates and can create duplicates.

Client creates accept QoS 0 or 1; MQTT 5 with a key also accepts QoS 2. Subscriptions
accept QoS 0–2. Server publications use QoS 1. MQTT acknowledgement, committed
operation acceptance and command execution status are separate: consumers must
handle repeat delivery of the same command ID. Retention is disabled, resources
are limited to 10 MiB, and API publication is best effort without an outbox.

Compose supplies `node1`, public connection addresses and anonymous loopback
access. MQTT sessions and offline queues live in the API's separate data volume;
resource creation and durable idempotency remain in PostgreSQL. For authenticated
deployments set `MQTT_ALLOW_ANONYMOUS=false` and `MQTT_USERS_FILE` to a read-only
JSON array of `{"username":"cs-viewer","password_hash":"<bcrypt hash>"}` entries.
Generate a hash with `server mqtt hash-password` using standard input. There is
no privileged network publisher account. Provide TLS at the deployment boundary.
See [the dev-server Compose example](examples/dev-server.compose.yaml) for a
Traefik deployment alongside an existing HA/Zigbee2MQTT Mosquitto broker.
The standalone settings are in [config.example.yaml](config.example.yaml).
Downstream consumers require the changes in the [client migration checklist](docs/part3-client-migration.md).

## Query Parameters

Common query parameters across list endpoints:

- `id` - Filter by resource ID or UID
- `q` - Full-text search
- `limit` - Page size
- `cursor` - Opaque continuation token from a `next` or `prev` link

Examples of resource-specific filters currently implemented:

- `parent`, `procedure` on systems
- `parent` on deployments
- `system`, `foi`, `observedProperty`, `phenomenonTime`, `resultTime` on datastreams
- `datastream`, `featureOfInterest`, `phenomenonTime`, `resultTime` on observations
- `controlstream`, `status`, `sender`, `issueTime` on commands

## Getting Started

Prerequisites:

- Go 1.26.3+ (the `toolchain` line in `go.mod` downloads Go 1.27.2 if needed)
- PostgreSQL with PostGIS and TimescaleDB
- Docker (recommended for local database/test workflows)

Run locally:

```bash
go mod download
cp config.example.yaml config.yaml
make run
```

Database migrations (goose, embedded in the binary) run at startup while
`database.auto_migrate` is true, the default. To run them as a separate step,
set it to false and run `make migrate` (or `server migrate`, or
`docker run <image> migrate`) before starting the server.

## Seed realistic demo data

The HTTP-only seeder creates an additive, namespaced Part 1/2 resource graph
against a running API. Copy and adjust
[`scripts/seed-connected-systems/config.example.yaml`](scripts/seed-connected-systems/config.example.yaml),
then run:

```bash
go run ./scripts/seed-connected-systems -config scripts/seed-connected-systems/config.example.yaml
```

Use `mode: observe` in that file to continuously post schema-correct random
observations to existing compatible datastreams.

Build and test:

```bash
make build
make test
make test-coverage
make lint   # golangci-lint, pinned in the Makefile
make vuln   # govulncheck
```

## Local TimescaleDB database

Start the API with its embedded MQTT endpoint and TimescaleDB:

```bash
docker compose up --build -d
```

The API is at `http://localhost:8070` (Swagger UI: `/api/ui/index.html`).
PostgreSQL is at `localhost:5433` with database `connected_systems` and
username/password `postgres`. MQTT is at `localhost:1883`, with WebSockets
at `ws://localhost:9001/mqtt`. Published ports bind to `127.0.0.1`. The embedded
endpoint allows anonymous local connections. Authenticated Compose runs can mount
a users file through `MQTT_USERS_DIR` and select it with
`MQTT_USERS_FILE=/run/csapi-mqtt/users.json`.

Set `API_PORT`, `DATABASE_PORT`, `MQTT_PORT`, `MQTT_WEBSOCKET_PORT`, or
`VIEWER_PORT` in your shell or `.env` to change host ports. API links follow
`API_PORT`; `API_BASE_URL` can override the advertised URL separately.
If another broker already uses ports 1883/9001, select different host ports with
`MQTT_PORT` and `MQTT_WEBSOCKET_PORT`. Keep shared HA/Zigbee2MQTT brokers running.

The optional viewer uses the published image, which requires access to
`docker.monogatari.dev` unless it is already cached. Enable it with:

```bash
docker compose --profile viewer up --build -d
```

Open `http://localhost:8081`; the viewer calls the localhost API directly.
`VIEWER_IMAGE` can select a different or locally built viewer image.

For a Go process running on the host, start only the dependencies with
`docker compose up -d db`, then set `DATABASE_PORT=5433` and
`API_BASE_URL` to match that process's listening port when running the server.
Set `MQTT_ENABLED=true` and configure authentication to start that process's
embedded MQTT listeners as well.
Stop the stack with `docker compose --profile viewer down`; named volumes
retain database and MQTT data.

`docker-compose.yml` runs PostgreSQL 18 with TimescaleDB and PostGIS. On
startup, the server enables TimescaleDB, PostGIS, unaccent, and pg_trgm and creates `observations` as a
hypertable partitioned by `result_time` in seven-day chunks. The Compose
database uses a new `timescale_data` volume; an existing `postgres_data`
development volume is intentionally left untouched and is not migrated.

Pull the published Docker image:

```bash
docker pull ericlo417/connected-systems:v1.0.1
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
