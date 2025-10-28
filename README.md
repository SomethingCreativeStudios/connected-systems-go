WORK IN PROGRESS

Mostly IGNORE for now

# OGC Connected Systems API - Part 1: Feature Resources

Go implementation of the OGC API - Connected Systems Standard (Part 1).

## Overview

This API provides access to Connected Systems metadata and feature resources following the OGC API - Connected Systems - Part 1: Feature Resources Standard (OGC 23-001).

The OGC API - Connected Systems Standard defines API building blocks for interacting with Connected Systems - any kind of system that can transmit data via communication networks, including sensors, actuators, platforms, robots, drones, and more.

## Features

This implementation supports the following resource types:

- **Systems**: Sensors, actuators, samplers, platforms, and other observing systems
- **Deployments**: Deployment descriptions of systems for specific purposes
- **Procedures**: Datasheets, methodologies, and system type descriptions
- **Sampling Features**: Sampling strategies and geometries
- **Properties**: Observable and controllable property definitions

## Standards Compliance

Implements the following conformance classes:

- Common (Requirements Class)
- System Features
- Subsystems
- Deployment Features
- Procedure Features
- Sampling Features
- Property Definitions
- Advanced Filtering
- GeoJSON Format
- SensorML Format (optional)

Based on:
- OGC API - Features - Part 1: Core
- OGC API - Common - Part 1: Core
- W3C Semantic Sensor Network Ontology (SOSA/SSN)
- OGC SensorML 3.0

## API Structure

### Canonical Endpoints

- `GET /` - Landing page
- `GET /conformance` - Conformance declaration
- `GET /collections` - Available collections
- `GET /systems` - All systems
- `GET /systems/{id}` - Specific system
- `GET /systems/{id}/subsystems` - System subsystems
- `GET /deployments` - All deployments
- `GET /deployments/{id}` - Specific deployment
- `GET /procedures` - All procedures
- `GET /procedures/{id}` - Specific procedure
- `GET /samplingFeatures` - All sampling features
- `GET /samplingFeatures/{id}` - Specific sampling feature
- `GET /properties` - All properties
- `GET /properties/{id}` - Specific property

### Query Parameters

Common filters:
- `id` - Filter by resource ID or UID
- `bbox` - Bounding box filter
- `datetime` - Time filter
- `limit` - Result limit
- `q` - Full-text search

Resource-specific filters:
- `parent` - Filter by parent system/deployment
- `procedure` - Filter systems by procedure
- `foi` - Filter by feature of interest
- `observedProperty` - Filter by observed property
- `controlledProperty` - Filter by controlled property
- `geom` - WKT geometry filter
- `recursive` - Include nested subsystems

## Technology Stack

- Go 1.21+
- Chi Router
- PostgreSQL with PostGIS
- GORM ORM
- Uber Zap Logger
- Viper Configuration

## Project Structure

```
connected-systems-go/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── model/          # Domain models (SOSA/SSN concepts)
│   ├── repository/     # Data access layer
│   ├── service/        # Business logic
│   └── config/         # Configuration management
├── pkg/
│   ├── geojson/        # GeoJSON encoding/decoding
│   ├── sensorml/       # SensorML encoding/decoding
│   └── filter/         # Query parameter parsing
├── migrations/         # Database migrations
├── docs/              # API documentation
└── test/              # Integration tests
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+ with PostGIS extension
- Docker (optional, for development)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/connected-systems-go
cd connected-systems-go

# Install dependencies
go mod download

# Set up configuration
cp config.example.yaml config.yaml

# Run database migrations
make migrate

# Start the server
make run
```

### Configuration

Edit `config.yaml`:

```yaml
server:
  port: 8080
  host: localhost

database:
  host: localhost
  port: 5432
  name: connected_systems
  user: postgres
  password: password

api:
  base_url: http://localhost:8080
  title: "OGC Connected Systems API"
  version: "1.0.0"
```

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Build
make build

# Run with hot reload
make watch
```

## API Documentation

API documentation is available at:
- OpenAPI 3.0 Spec: `/api` (application/vnd.oai.openapi+json;version=3.0)
- Swagger UI: `/docs`

## Contributing

Contributions are welcome! Please read our contributing guidelines.

## License

This project is licensed under the MIT License - see LICENSE file for details.

## References

- [OGC API - Connected Systems - Part 1 Standard](https://docs.ogc.org/is/23-001/23-001.html)
- [OGC API - Features Standard](https://ogcapi.ogc.org/features/)
- [W3C SOSA/SSN Ontology](https://www.w3.org/TR/vocab-ssn/)
- [OGC SensorML](https://www.ogc.org/standards/sensorml)
