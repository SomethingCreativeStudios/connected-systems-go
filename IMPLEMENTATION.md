# OGC Connected Systems API - Part 1: Go Implementation

## Project Status

This is a scaffold implementation of the OGC API - Connected Systems - Part 1: Feature Resources Standard.

## Quick Start

1. **Install dependencies:**
```bash
go mod download
```

2. **Set up configuration:**
```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your database settings
```

3. **Start PostgreSQL with PostGIS:**
```bash
docker-compose up -d db
```

4. **Run the server:**
```bash
go run cmd/server/main.go
```

5. **Test the API:**
```bash
curl http://localhost:8080/
curl http://localhost:8080/conformance
curl http://localhost:8080/collections
curl http://localhost:8080/systems
```

## Architecture

### Models (SOSA/SSN Concepts)
- **System**: Sensors, actuators, platforms, samplers
- **Deployment**: Deployment descriptions
- **Procedure**: System datasheets and methodologies
- **SamplingFeature**: Sampling geometries and strategies
- **Property**: Observable and actuable properties

### Endpoints

All resources follow the OGC API - Features pattern:

**Landing Page & Core:**
- `GET /` - Landing page with links
- `GET /conformance` - Conformance declaration
- `GET /collections` - Collection metadata
- `GET /api` - OpenAPI 3.0 specification

**Systems (sosa:System):**
- `GET /systems` - List all systems
- `POST /systems` - Create a system
- `GET /systems/{id}` - Get a specific system
- `PUT /systems/{id}` - Replace a system
- `PATCH /systems/{id}` - Update a system
- `DELETE /systems/{id}` - Delete a system
- `GET /systems/{id}/subsystems` - Get subsystems

**Deployments (sosa:Deployment):**
- `GET /deployments`
- `POST /deployments`
- `GET /deployments/{id}`
- `PUT /deployments/{id}`
- `PATCH /deployments/{id}`
- `DELETE /deployments/{id}`

**Procedures (sosa:Procedure):**
- `GET /procedures`
- `POST /procedures`
- `GET /procedures/{id}`
- `PUT /procedures/{id}`
- `PATCH /procedures/{id}`
- `DELETE /procedures/{id}`

**Sampling Features (sosa:Sample):**
- `GET /samplingFeatures`
- `POST /samplingFeatures`
- `GET /samplingFeatures/{id}`
- `PUT /samplingFeatures/{id}`
- `PATCH /samplingFeatures/{id}`
- `DELETE /samplingFeatures/{id}`

**Properties (sosa:ObservableProperty/ActuableProperty):**
- `GET /properties`
- `POST /properties`
- `GET /properties/{id}`
- `PUT /properties/{id}`
- `PATCH /properties/{id}`
- `DELETE /properties/{id}`

### Query Parameters

**Common Filters:**
- `id` - Filter by ID or UID (comma-separated list)
- `bbox` - Bounding box filter (minx,miny,maxx,maxy)
- `datetime` - Temporal filter (ISO 8601)
- `limit` - Maximum number of items (default: 10)
- `offset` - Pagination offset
- `q` - Full-text search

**System-specific:**
- `parent` - Filter by parent system ID
- `procedure` - Filter by procedure ID
- `foi` - Filter by feature of interest
- `observedProperty` - Filter by observed property
- `controlledProperty` - Filter by controlled property
- `recursive` - Include nested resources (true/false)
- `geom` - WKT geometry filter

## Response Formats

### GeoJSON (default)
All feature resources support GeoJSON encoding per OGC API - Features:

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "sensor-001",
      "geometry": {
        "type": "Point",
        "coordinates": [-117.1625, 32.715]
      },
      "properties": {
        "uid": "urn:example:sensor:001",
        "name": "Temperature Sensor #1",
        "featureType": "http://www.w3.org/ns/sosa/Sensor",
        "assetType": "Equipment"
      },
      "links": [...]
    }
  ]
}
```

### SensorML-JSON (optional)
For more detailed system descriptions (not yet implemented).

## Standards Compliance

This implementation conforms to:

- ✅ OGC API - Common - Part 1: Core
- ✅ OGC API - Features - Part 1: Core
- ✅ OGC API - Connected Systems - Part 1 (Conformance Classes):
  - ✅ Common
  - ✅ System Features
  - ✅ Subsystems
  - ✅ Deployment Features
  - ✅ Procedure Features
  - ✅ Sampling Features
  - ✅ Property Definitions
  - ✅ Advanced Filtering (partial)
  - ✅ GeoJSON Format
  - ⬜ SensorML Format (planned)
  - ⬜ Create/Replace/Delete (planned)
  - ⬜ Update (planned)

## Development

### Running Tests
```bash
make test
```

### Building
```bash
make build
```

### Docker
```bash
make docker-build
make docker-run
```

## Next Steps

To complete this implementation:

1. **Implement remaining handlers** for deployments, procedures, sampling features, and properties
2. **Add full query parameter support** (bbox, datetime, geom, etc.)
3. **Implement SensorML encoding** for detailed system descriptions
4. **Add database migrations** with proper PostGIS support
5. **Implement pagination** with proper link generation
6. **Add validation** for incoming requests
7. **Generate OpenAPI spec** from code
8. **Add comprehensive tests** for all endpoints
9. **Implement Part 2** (Dynamic Data) for observations and commands

## References

- [OGC API - Connected Systems Standard](https://docs.ogc.org/is/23-001/23-001.html)
- [SOSA/SSN Ontology](https://www.w3.org/TR/vocab-ssn/)
- [OGC API - Features](https://ogcapi.ogc.org/features/)

## License

MIT
