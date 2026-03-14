package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, collections, h.cfg.API.BaseURL+r.URL.Path, int(10), r.URL.Query(), queryparams.QueryParams{})

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "collectionId")
	// Redirect known collection IDs to canonical endpoints
	switch id {
	case "systems":
		http.Redirect(w, r, "/systems", http.StatusTemporaryRedirect)
		return
	case "datastreams":
		http.Redirect(w, r, "/datastreams", http.StatusTemporaryRedirect)
		return
	case "observations":
		http.Redirect(w, r, "/observations", http.StatusTemporaryRedirect)
		return
	case "controlstreams":
		http.Redirect(w, r, "/controlstreams", http.StatusTemporaryRedirect)
		return
	case "commands":
		http.Redirect(w, r, "/commands", http.StatusTemporaryRedirect)
		return
	case "systemEvents":
		http.Redirect(w, r, "/systemEvents", http.StatusTemporaryRedirect)
		return
	case "deployments":
		http.Redirect(w, r, "/deployments", http.StatusTemporaryRedirect)
		return
	case "procedures":
		http.Redirect(w, r, "/procedures", http.StatusTemporaryRedirect)
		return
	case "samplingFeatures":
		http.Redirect(w, r, "/samplingFeatures", http.StatusTemporaryRedirect)
		return
	case "properties":
		http.Redirect(w, r, "/properties", http.StatusTemporaryRedirect)
		return
	}
	collection, err := h.Repo.GetCollectionByID(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Collection not found"))
		return
	}
	w.Header().Set("Content-Type", "application/json")

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, collection)
	if err != nil {
		h.logger.Error("Failed to serialize collection", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize collection"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
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
