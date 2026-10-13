package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/queryables"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

type CollectionHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	Repo   *repository.CollectionRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Collection]
}

func NewCollectionHandler(cfg *config.Config, logger *zap.Logger, repo *repository.CollectionRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Collection]) *CollectionHandler {
	return &CollectionHandler{cfg: cfg, Repo: repo, logger: logger, fc: fc}
}

// CreateCollection creates a new collection
//
// @Summary     Create collection
// @Description Creates a new feature collection
// @Tags        Collections
// @Accept      json
// @Produce     json
// @Param       collection  body  map[string]any  true  "Collection resource"
// @Success     201  {object}  map[string]any
// @Failure     400
// @Failure     500
// @Router      /collections [post]
func (h *CollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	// Catalog declarations are accepted on creation, but are exposed only by
	// the queryables endpoint rather than collection representations.
	var input struct {
		domains.Collection
		Queryables common_shared.Properties `json:"queryables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeValidationError(w, r, err)
		return
	}
	collection := input.Collection
	collection.Queryables = input.Queryables
	if collection.ID == "" {
		collection.ID = uuid.New().String()
	}
	if collection.ItemType == "" {
		collection.ItemType = "feature"
	}
	if collection.ItemType == "feature" {
		if err := queryables.ValidateCustomCatalog(collection.Queryables); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}
	ensureCollectionQueryablesLink(&collection, h.cfg.API.BaseURL)
	response, ok := prepareResource(w, r, h.logger, h.fc, &collection)
	if !ok {
		return
	}
	ctx := r.Context()
	if err := h.Repo.CreateCollection(ctx, &collection); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Collection", "Failed to create collection")
		return
	}
	w.Header().Set("Location", strings.TrimRight(h.cfg.API.BaseURL, "/")+"/collections/"+collection.ID)
	response.write(w, http.StatusCreated)
}

// GetQueryables returns the OGC API Features Part 3 JSON Schema catalog for a
// canonical CS collection, nested resource catalog, or a generic collection.
//
// @Summary Get collection queryables
// @Description Returns the declared CQL2 queryable properties as JSON Schema
// @Tags Collections
// @Produce application/schema+json
// @Param collectionId path string true "Collection ID"
// @Success 200 {object} map[string]any
// @Failure 404 {object} map[string]string
// @Router /collections/{collectionId}/queryables [get]
func (h *CollectionHandler) GetQueryables(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "collectionId")
	resource := queryables.Resource(id)
	if _, ok := queryables.PropertiesFor(resource); !ok {
		collection, err := h.Repo.GetCollectionByID(r.Context(), id)
		if err != nil {
			writeRepositoryError(w, r, h.logger, err, "Collection", "Failed to load collection")
			return
		}
		resource = queryables.Features
		schema, err := queryables.SchemaWithCatalog(resource, h.cfg.API.BaseURL+r.URL.Path, queryables.CatalogOrDefault(collection.Queryables))
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/schema+json")
		writeNegotiated(w, schema)
		return
	}
	schema, err := queryables.Schema(resource, h.cfg.API.BaseURL+r.URL.Path)
	if err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/schema+json")
	writeNegotiated(w, schema)
}

// ListCollections returns all collections
//
// @Summary     List collections
// @Description Returns all feature collections including canonical OGC CS resource collections
// @Tags        Collections
// @Produce     json
// @Success     200  {object}  map[string]any
// @Failure     500
// @Router      /collections [get]
func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	collections, err := h.Repo.ListCollections(ctx)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Collection", "Failed to list collections")
		return
	}

	collections = ensureCanonicalCollections(collections, h.cfg.API.BaseURL)
	for _, collection := range collections {
		ensureCollectionQueryablesLink(collection, h.cfg.API.BaseURL)
	}

	// OGC API - Common requires /collections to return { "links": [...], "collections": [...] }
	// not a GeoJSON FeatureCollection, so we bypass the formatter here.
	type collectionsResponse struct {
		Links          common_shared.Links   `json:"links"`
		Collections    []*domains.Collection `json:"collections"`
		NumberMatched  int                   `json:"numberMatched"`
		NumberReturned int                   `json:"numberReturned"`
	}

	resp := collectionsResponse{
		Links:          common_shared.Links{},
		Collections:    collections,
		NumberMatched:  len(collections),
		NumberReturned: len(collections),
	}

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, resp)
}

// GetCollection returns a single collection by ID
//
// @Summary     Get collection
// @Description Returns a single feature collection by ID
// @Tags        Collections
// @Produce     json
// @Param       collectionId  path  string  true  "Collection ID"
// @Success     200  {object}  map[string]any
// @Failure     404
// @Router      /collections/{collectionId} [get]
func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "collectionId")

	// For canonical collections, return their metadata directly rather than
	// redirecting (a redirect would return the items list, not the collection record).
	canonicals := ensureCanonicalCollections(nil, h.cfg.API.BaseURL)
	for _, c := range canonicals {
		if c.ID == id {
			w.Header().Set("Content-Type", mediatype.JSON)
			render.JSON(w, r, c)
			return
		}
	}

	collection, err := h.Repo.GetCollectionByID(ctx, id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Collection", "Failed to load collection")
		return
	}
	ensureCollectionQueryablesLink(collection, h.cfg.API.BaseURL)

	w.Header().Set("Content-Type", mediatype.JSON)
	writeNegotiated(w, collection)
}

func ensureCollectionQueryablesLink(collection *domains.Collection, baseURL string) {
	if collection == nil || collection.ID == "" {
		return
	}
	for _, link := range collection.Links {
		if link.Rel == "http://www.opengis.net/def/rel/ogc/1.0/queryables" {
			return
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	collection.Links = append(collection.Links, common_shared.Link{
		Href: baseURL + "/collections/" + collection.ID + "/queryables",
		Rel:  "http://www.opengis.net/def/rel/ogc/1.0/queryables",
		Type: "application/schema+json",
	})
}

func ensureCanonicalCollections(existing []*domains.Collection, baseURL string) []*domains.Collection {
	byID := make(map[string]struct{}, len(existing))
	for _, c := range existing {
		if c == nil {
			continue
		}
		byID[c.ID] = struct{}{}
	}

	defaults := []*domains.Collection{
		newCanonicalCollection(baseURL, "systems", "Systems", "All system instances (sensors, actuators, platforms, etc.)", "feature", mediatype.GeoJSON, "sosa:System"),
		newCanonicalCollection(baseURL, "deployments", "Deployments", "System deployment descriptions", "feature", mediatype.GeoJSON, "sosa:Deployment"),
		newCanonicalCollection(baseURL, "procedures", "Procedures", "System datasheets and methodologies", "feature", mediatype.SensorML, "sosa:Procedure"),
		newCanonicalCollection(baseURL, "samplingFeatures", "Sampling Features", "Sampling strategies and geometries", "feature", mediatype.GeoJSON, "sosa:Sample"),
		newCanonicalCollection(baseURL, "properties", "Properties", "Observable and controllable property definitions", "record", mediatype.SensorML, "sosa:Property"),
		newCanonicalCollection(baseURL, "datastreams", "Datastreams", "Observation datastream metadata resources", "record", mediatype.JSON, ""),
		newCanonicalCollection(baseURL, "observations", "Observations", "Observation resources across datastreams", "record", mediatype.JSON, ""),
		newCanonicalCollection(baseURL, "controlstreams", "Control Streams", "Tasking/control channel metadata resources", "record", mediatype.JSON, ""),
		newCanonicalCollection(baseURL, "commands", "Commands", "Command resources issued through control streams", "record", mediatype.JSON, ""),
		newCanonicalCollection(baseURL, "systemEvents", "System Events", "System event resources", "record", mediatype.JSON, ""),
	}

	out := make([]*domains.Collection, 0, len(existing)+len(defaults))
	out = append(out, existing...)
	for _, d := range defaults {
		if _, ok := byID[d.ID]; ok {
			continue
		}
		out = append(out, d)
	}

	return out
}

func newCanonicalCollection(baseURL, id, title, description, itemType, mediaType, featureType string) *domains.Collection {
	trimmedBaseURL := strings.TrimRight(baseURL, "/")
	if trimmedBaseURL == "" {
		trimmedBaseURL = "/"
	}

	return &domains.Collection{
		ID:          id,
		Title:       title,
		Description: description,
		ItemType:    itemType,
		FeatureType: featureType,
		Links: []common_shared.Link{
			{Href: trimmedBaseURL + "/collections/" + id, Rel: "self", Type: mediatype.JSON},
			{Href: trimmedBaseURL + "/collections/" + id + "/items", Rel: "items", Type: mediaType},
			{Href: trimmedBaseURL + "/collections/" + id + "/queryables", Rel: "http://www.opengis.net/def/rel/ogc/1.0/queryables", Type: "application/schema+json"},
		},
	}
}

// Add Create, Update, Delete if needed for admin endpoints
