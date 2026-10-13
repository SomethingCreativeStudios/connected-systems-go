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

// CollectionsHandler handles collection metadata
type CollectionsHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewCollectionsHandler creates a new CollectionsHandler
func NewCollectionsHandler(cfg *config.Config, logger *zap.Logger) *CollectionsHandler {
	return &CollectionsHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// GetCollections returns all collections
//
// @Summary     Get collections metadata
// @Description Returns metadata for all OGC API resource collections
// @Tags        Collections
// @Produce     json
// @Success     200  {object}  map[string]any
// @Router      /collections [get]
func (h *CollectionsHandler) GetCollections(w http.ResponseWriter, r *http.Request) {
	baseURL := h.cfg.API.BaseURL

	collections := struct {
		Links       common_shared.Links        `json:"links"`
		Collections []model.CollectionMetadata `json:"collections"`
	}{
		Links: common_shared.Links{
			{Href: baseURL + "/collections", Rel: "self", Type: mediatype.JSON},
		},
		Collections: []model.CollectionMetadata{
			{
				ID:          "systems",
				Title:       "Systems",
				Description: "All system instances (sensors, actuators, platforms, etc.)",
				ItemType:    "feature",
				FeatureType: "sosa:System",
				Links: common_shared.Links{
					{Href: baseURL + "/collections/systems", Rel: "self"},
					{Href: baseURL + "/systems", Rel: "items", Type: mediatype.GeoJSON},
				},
			},
			{
				ID:          "deployments",
				Title:       "Deployments",
				Description: "System deployment descriptions",
				ItemType:    "feature",
				FeatureType: "sosa:Deployment",
				Links: common_shared.Links{
					{Href: baseURL + "/collections/deployments", Rel: "self"},
					{Href: baseURL + "/deployments", Rel: "items", Type: mediatype.GeoJSON},
				},
			},
			{
				ID:          "procedures",
				Title:       "Procedures",
				Description: "System datasheets and methodologies",
				ItemType:    "feature",
				FeatureType: "sosa:Procedure",
				Links: common_shared.Links{
					{Href: baseURL + "/collections/procedures", Rel: "self"},
					{Href: baseURL + "/procedures", Rel: "items", Type: mediatype.GeoJSON},
				},
			},
			{
				ID:          "samplingFeatures",
				Title:       "Sampling Features",
				Description: "Sampling strategies and geometries",
				ItemType:    "feature",
				FeatureType: "sosa:Sample",
				Links: common_shared.Links{
					{Href: baseURL + "/collections/samplingFeatures", Rel: "self"},
					{Href: baseURL + "/samplingFeatures", Rel: "items", Type: mediatype.GeoJSON},
				},
			},
			{
				ID:          "properties",
				Title:       "Properties",
				Description: "Observable and controllable property definitions",
				ItemType:    "sosa:Property",
				Links: common_shared.Links{
					{Href: baseURL + "/collections/properties", Rel: "self"},
					{Href: baseURL + "/properties", Rel: "items", Type: mediatype.JSON},
				},
			},
		},
	}

	render.JSON(w, r, collections)
}

// GetCollection returns a single collection metadata
//
// @Summary     Get collection metadata (not implemented)
// @Tags        Collections
// @Produce     json
// @Success     501  {object}  map[string]string
// @Router      /collections/{collectionId} [get]
func (h *CollectionsHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement single collection retrieval
	writeError(w, r, http.StatusNotImplemented, "Not implemented")
}

// GetCollectionItems returns items from a collection
//
// @Summary     Get collection items (not implemented)
// @Tags        Collections
// @Produce     json
// @Success     501  {object}  map[string]string
// @Router      /collections/{collectionId}/items [get]
func (h *CollectionsHandler) GetCollectionItems(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement collection items retrieval
	writeError(w, r, http.StatusNotImplemented, "Not implemented")
}
