package api

import (
	"net/http"

	"github.com/SomethingCreativeStudios/connected-systems-go/docs"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	serializers "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters/geojson_formatters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters/sensorml_formatters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/queryables"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// NewRouter creates and configures the API router
func NewRouter(cfg *config.Config, logger *zap.Logger, repos *repository.Repositories, pubSubPublisher *pubsub.Publisher) http.Handler {
	r := chi.NewRouter()
	links := newLinkResolver(cfg, repos)
	creates := NewCreateOperations(cfg, repos, pubSubPublisher, logger)

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(repository.WithLinkBaseURL(r.Context(), cfg.API.BaseURL)))
		})
	})
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	maxBodyBytes := defaultMaxBodyBytes
	if cfg != nil && cfg.API.MaxBodyBytes > 0 {
		maxBodyBytes = cfg.API.MaxBodyBytes
	}
	r.Use(limitRequestBody(maxBodyBytes))
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(pubsub.HTTPResourceEventMiddleware(
		pubSubPublisher,
		pubsub.NewRepositoryChangeResolver(cfg.API.BaseURL, repos, representer(links)),
		logger,
	))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link", "Location"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Create handlers
	landingHandler := NewLandingHandler(cfg, logger)
	conformanceHandler := NewConformanceHandler(cfg, logger)
	asyncAPIHandler := NewAsyncAPIHandler(cfg)

	// Formatter collections: the first formatter is the default representation.
	systemFormatterCollection := systemFormatters(links)
	deploymentFormatterCollection := serializers.NewFormatterCollection(
		serializers.Adapt(geojson_formatters.NewDeploymentGeoJSONFormatter(links)),
		serializers.Adapt(sensorml_formatters.NewDeploymentSensorMLFormatter(links)),
	)
	procedureFormatterCollection := serializers.NewFormatterCollection(
		serializers.Adapt(geojson_formatters.NewProcedureGeoJSONFormatter(links)),
		serializers.Adapt(sensorml_formatters.NewProcedureSensorMLFormatter(links)),
	)
	samplingFeatureFormatterCollection := serializers.NewFormatterCollection(
		serializers.Adapt(geojson_formatters.NewSamplingFeatureGeoJSONFormatter(links)),
		serializers.Adapt(sensorml_formatters.NewSamplingFeatureSensorMLFormatter(links)),
	)
	// SensorML is the default representation for properties.
	propertyFormatterCollection := serializers.NewFormatterCollection(
		serializers.Adapt(sensorml_formatters.NewPropertySensorMLFormatter(links)),
		serializers.Adapt(geojson_formatters.NewPropertyGeoJSONFormatter(links)),
	)
	featureFormatterCollection := serializers.NewFormatterCollection(serializers.Adapt(geojson_formatters.NewFeatureGeoJSONFormatter(links)))
	collectionFormatterCollection := serializers.NewFormatterCollection(serializers.Adapt(geojson_formatters.NewFeatureCollectionGeoJSONFormatter(links)))
	datastreamFormatterCollection := datastreamFormatters(links)
	controlStreamFormatterCollection := controlStreamFormatters(links)
	commandFormatterCollection := commandFormatters(links)
	observationFormatterCollection := observationFormatters(links)
	systemFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.System))
	deploymentFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Deployment))
	procedureFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Procedure))
	samplingFeatureFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.SamplingFeature))
	propertyFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Property))
	datastreamFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Datastream))
	controlStreamFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.ControlStream))
	commandFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Command))
	observationFormatterCollection.SetRequestValidator(requestValidator.For(contractvalidation.Observation))

	collectionHandler := NewCollectionHandler(cfg, logger, repos.Collection, collectionFormatterCollection)
	deploymentHandler := NewDeploymentHandler(cfg, logger, repos.Deployment, deploymentFormatterCollection)
	systemHandler := NewSystemHandler(cfg, logger, repos.System, repos.SystemHistory, systemFormatterCollection, repos.Deployment, deploymentFormatterCollection, repos.Procedure, procedureFormatterCollection, creates)
	procedureHandler := NewProcedureHandler(cfg, logger, repos.Procedure, procedureFormatterCollection)
	samplingFeatureHandler := NewSamplingFeatureHandler(cfg, logger, repos.SamplingFeature, repos.System, samplingFeatureFormatterCollection)
	propertyHandler := NewPropertyHandler(cfg, logger, repos.Property, propertyFormatterCollection)
	featureHandler := NewFeatureHandler(cfg, logger, repos.Feature, repos.Collection, featureFormatterCollection)
	datastreamHandler := NewDatastreamHandler(cfg, logger, repos.Datastream, datastreamFormatterCollection, requestValidator, creates)
	observationHandler := NewObservationHandler(cfg, logger, repos.Observation, repos.Datastream, observationFormatterCollection, creates)
	controlStreamHandler := NewControlStreamHandler(cfg, logger, repos.ControlStream, controlStreamFormatterCollection, requestValidator, creates)
	commandHandler := NewCommandHandler(cfg, logger, repos.Command, repos.ControlStream, commandFormatterCollection, requestValidator, creates)
	systemEventHandler := NewSystemEventHandler(cfg, logger, repos.SystemEvent, repos.System, requestValidator, creates)
	// list marks a list route with the queryables resource its CQL2 filter
	// is compiled against.
	list := queryablesMiddleware(cfg.API.BaseURL, repos.Collection)

	// Routes

	// Landing page
	r.Get("/", landingHandler.GetLandingPage)

	// Conformance
	r.Get("/conformance", conformanceHandler.GetConformance)

	// Publish/subscribe service description
	r.Get("/asyncapi", asyncAPIHandler.GetAsyncAPI)

	// Collections
	r.Post("/collections", collectionHandler.CreateCollection)
	r.Get("/collections", collectionHandler.ListCollections)
	r.Get("/collections/{collectionId}", collectionHandler.GetCollection)
	r.Get("/collections/{collectionId}/queryables", collectionHandler.GetQueryables)

	// OGC API Features endpoints (within collections)
	r.Route("/collections/{collectionId}/items", func(r chi.Router) {
		collectionItemsHandler := NewCollectionItemsHandler(featureHandler, map[string]http.HandlerFunc{
			"systems":          systemHandler.ListSystems,
			"deployments":      deploymentHandler.ListDeployments,
			"procedures":       procedureHandler.ListProcedures,
			"samplingFeatures": samplingFeatureHandler.ListSamplingFeatures,
			"properties":       propertyHandler.ListProperties,
			"datastreams":      datastreamHandler.ListDatastreams,
			"observations":     observationHandler.ListObservations,
			"controlstreams":   controlStreamHandler.ListControlStreams,
			"commands":         commandHandler.ListCommands,
			"systemEvents":     systemEventHandler.ListSystemEvents,
		})
		r.With(list(queryables.Features)).Get("/", collectionItemsHandler.List)
		r.With(list(queryables.Features)).Head("/", collectionItemsHandler.Head)
		r.Post("/", featureHandler.CreateFeature)

		r.Route("/{featureId}", func(r chi.Router) {
			r.Get("/", featureHandler.GetFeature)
			r.Put("/", featureHandler.UpdateFeature)
			r.Delete("/", featureHandler.DeleteFeature)
		})
	})

	// Systems (canonical endpoints)
	r.Route("/systems", func(r chi.Router) {
		r.With(list(queryables.Systems)).Get("/", systemHandler.ListSystems)
		r.Post("/", systemHandler.CreateSystem)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", systemHandler.GetSystem)
			r.Put("/", systemHandler.UpdateSystem)
			r.Delete("/", systemHandler.DeleteSystem)

			// Nested Systems endpoints
			r.With(list(queryables.Systems)).Get("/subsystems", systemHandler.GetSubsystems)
			r.Post("/subsystems", systemHandler.AddSubsystem)

			// Associated resource endpoint
			r.With(list(queryables.Deployments)).Get("/deployments", systemHandler.GetDeployments)
			r.With(list(queryables.Procedures)).Get("/procedures", systemHandler.GetProcedures)
			r.With(list(queryables.SamplingFeatures)).Get("/samplingFeatures", samplingFeatureHandler.GetSystemSamplingFeatures)
			r.With(list(queryables.Datastreams)).Get("/datastreams", datastreamHandler.ListSystemDatastreams)
			r.With(list(queryables.ControlStreams)).Get("/controlstreams", controlStreamHandler.ListSystemControlStreams)
			r.With(list(queryables.SystemEvents)).Get("/events", systemEventHandler.ListEventsBySystem)
			r.Post("/events", systemEventHandler.CreateEventBySystem)
			r.With(list(queryables.SystemHistory)).Get("/history", systemHandler.ListSystemHistory)

			// Sampling Features endpoint
			r.Post("/samplingFeatures", samplingFeatureHandler.CreateSamplingFeature)
			r.Post("/datastreams", datastreamHandler.CreateDatastream)
			r.Post("/controlstreams", controlStreamHandler.CreateControlStream)

			r.Route("/events/{eventId}", func(r chi.Router) {
				r.Get("/", systemEventHandler.GetEventByID)
				r.Put("/", systemEventHandler.UpdateEventByID)
				r.Delete("/", systemEventHandler.DeleteEventByID)
			})

			r.Route("/history/{revId}", func(r chi.Router) {
				r.Get("/", systemHandler.GetSystemHistoryRevision)
				r.Put("/", systemHandler.UpdateSystemHistoryRevision)
				r.Delete("/", systemHandler.DeleteSystemHistoryRevision)
			})
		})
	})

	// System Events
	r.Route("/systemEvents", func(r chi.Router) {
		r.With(list(queryables.SystemEvents)).Get("/", systemEventHandler.ListSystemEvents)
	})

	// Datastreams (Part 2: Dynamic Data)
	r.Route("/datastreams", func(r chi.Router) {
		r.With(list(queryables.Datastreams)).Get("/", datastreamHandler.ListDatastreams)

		r.Route("/{dataStreamId}", func(r chi.Router) {
			r.Get("/", datastreamHandler.GetDatastream)
			r.Put("/", datastreamHandler.UpdateDatastream)
			r.Delete("/", datastreamHandler.DeleteDatastream)

			r.Get("/schema", datastreamHandler.GetDatastreamSchema)
			r.Put("/schema", datastreamHandler.UpdateDatastreamSchema)

			r.With(list(queryables.Observations)).Get("/observations", observationHandler.ListDatastreamObservations)
			r.Post("/observations", observationHandler.CreateDatastreamObservation)
		})
	})

	// Control Streams (Part 2: Dynamic Data - Tasking)
	r.Route("/controlstreams", func(r chi.Router) {
		r.With(list(queryables.ControlStreams)).Get("/", controlStreamHandler.ListControlStreams)

		r.Route("/{controlStreamId}", func(r chi.Router) {
			r.Get("/", controlStreamHandler.GetControlStream)
			r.Put("/", controlStreamHandler.UpdateControlStream)
			r.Delete("/", controlStreamHandler.DeleteControlStream)

			r.Get("/schema", controlStreamHandler.GetControlStreamSchema)
			r.Put("/schema", controlStreamHandler.UpdateControlStreamSchema)

			r.With(list(queryables.Commands)).Get("/commands", commandHandler.ListControlStreamCommands)
			r.Post("/commands", commandHandler.CreateControlStreamCommand)
		})
	})

	// Commands (Part 2: Dynamic Data - Tasking)
	r.Route("/commands", func(r chi.Router) {
		r.With(list(queryables.Commands)).Get("/", commandHandler.ListCommands)

		r.Route("/{cmdId}", func(r chi.Router) {
			r.Get("/", commandHandler.GetCommand)
			r.Put("/", commandHandler.UpdateCommand)
			r.Delete("/", commandHandler.DeleteCommand)

			r.Route("/status", func(r chi.Router) {
				r.With(list(queryables.CommandStatuses)).Get("/", commandHandler.ListCommandStatusReports)
				r.Post("/", commandHandler.CreateCommandStatusReport)

				r.Route("/{statusId}", func(r chi.Router) {
					r.Get("/", commandHandler.GetCommandStatusReport)
					r.Put("/", commandHandler.UpdateCommandStatusReport)
					r.Delete("/", commandHandler.DeleteCommandStatusReport)
				})
			})

			r.Route("/result", func(r chi.Router) {
				r.With(list(queryables.CommandResults)).Get("/", commandHandler.ListCommandResults)
				r.Post("/", commandHandler.CreateCommandResult)

				r.Route("/{resultId}", func(r chi.Router) {
					r.Get("/", commandHandler.GetCommandResult)
					r.Put("/", commandHandler.UpdateCommandResult)
					r.Delete("/", commandHandler.DeleteCommandResult)
				})
			})
		})
	})

	// Observations (Part 2: Dynamic Data)
	r.Route("/observations", func(r chi.Router) {
		r.With(list(queryables.Observations)).Get("/", observationHandler.ListObservations)

		r.Route("/{obsId}", func(r chi.Router) {
			r.Get("/", observationHandler.GetObservation)
			r.Put("/", observationHandler.UpdateObservation)
			r.Delete("/", observationHandler.DeleteObservation)
		})
	})

	// Deployments (canonical endpoints)
	r.Route("/deployments", func(r chi.Router) {
		r.With(list(queryables.Deployments)).Get("/", deploymentHandler.ListDeployments)
		r.Post("/", deploymentHandler.CreateDeployment)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", deploymentHandler.GetDeployment)
			r.With(list(queryables.SamplingFeatures)).Get("/samplingFeatures", deploymentSamplingFeatures(deploymentHandler, samplingFeatureHandler))
			r.With(list(queryables.Features)).Get("/featuresOfInterest", deploymentFeaturesOfInterest(deploymentHandler, featureHandler))
			r.With(list(queryables.Datastreams)).Get("/datastreams", deploymentDatastreams(deploymentHandler, datastreamHandler))
			r.With(list(queryables.ControlStreams)).Get("/controlstreams", deploymentControlstreams(deploymentHandler, controlStreamHandler))
			r.Put("/", deploymentHandler.UpdateDeployment)
			r.Delete("/", deploymentHandler.DeleteDeployment)

			// Subdeployments endpoint
			r.With(list(queryables.Deployments)).Get("/subdeployments", deploymentHandler.ListSubdeployments)
			r.Post("/subdeployments", deploymentHandler.AddSubdeployment)
		})
	})

	// Procedures (canonical endpoints)
	r.Route("/procedures", func(r chi.Router) {
		r.With(list(queryables.Procedures)).Get("/", procedureHandler.ListProcedures)
		r.Post("/", procedureHandler.CreateProcedure)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", procedureHandler.GetProcedure)
			r.Put("/", procedureHandler.UpdateProcedure)
			r.Delete("/", procedureHandler.DeleteProcedure)
		})
	})

	// Sampling Features (canonical endpoints)
	r.Route("/samplingFeatures", func(r chi.Router) {
		r.With(list(queryables.SamplingFeatures)).Get("/", samplingFeatureHandler.ListSamplingFeatures)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", samplingFeatureHandler.GetSamplingFeature)
			r.Put("/", samplingFeatureHandler.UpdateSamplingFeature)
			r.Delete("/", samplingFeatureHandler.DeleteSamplingFeature)
		})
	})

	// Properties (canonical endpoints)
	r.Route("/properties", func(r chi.Router) {
		r.With(list(queryables.Properties)).Get("/", propertyHandler.ListProperties)
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
		//nolint:errcheck
		w.Write(docs.OpenAPI3Spec)
	})

	// Swagger UI
	r.Get("/api/ui/*", httpSwagger.Handler(
		httpSwagger.URL("/api"),
	))

	return r
}

// newLinkResolver gives the formatters the API base URL, which makes link
// hrefs absolute, and the repository lookups that title links and find which
// nested collections have members.
func newLinkResolver(cfg *config.Config, repos *repository.Repositories) *serializers.LinkResolver {
	links := &serializers.LinkResolver{}
	if cfg != nil {
		links.BaseURL = cfg.API.BaseURL
	}
	if repos != nil {
		links.Systems = repos.System
		links.Procedures = repos.Procedure
		links.Deployments = repos.Deployment
		links.SamplingFeatures = repos.SamplingFeature
		links.SystemChildren = repos.System
		links.DeploymentChildren = repos.Deployment
		links.DeploymentSystems = repos.Deployment
		links.ProcedureChildren = repos.Procedure
	}
	return links
}
