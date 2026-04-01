package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
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

func (h *CollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var collection domains.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request payload"))
		return
	}
	if collection.ID == "" {
		collection.ID = uuid.New().String()
	}
	if collection.ItemType == "" {
		collection.ItemType = "feature"
	}
	ctx := r.Context()
	if err := h.Repo.CreateCollection(ctx, &collection); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusCreated)

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, &collection)
	if err != nil {
		h.logger.Error("Failed to serialize collection", zap.String("id", collection.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize collection"})
		return
	}

	json.NewEncoder(w).Encode(serialized)
}

func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	collections, err := h.Repo.ListCollections(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	collections = ensureCanonicalCollections(collections, h.cfg.API.BaseURL)

	// OGC API - Common requires /collections to return { "links": [...], "collections": [...] }
	// not a GeoJSON FeatureCollection, so we bypass the formatter here.
	type collectionsResponse struct {
		Links          common_shared.Links  `json:"links"`
		Collections    []*domains.Collection `json:"collections"`
		NumberMatched  int                  `json:"numberMatched"`
		NumberReturned int                  `json:"numberReturned"`
	}

	resp := collectionsResponse{
		Links:          common_shared.Links{},
		Collections:    collections,
		NumberMatched:  len(collections),
		NumberReturned: len(collections),
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, resp)
}

func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "collectionId")

	// For canonical collections, return their metadata directly rather than
	// redirecting (a redirect would return the items list, not the collection record).
	canonicals := ensureCanonicalCollections(nil, h.cfg.API.BaseURL)
	for _, c := range canonicals {
		if c.ID == id {
			w.Header().Set("Content-Type", "application/json")
			render.JSON(w, r, c)
			return
		}
	}

	collection, err := h.Repo.GetCollectionByID(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Collection not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, collection)
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
		newCanonicalCollection(baseURL, "systems", "Systems", "All system instances (sensors, actuators, platforms, etc.)", "feature", "application/geo+json"),
		newCanonicalCollection(baseURL, "deployments", "Deployments", "System deployment descriptions", "feature", "application/geo+json"),
		newCanonicalCollection(baseURL, "procedures", "Procedures", "System datasheets and methodologies", "feature", "application/geo+json"),
		newCanonicalCollection(baseURL, "samplingFeatures", "Sampling Features", "Sampling strategies and geometries", "feature", "application/geo+json"),
		newCanonicalCollection(baseURL, "properties", "Properties", "Observable and controllable property definitions", "feature", "application/json"),
		newCanonicalCollection(baseURL, "datastreams", "Datastreams", "Observation datastream metadata resources", "feature", "application/json"),
		newCanonicalCollection(baseURL, "observations", "Observations", "Observation resources across datastreams", "feature", "application/json"),
		newCanonicalCollection(baseURL, "controlstreams", "Control Streams", "Tasking/control channel metadata resources", "feature", "application/json"),
		newCanonicalCollection(baseURL, "commands", "Commands", "Command resources issued through control streams", "feature", "application/json"),
		newCanonicalCollection(baseURL, "systemEvents", "System Events", "System event resources", "feature", "application/json"),
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

func newCanonicalCollection(baseURL, id, title, description, itemType, mediaType string) *domains.Collection {
	trimmedBaseURL := strings.TrimRight(baseURL, "/")
	if trimmedBaseURL == "" {
		trimmedBaseURL = "/"
	}

	return &domains.Collection{
		ID:          id,
		Title:       title,
		Description: description,
		ItemType:    itemType,
		Links: []common_shared.Link{
			{Href: trimmedBaseURL + "/collections/" + id, Rel: "self", Type: "application/json"},
			{Href: trimmedBaseURL + "/" + id, Rel: "items", Type: mediaType},
		},
	}
}

// Add Create, Update, Delete if needed for admin endpoints
