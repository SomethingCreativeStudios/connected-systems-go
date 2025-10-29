package seriallizers

type GeoJsonSeriallizable[T any] interface {
	ToGeoJSON() T
}
