package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	serializers "github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/model/formaters/geojson_formatters"
	"github.com/yourusername/connected-systems-go/internal/model/formaters/sensorml_formatters"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// NewRouter creates and configures the API router
func NewRouter(cfg *config.Config, logger *zap.Logger, repos *repository.Repositories) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Create handlers
	landingHandler := NewLandingHandler(cfg, logger)
	conformanceHandler := NewConformanceHandler(cfg, logger)

	// Create formatter collections and inject lightweight repository readers
	systemFormatterCollection := buildSystemFormatterCollection(repos)
	deploymentFormatterCollection := buildDeploymentFormatterCollection(repos)
	procedureFormatterCollection := buildProcedureFormatterCollection(repos)
	samplingFeatureFormatterCollection := buildSamplingFeatureFormatterCollection(repos)
	propertyFormatterCollection := buildPropertyFormatterCollection(repos)
	featureFormatterCollection := buildFeatureFormatterCollection(repos)
	collectionFormatterCollection := buildCollectionFormatterCollection(repos)

	collectionHandler := NewCollectionHandler(cfg, logger, repos.Collection, collectionFormatterCollection)
	deploymentHandler := NewDeploymentHandler(cfg, logger, repos.Deployment, deploymentFormatterCollection)
	systemHandler := NewSystemHandler(cfg, logger, repos.System, systemFormatterCollection, repos.Deployment, deploymentFormatterCollection)
	procedureHandler := NewProcedureHandler(cfg, logger, repos.Procedure, procedureFormatterCollection)
	samplingFeatureHandler := NewSamplingFeatureHandler(cfg, logger, repos.SamplingFeature, samplingFeatureFormatterCollection)
	propertyHandler := NewPropertyHandler(cfg, logger, repos.Property, propertyFormatterCollection)
	featureHandler := NewFeatureHandler(cfg, logger, repos.Feature, featureFormatterCollection)

	// Routes

	// Landing page
	r.Get("/", landingHandler.GetLandingPage)

	// Conformance
	r.Get("/conformance", conformanceHandler.GetConformance)

	// Collections
	r.Post("/collections", collectionHandler.CreateCollection)
	r.Get("/collections", collectionHandler.ListCollections)
	r.Get("/collections/{collectionId}", collectionHandler.GetCollection)

	// OGC API Features endpoints (within collections)
	r.Route("/collections/{collectionId}/items", func(r chi.Router) {
		r.Get("/", featureHandler.ListFeatures)
		r.Post("/", featureHandler.CreateFeature)

		r.Route("/{featureId}", func(r chi.Router) {
			r.Get("/", featureHandler.GetFeature)
			r.Put("/", featureHandler.UpdateFeature)
			r.Delete("/", featureHandler.DeleteFeature)
		})
	})

	// Systems (canonical endpoints)
	r.Route("/systems", func(r chi.Router) {
		r.Get("/", systemHandler.ListSystems)
		r.Post("/", systemHandler.CreateSystem)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", systemHandler.GetSystem)
			r.Put("/", systemHandler.UpdateSystem)
			r.Delete("/", systemHandler.DeleteSystem)

			// Nested Systems endpoints
			r.Get("/subsystems", systemHandler.GetSubsystems)
			r.Post("/subsystems", systemHandler.AddSubsystem)

			// Associated resource endpoint
			r.Get("/deployments", systemHandler.GetDeployments)
			r.Get("/samplingFeatures", samplingFeatureHandler.GetSystemSamplingFeatures)

			// Sampling Features endpoint
			r.Post("/samplingFeatures", samplingFeatureHandler.CreateSamplingFeature)
		})
	})

	// Deployments (canonical endpoints)
	r.Route("/deployments", func(r chi.Router) {
		r.Get("/", deploymentHandler.ListDeployments)
		r.Post("/", deploymentHandler.CreateDeployment)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", deploymentHandler.GetDeployment)
			r.Put("/", deploymentHandler.UpdateDeployment)
			r.Delete("/", deploymentHandler.DeleteDeployment)

			// Subdeployments endpoint
			r.Get("/subdeployments", deploymentHandler.ListSubdeployments)
			r.Post("/subdeployments", deploymentHandler.AddSubdeployment)
		})
	})

	// Procedures (canonical endpoints)
	r.Route("/procedures", func(r chi.Router) {
		r.Get("/", procedureHandler.ListProcedures)
		r.Post("/", procedureHandler.CreateProcedure)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", procedureHandler.GetProcedure)
			r.Put("/", procedureHandler.UpdateProcedure)
			r.Delete("/", procedureHandler.DeleteProcedure)
		})
	})

	// Sampling Features (canonical endpoints)
	r.Route("/samplingFeatures", func(r chi.Router) {
		r.Get("/", samplingFeatureHandler.ListSamplingFeatures)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", samplingFeatureHandler.GetSamplingFeature)
			r.Put("/", samplingFeatureHandler.UpdateSamplingFeature)
			r.Delete("/", samplingFeatureHandler.DeleteSamplingFeature)
		})
	})

	// Properties (canonical endpoints)
	r.Route("/properties", func(r chi.Router) {
		r.Get("/", propertyHandler.ListProperties)
		r.Post("/", propertyHandler.CreateProperty)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", propertyHandler.GetProperty)
			r.Put("/", propertyHandler.UpdateProperty)
			r.Delete("/", propertyHandler.DeleteProperty)
		})
	})

	// OpenAPI spec
	r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.oai.openapi+json;version=3.0")
		fmt.Fprint(w, getOpenAPISpec(cfg))
	})

	return r
}

