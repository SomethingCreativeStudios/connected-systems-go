package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

type CollectionHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	Repo   *repository.CollectionRepository
	fc     *serializers.MultiFormatFormatterCollection[*domains.Collection]
}

func NewCollectionHandler(cfg *config.Config, logger *zap.Logger, repo *repository.CollectionRepository, fc *serializers.MultiFormatFormatterCollection[*domains.Collection]) *CollectionHandler {
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

// Add Create, Update, Delete if needed for admin endpoints
