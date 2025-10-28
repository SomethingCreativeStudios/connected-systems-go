package api

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
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
		},
	}

	render.JSON(w, r, conformance)
}
