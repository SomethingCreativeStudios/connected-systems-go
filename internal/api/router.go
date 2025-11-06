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
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/model/serializers/geojson_serializers"
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

	// Create serializer and inject lightweight repository readers
	systemSerializerCollection := buildSystemSerializerCollection(repos)
	deploymentSerializerCollection := buildDeploymentSerializerCollection(repos)
	procedureSerializerCollection := buildProcedureSerializerCollection(repos)
	samplingFeatureSerializerCollection := buildSamplingFeatureSerializerCollection(repos)
	propertySerializerCollection := buildPropertySerializerCollection(repos)
	featureSerializerCollection := buildFeatureSerializerCollection(repos)
	collectionSerializerCollection := buildCollectionSerializerCollection(repos)

	collectionHandler := NewCollectionHandler(repos.Collection, collectionSerializerCollection)
	systemHandler := NewSystemHandler(cfg, logger, repos.System, systemSerializerCollection)
	deploymentHandler := NewDeploymentHandler(cfg, logger, repos.Deployment, deploymentSerializerCollection)
	procedureHandler := NewProcedureHandler(cfg, logger, repos.Procedure, procedureSerializerCollection)
	samplingFeatureHandler := NewSamplingFeatureHandler(cfg, logger, repos.SamplingFeature, samplingFeatureSerializerCollection)
	propertyHandler := NewPropertyHandler(cfg, logger, repos.Property, propertySerializerCollection)
	featureHandler := NewFeatureHandler(cfg, logger, repos.Feature, featureSerializerCollection)

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

			// Nested endpoints
			r.Get("/subsystems", systemHandler.GetSubsystems)
			r.Post("/subsystems", systemHandler.AddSubsystem)
			r.Get("/samplingFeatures", samplingFeatureHandler.GetSystemSamplingFeatures)
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
		r.Post("/", samplingFeatureHandler.CreateSamplingFeature)

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

// Serializers
// TODO: Maybe move to a different area?

func buildSystemSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.SystemGeoJSONFeature, *domains.System] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.SystemGeoJSONFeature, *domains.System]{
		"application/geo+json": geojson_serializers.NewSystemGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewSystemGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildDeploymentSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.DeploymentGeoJSONFeature, *domains.Deployment] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.DeploymentGeoJSONFeature, *domains.Deployment]{
		"application/geo+json": geojson_serializers.NewDeploymentGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewDeploymentGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildProcedureSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.ProcedureGeoJSONFeature, *domains.Procedure] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.ProcedureGeoJSONFeature, *domains.Procedure]{
		"application/geo+json": geojson_serializers.NewProcedureGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewProcedureGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildSamplingFeatureSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]{
		"application/geo+json": geojson_serializers.NewSamplingFeatureGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewSamplingFeatureGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildPropertySerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.PropertyGeoJSONFeature, *domains.Property] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.PropertyGeoJSONFeature, *domains.Property]{
		"application/geo+json": geojson_serializers.NewPropertyGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewPropertyGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildFeatureSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.FeatureGeoJSONFeature, *domains.Feature] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.FeatureGeoJSONFeature, *domains.Feature]{
		"application/geo+json": geojson_serializers.NewFeatureGeoJSONSerializer(),
		"default":              geojson_serializers.NewFeatureGeoJSONSerializer(),
	}

	return serializers.NewSerializerCollection(serMap)
}

func buildCollectionSerializerCollection(repos *repository.Repositories) *serializers.SerializerCollection[domains.CollectionGeoJSONFeature, *domains.Collection] {
	// create concrete serializers and register them by content type
	serMap := map[string]serializers.Serializer[domains.CollectionGeoJSONFeature, *domains.Collection]{
		"application/geo+json": geojson_serializers.NewCollectionGeoJSONSerializer(repos),
		"default":              geojson_serializers.NewCollectionGeoJSONSerializer(repos),
	}

	return serializers.NewSerializerCollection(serMap)
}
