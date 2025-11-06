# OGC Connected Systems API - Example JSON Documents

This directory contains example JSON documents for creating resources via the OGC Connected Systems API and OGC API - Features.

## OGC API - Features (Part 1: Core)

Generic features following the OGC API - Features Part 1 specification.

### Point Features
- **`create-feature-point.json`** - Building feature with point geometry
  ```bash
  curl -X POST http://localhost:8080/collections/buildings/items \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-feature-point.json
  ```

### Polygon Features
- **`create-feature-polygon.json`** - Land parcel with polygon geometry
  ```bash
  curl -X POST http://localhost:8080/collections/parcels/items \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-feature-polygon.json
  ```

### LineString Features
- **`create-feature-linestring.json`** - Road feature with linestring geometry
  ```bash
  curl -X POST http://localhost:8080/collections/roads/items \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-feature-linestring.json
  ```

### Observation Features
- **`create-feature-observation.json`** - Temporal observation with point geometry
  ```bash
  curl -X POST http://localhost:8080/collections/observations/items \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-feature-observation.json
  ```

## Connected Systems Resources

### Systems

Systems represent sensors, actuators, platforms, or composite systems.

### Sensors
- **`create-system-sensor.json`** - Temperature sensor example
  ```bash
  curl -X POST http://localhost:8080/systems \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-system-sensor.json
  ```

### Platforms
- **`create-system-platform.json`** - UAV platform example
  ```bash
  curl -X POST http://localhost:8080/systems \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-system-platform.json
  ```

### Actuators
- **`create-system-actuator.json`** - Control valve actuator
  ```bash
  curl -X POST http://localhost:8080/systems \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-system-actuator.json
  ```

### Composite Systems
- **`create-system-composite.json`** - Multi-sensor weather station
  ```bash
  curl -X POST http://localhost:8080/systems \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-system-composite.json
  ```

## Deployments

Deployments describe when and where systems are deployed.

### Fixed Deployments
- **`create-deployment-fixed.json`** - Weather station deployment
  ```bash
  curl -X POST http://localhost:8080/deployments \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-deployment-fixed.json
  ```

### Mobile Deployments
- **`create-deployment-mobile.json`** - UAV survey mission
  ```bash
  curl -X POST http://localhost:8080/deployments \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-deployment-mobile.json
  ```

## Procedures

Procedures describe system datasheets, methodologies, or standard methods.

### Equipment Datasheets
- **`create-procedure-datasheet.json`** - Digital thermometer specs
  ```bash
  curl -X POST http://localhost:8080/procedures \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-procedure-datasheet.json
  ```

### Standard Methods
- **`create-procedure-method.json`** - EPA water sampling method
  ```bash
  curl -X POST http://localhost:8080/procedures \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-procedure-method.json
  ```

## Sampling Features

Sampling features represent the sampling strategy and geometry.

### Sampling Points
- **`create-sampling-feature-point.json`** - Point-based measurement location
  ```bash
  curl -X POST http://localhost:8080/samplingFeatures \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-sampling-feature-point.json
  ```

### Sampling Curves
- **`create-sampling-feature-curve.json`** - River transect line
  ```bash
  curl -X POST http://localhost:8080/samplingFeatures \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-sampling-feature-curve.json
  ```

### Sampling Surfaces
- **`create-sampling-feature-surface.json`** - Satellite image footprint
  ```bash
  curl -X POST http://localhost:8080/samplingFeatures \
    -H "Content-Type: application/geo+json" \
    -d @examples/create-sampling-feature-surface.json
  ```

## Properties

Properties define observable or actuable characteristics.

### Observable Properties
- **`create-property-observable-temperature.json`** - Air temperature
  ```bash
  curl -X POST http://localhost:8080/properties \
    -H "Content-Type: application/json" \
    -d @examples/create-property-observable-temperature.json
  ```

- **`create-property-observable-humidity.json`** - Relative humidity
  ```bash
  curl -X POST http://localhost:8080/properties \
    -H "Content-Type: application/json" \
    -d @examples/create-property-observable-humidity.json
  ```

- **`create-property-observable-ndvi.json`** - Vegetation index
  ```bash
  curl -X POST http://localhost:8080/properties \
    -H "Content-Type: application/json" \
    -d @examples/create-property-observable-ndvi.json
  ```

### Actuable Properties
- **`create-property-actuable-valve.json`** - Valve position control
  ```bash
  curl -X POST http://localhost:8080/properties \
    -H "Content-Type: application/json" \
    -d @examples/create-property-actuable-valve.json
  ```

## Testing Workflow

1. **Start the server:**
   ```bash
   go run cmd/server/main.go
   ```

2. **Create properties first** (referenced by other resources):
   ```bash
   curl -X POST http://localhost:8080/properties \
     -H "Content-Type: application/json" \
     -d @examples/create-property-observable-temperature.json
   ```

3. **Create procedures** (system datasheets):
   ```bash
   curl -X POST http://localhost:8080/procedures \
     -H "Content-Type: application/geo+json" \
     -d @examples/create-procedure-datasheet.json
   ```

4. **Create systems:**
   ```bash
   curl -X POST http://localhost:8080/systems \
     -H "Content-Type: application/geo+json" \
     -d @examples/create-system-sensor.json
   ```

5. **Create deployments:**
   ```bash
   curl -X POST http://localhost:8080/deployments \
     -H "Content-Type: application/geo+json" \
     -d @examples/create-deployment-fixed.json
   ```

6. **Create sampling features:**
   ```bash
   curl -X POST http://localhost:8080/samplingFeatures \
     -H "Content-Type: application/geo+json" \
     -d @examples/create-sampling-feature-point.json
   ```

7. **Query resources:**
   ```bash
   # List all systems
   curl http://localhost:8080/systems
   
   # Get specific system
   curl http://localhost:8080/systems/{id}
   
   # Filter by type
   curl "http://localhost:8080/systems?q=temperature"
   
   # Get subsystems
   curl http://localhost:8080/systems/{id}/subsystems?recursive=true
   ```

## Resource Types & Feature Types

| Resource | SOSA/SSN Type | Example |
|----------|---------------|---------|
| System (Sensor) | `sosa:Sensor` | Temperature sensor, GPS receiver |
| System (Actuator) | `sosa:Actuator` | Valve, motor controller |
| System (Platform) | `sosa:Platform` | UAV, weather station tower |
| System (Sampler) | `sosa:Sampler` | Water sampler, air sampler |
| Deployment | `ssn:Deployment` | Field campaign, permanent installation |
| Procedure | `sosa:Procedure` | Datasheet, calibration method |
| Sampling Feature | `sosa:Sample` | Measurement point, transect, footprint |
| Property | `sosa:ObservableProperty` / `sosa:ActuableProperty` | Temperature, humidity, valve position |

## Notes

- All System, Deployment, Procedure, and Sampling Feature resources use GeoJSON Feature format
- Property resources use plain JSON (they are not features)
- The `uid` field should contain a globally unique URI (preferably a URN)
- The `featureType` field identifies the SOSA/SSN concept
- Geometry can be Point, LineString, Polygon, or null (for procedures)
- Links follow RFC 8288 web linking conventions
