package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	geom "github.com/twpayne/go-geom"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostGISContainer wraps a testcontainer instance with helper methods
type PostGISContainer struct {
	container tc.Container
	DSN       string
}

// StartPostGISContainer starts a PostGIS container for tests and returns a container wrapper
func StartPostGISContainer(ctx context.Context, t *testing.T) *PostGISContainer {
	t.Helper()

	req := tc.ContainerRequest{
		Image:        "imresamu/postgis:18-3.6.0-alpine3.22",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=test password=secret dbname=testdb sslmode=disable", host, port.Port())

	return &PostGISContainer{
		container: container,
		DSN:       dsn,
	}
}

// Terminate stops and removes the container
func (c *PostGISContainer) Terminate(ctx context.Context) error {
	if c.container != nil {
		return c.container.Terminate(ctx)
	}
	return nil
}

// OpenTestDBOptions configures the test database setup
type OpenTestDBOptions struct {
	// EnableLogging enables SQL query logging (useful for debugging)
	EnableLogging bool
	// Models to auto-migrate (in order)
	Models []interface{}
}

// OpenTestDB opens a GORM database connection with PostGIS extension and auto-migration
func OpenTestDB(t *testing.T, dsn string, opts OpenTestDBOptions) *gorm.DB {
	t.Helper()

	config := &gorm.Config{}
	if opts.EnableLogging {
		config.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	require.NoError(t, err)

	// Ensure PostGIS is available
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Wait a short moment for postgres readiness
	time.Sleep(250 * time.Millisecond)

	// Create PostGIS extension if missing
	err = db.Exec("CREATE EXTENSION IF NOT EXISTS postgis;").Error
	require.NoError(t, err)

	// Auto-migrate models in the order provided
	if len(opts.Models) > 0 {
		for _, model := range opts.Models {
			err = db.AutoMigrate(model)
			require.NoError(t, err, "failed to migrate model: %T", model)
		}
	}

	// Ping to ensure connectivity
	require.NoError(t, sqlDB.Ping())

	return db
}

// DefaultSystemModels returns the standard migration order for System-related tests
func DefaultSystemModels() []interface{} {
	return []interface{}{
		&domains.System{},
		&domains.Deployment{},
		&domains.Procedure{},
		&domains.SamplingFeature{},
		&domains.Property{},
		&domains.Feature{},
		&domains.Collection{},
	}
}

// AllModels returns all domain models in proper migration order
func AllModels() []interface{} {
	return []interface{}{
		&domains.System{},
		&domains.Deployment{},
		&domains.Procedure{},
		&domains.SamplingFeature{},
		&domains.Property{},
		&domains.Feature{},
		&domains.Collection{},
	}
}

func PtrTime(t time.Time) *time.Time {
	return &t
}

func PtrStr(s string) *string {
	return &s
}

func PtrBool(b bool) *bool {
	return &b
}

// Geometry maker functions for tests

// MakePoint creates a PostGIS Point geometry (lon, lat) with SRID 4326 (WGS84)
func MakePoint(lon, lat float64) *common_shared.GoGeom {
	point := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat})
	point.SetSRID(4326) // WGS84
	return &common_shared.GoGeom{T: point}
}

// MakeLineString creates a PostGIS LineString from a series of (lon, lat) coordinate pairs
// Example: MakeLineString([]float64{-122.0, 37.0, -122.1, 37.1})
func MakeLineString(coords []float64) *common_shared.GoGeom {
	if len(coords)%2 != 0 {
		panic("coords must have even length (lon, lat pairs)")
	}
	line := geom.NewLineString(geom.XY).MustSetCoords(coordPairsToCoords(coords))
	line.SetSRID(4326)
	return &common_shared.GoGeom{T: line}
}

// MakePolygon creates a PostGIS Polygon from a series of (lon, lat) coordinate pairs
// The first and last coordinates should be the same to close the ring
// Example: MakePolygon([]float64{-122.0, 37.0, -122.1, 37.0, -122.1, 37.1, -122.0, 37.1, -122.0, 37.0})
func MakePolygon(coords []float64) *common_shared.GoGeom {
	if len(coords)%2 != 0 {
		panic("coords must have even length (lon, lat pairs)")
	}
	ring := coordPairsToCoords(coords)
	poly := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{ring})
	poly.SetSRID(4326)
	return &common_shared.GoGeom{T: poly}
}

