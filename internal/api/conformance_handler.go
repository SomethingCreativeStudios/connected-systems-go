package api

import (
	"net/http"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// ConformanceHandler handles conformance declarations
type ConformanceHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewConformanceHandler creates a new ConformanceHandler
func NewConformanceHandler(cfg *config.Config, logger *zap.Logger) *ConformanceHandler {
	return &ConformanceHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// GetConformance returns the conformance declaration
//
// @Summary     Get conformance declaration
// @Description Returns the list of OGC conformance classes this server implements
// @Tags        Core
// @Produce     json
// @Success     200  {object}  map[string]any
// @Router      /conformance [get]
func (h *ConformanceHandler) GetConformance(w http.ResponseWriter, r *http.Request) {
	conformance := model.ConformanceDeclaration{
		ConformsTo: []string{
			// OGC API - Common
			"http://www.opengis.net/spec/ogcapi-common-1/1.0/conf/core",
			"http://www.opengis.net/spec/ogcapi-common-1/1.0/conf/landing-page",
			"http://www.opengis.net/spec/ogcapi-common-1/1.0/conf/json",
			"http://www.opengis.net/spec/ogcapi-common-2/1.0/conf/collections",

			// OGC API - Features
			"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/core",
			"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/geojson",

			// OGC API - Features Part 3 (Filtering and Queryables). CQL2
			// custom functions are deliberately not advertised; the configured
			// functions are only CQL2 standard conformance functions.
			"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/queryables",
			"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/queryables-query-parameters",
			"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/filter",
			"http://www.opengis.net/spec/ogcapi-features-3/1.0/conf/features-filter",
			"http://www.opengis.net/spec/cql2/1.0/conf/basic-cql2",
			"http://www.opengis.net/spec/cql2/1.0/conf/advanced-comparison-operators",
			"http://www.opengis.net/spec/cql2/1.0/conf/case-insensitive-comparison",
			"http://www.opengis.net/spec/cql2/1.0/conf/accent-insensitive-comparison",
			"http://www.opengis.net/spec/cql2/1.0/conf/spatial-functions",
			"http://www.opengis.net/spec/cql2/1.0/conf/temporal-functions",
			"http://www.opengis.net/spec/cql2/1.0/conf/array-functions",
			"http://www.opengis.net/spec/cql2/1.0/conf/property-property",
			"http://www.opengis.net/spec/cql2/1.0/conf/arithmetic",
			"http://www.opengis.net/spec/cql2/1.0/conf/cql2-text",
			"http://www.opengis.net/spec/cql2/1.0/conf/cql2-json",

			// OGC API - Connected Systems - Part 1
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/api-common",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/system",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/subsystem",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/deployment",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/procedure",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/sf",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/property",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/advanced-filtering",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-1/1.0/conf/geojson",

			// OGC API - Connected Systems - Part 2 (Dynamic Data)
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/api-common",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/datastream",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/observation",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/controlstream",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/command",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/system-event",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/system-history",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/json",
			"http://www.opengis.net/spec/ogcapi-connectedsystems-2/1.0/conf/create-replace-delete",
		},
	}

	render.JSON(w, r, conformance)
}
