package api

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
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
func (h *LandingHandler) GetLandingPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.API.BaseURL

	landingPage := model.LandingPage{
		Title:       h.cfg.API.Title,
		Description: h.cfg.API.Description,
		Links: common_shared.Links{
			{
				Href:  baseURL + "/",
				Rel:   "self",
				Type:  "application/json",
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
				Type:  "application/json",
				Title: "Conformance declaration",
			},
			{
				Href:  baseURL + "/collections",
				Rel:   "data",
				Type:  "application/json",
				Title: "Collections",
			},
			{
				Href:  baseURL + "/systems",
				Rel:   "data",
				Type:  "application/geo+json",
				Title: "Systems",
			},
			{
				Href:  baseURL + "/deployments",
				Rel:   "data",
				Type:  "application/geo+json",
				Title: "Deployments",
			},
			{
				Href:  baseURL + "/procedures",
				Rel:   "data",
				Type:  "application/geo+json",
				Title: "Procedures",
			},
			{
				Href:  baseURL + "/samplingFeatures",
				Rel:   "data",
				Type:  "application/geo+json",
				Title: "Sampling Features",
			},
			{
				Href:  baseURL + "/properties",
				Rel:   "data",
				Type:  "application/json",
				Title: "Properties",
			},
		},
	}

	render.JSON(w, r, landingPage)
}