// MakeMultiPoint creates a PostGIS MultiPoint from a series of (lon, lat) coordinate pairs
func MakeMultiPoint(coords []float64) *common_shared.GoGeom {
	if len(coords)%2 != 0 {
		panic("coords must have even length (lon, lat pairs)")
	}
	points := coordPairsToCoords(coords)
	mp := geom.NewMultiPoint(geom.XY).MustSetCoords(points)
	mp.SetSRID(4326)
	return &common_shared.GoGeom{T: mp}
}

// MakeMultiLineString creates a PostGIS MultiLineString from multiple line coordinate arrays
// Example: MakeMultiLineString([][]float64{{-122.0, 37.0, -122.1, 37.1}, {-122.2, 37.2, -122.3, 37.3}})
func MakeMultiLineString(lines [][]float64) *common_shared.GoGeom {
	coords := make([][]geom.Coord, len(lines))
	for i, line := range lines {
		coords[i] = coordPairsToCoords(line)
	}
	mls := geom.NewMultiLineString(geom.XY).MustSetCoords(coords)
	mls.SetSRID(4326)
	return &common_shared.GoGeom{T: mls}
}

// MakeMultiPolygon creates a PostGIS MultiPolygon from multiple polygon ring arrays
// Each polygon is an array of rings (first ring is exterior, subsequent rings are holes)
//
//	Example: MakeMultiPolygon([][][]float64{
//	  {{-122.0, 37.0, -122.1, 37.0, -122.1, 37.1, -122.0, 37.1, -122.0, 37.0}},
//	  {{-122.2, 37.2, -122.3, 37.2, -122.3, 37.3, -122.2, 37.3, -122.2, 37.2}},
//	})
func MakeMultiPolygon(polygons [][][]float64) *common_shared.GoGeom {
	polyCoords := make([][][]geom.Coord, len(polygons))
	for i, poly := range polygons {
		polyCoords[i] = make([][]geom.Coord, len(poly))
		for j, ring := range poly {
			polyCoords[i][j] = coordPairsToCoords(ring)
		}
	}
	mpoly := geom.NewMultiPolygon(geom.XY).MustSetCoords(polyCoords)
	mpoly.SetSRID(4326)
	return &common_shared.GoGeom{T: mpoly}
}

// MakeGeometryCollection creates a PostGIS GeometryCollection from multiple geometries
func MakeGeometryCollection(geoms ...*common_shared.GoGeom) *common_shared.GoGeom {
	gts := make([]geom.T, 0, len(geoms))
	for _, g := range geoms {
		if g != nil && g.T != nil {
			gts = append(gts, g.T)
		}
	}
	gc := geom.NewGeometryCollection().SetSRID(4326)
	// GeometryCollection uses Push to add geometries
	for _, gt := range gts {
		gc.Push(gt)
	}
	return &common_shared.GoGeom{T: gc}
}

// Helper function to convert flat coordinate pairs to geom.Coord slice
func coordPairsToCoords(coords []float64) []geom.Coord {
	result := make([]geom.Coord, len(coords)/2)
	for i := 0; i < len(coords); i += 2 {
		result[i/2] = geom.Coord{coords[i], coords[i+1]}
	}
	return result
}

// TestBoundingBoxLA returns a BoundingBox that contains Los Angeles, CA
func TestBoundingBoxLA() *common_shared.BoundingBox {
	// Los Angeles city center: lat 34.0522, lon -118.2437
	// We'll use a box roughly 0.1 deg around the center
	box := common_shared.BoundingBox{
		MinX: -118.30, // west
		MinY: 34.00,   // south
		MaxX: -118.18, // east
		MaxY: 34.10,   // north
	}

	return &box
}

// TestBoundingBoxLA_SF returns a BoundingBox that intersects both Los Angeles and San Francisco
func TestBoundingBoxLA_SF() *common_shared.BoundingBox {
	// LA: lat 34.0522, lon -118.2437
	// SF: lat 37.7749, lon -122.4194
	// We'll use a box that covers from just south of LA to just north of SF, and from west of SF to east of LA
	box := common_shared.BoundingBox{
		MinX: -123.0, // west of SF
		MinY: 33.5,   // south of LA
		MaxX: -117.5, // east of LA
		MaxY: 38.0,   // north of SF
	}
	return &box
}

// TestBoundingBoxSeattle returns a BoundingBox that contains Seattle, WA
func TestBoundingBoxSeattle() *common_shared.BoundingBox {
	// Seattle city center: lat 47.6062, lon -122.3321
	// We'll use a box roughly 0.1 deg around the center
	test := common_shared.BoundingBox{
		MinX: -122.38, // west
		MinY: 47.55,   // south
		MaxX: -122.28, // east
		MaxY: 47.65,   // north
	}

	return &test
}
