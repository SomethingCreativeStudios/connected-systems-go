package api

import (
	"net/http"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// LandingHandler handles the API landing page
type LandingHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewLandingHandler creates a new LandingHandler
func NewLandingHandler(cfg *config.Config, logger *zap.Logger) *LandingHandler {
	return &LandingHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// GetLandingPage returns the API landing page
//
// @Summary     Get landing page
// @Description Returns the API landing page with links to available resources
// @Tags        Core
// @Produce     json
// @Success     200  {object}  map[string]any
// @Router      / [get]
func (h *LandingHandler) GetLandingPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.API.BaseURL

	landingPage := model.LandingPage{
		Title:       h.cfg.API.Title,
		Description: h.cfg.API.Description,
		Links: common_shared.Links{
			{
				Href:  baseURL + "/",
				Rel:   "self",
				Type:  mediatype.JSON,
				Title: "This document",
			},
			{
				Href:  baseURL + "/api",
				Rel:   "service-desc",
				Type:  "application/vnd.oai.openapi+json;version=3.0",
				Title: "API definition",
			},
			{
				Href:  baseURL + "/conformance",
				Rel:   "conformance",
				Type:  mediatype.JSON,
				Title: "Conformance declaration",
			},
			{
				Href:  baseURL + "/collections",
				Rel:   "data",
				Type:  mediatype.JSON,
				Title: "Collections",
			},
			{
				Href:  baseURL + "/systems",
				Rel:   "data",
				Type:  mediatype.GeoJSON,
				Title: "Systems",
			},
			{
				Href:  baseURL + "/deployments",
				Rel:   "data",
				Type:  mediatype.GeoJSON,
				Title: "Deployments",
			},
			{
				Href:  baseURL + "/procedures",
				Rel:   "data",
				Type:  mediatype.GeoJSON,
				Title: "Procedures",
			},
			{
				Href:  baseURL + "/samplingFeatures",
				Rel:   "data",
				Type:  mediatype.GeoJSON,
				Title: "Sampling Features",
			},
			{
				Href:  baseURL + "/properties",
				Rel:   "data",
				Type:  mediatype.JSON,
				Title: "Properties",
			},
		},
	}
	if pubSubResourceDataConfigured(h.cfg) || pubSubResourceEventsConfigured(h.cfg) || pubSubBatchResourceEventsConfigured(h.cfg) {
		landingPage.Links = append(landingPage.Links, common_shared.Link{
			Href:  baseURL + "/asyncapi",
			Rel:   "service-desc",
			Type:  asyncAPIContentType,
			Title: "Pub/Sub MQTT service definition",
		})
	}

	render.JSON(w, r, landingPage)
}