func getOpenAPISpec(cfg *config.Config) string {
	// TODO: Implement OpenAPI 3.0 spec generation
	return `{"openapi": "3.0.0", "info": {"title": "` + cfg.API.Title + `", "version": "` + cfg.API.Version + `"}}`
}

// Formatters
// These provide both serialization and deserialization capabilities

func buildSystemFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.System] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.System]("application/geo+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewSystemGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Register SensorML formatter
	sensorMLFormatter := sensorml_formatters.NewSystemSensorMLFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/sml+json", sensorMLFormatter)

	// Set default (GeoJSON is the default for systems)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}

func buildDeploymentFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.Deployment] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.Deployment]("application/geo+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewDeploymentGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Register SensorML formatter
	sensorMLFormatter := sensorml_formatters.NewDeploymentSensorMLFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/sml+json", sensorMLFormatter)

	// Set default (GeoJSON is the default for deployments)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}

func buildProcedureFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.Procedure] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.Procedure]("application/geo+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewProcedureGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Register SensorML formatter
	sensorMLFormatter := sensorml_formatters.NewProcedureSensorMLFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/sml+json", sensorMLFormatter)

	// Set default (GeoJSON is the default for procedures)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}

func buildSamplingFeatureFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.SamplingFeature] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.SamplingFeature]("application/geo+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewSamplingFeatureGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Register SensorML formatter
	sensorMLFormatter := sensorml_formatters.NewSamplingFeatureSensorMLFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/sml+json", sensorMLFormatter)

	// Set default (GeoJSON is the default for sampling features)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}

func buildPropertyFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.Property] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.Property]("application/sml+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewPropertyGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Register SensorML formatter
	sensorMLFormatter := sensorml_formatters.NewPropertySensorMLFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/sml+json", sensorMLFormatter)

	// Set default (SensorML is the default for properties per OGC Connected Systems)
	serializers.RegisterFormatterTypedDefault(collection, sensorMLFormatter, "application/sml+json")

	return collection
}

func buildFeatureFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.Feature] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.Feature]("application/sml+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewFeatureGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Set default (SensorML is the default for properties per OGC Connected Systems)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}

func buildCollectionFormatterCollection(repos *repository.Repositories) *serializers.MultiFormatFormatterCollection[*domains.Collection] {
	collection := serializers.NewMultiFormatFormatterCollection[*domains.Collection]("application/sml+json")

	// Register GeoJSON formatter
	geoJSONFormatter := geojson_formatters.NewFeatureCollectionGeoJSONFormatter(repos)
	serializers.RegisterFormatterTyped(collection, "application/geo+json", geoJSONFormatter)

	// Set default (SensorML is the default for properties per OGC Connected Systems)
	serializers.RegisterFormatterTypedDefault(collection, geoJSONFormatter, "application/geo+json")

	return collection
}
