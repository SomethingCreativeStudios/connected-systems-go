package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
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
	collectionsHandler := NewCollectionsHandler(cfg, logger)
	systemHandler := NewSystemHandler(cfg, logger, repos.System)
	deploymentHandler := NewDeploymentHandler(cfg, logger, repos.Deployment)
	procedureHandler := NewProcedureHandler(cfg, logger, repos.Procedure)
	samplingFeatureHandler := NewSamplingFeatureHandler(cfg, logger, repos.SamplingFeature)
	propertyHandler := NewPropertyHandler(cfg, logger, repos.Property)

	// Routes

	// Landing page
	r.Get("/", landingHandler.GetLandingPage)

	// Conformance
	r.Get("/conformance", conformanceHandler.GetConformance)

	// Collections
	r.Get("/collections", collectionsHandler.GetCollections)
	r.Get("/collections/{collectionId}", collectionsHandler.GetCollection)
	r.Get("/collections/{collectionId}/items", collectionsHandler.GetCollectionItems)

	// Systems (canonical endpoints)
	r.Route("/systems", func(r chi.Router) {
		r.Get("/", systemHandler.ListSystems)
		r.Post("/", systemHandler.CreateSystem)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", systemHandler.GetSystem)
			r.Put("/", systemHandler.UpdateSystem)
			r.Patch("/", systemHandler.PatchSystem)
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
			r.Patch("/", deploymentHandler.PatchDeployment)
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
			r.Patch("/", procedureHandler.PatchProcedure)
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
			r.Patch("/", samplingFeatureHandler.PatchSamplingFeature)
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
			r.Patch("/", propertyHandler.PatchProperty)
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
