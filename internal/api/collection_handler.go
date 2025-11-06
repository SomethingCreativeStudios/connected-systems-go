package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

type CollectionHandler struct {
	Repo *repository.CollectionRepository
	sc   *serializers.SerializerCollection[domains.CollectionGeoJSONFeature, *domains.Collection]
}

func NewCollectionHandler(repo *repository.CollectionRepository, sc *serializers.SerializerCollection[domains.CollectionGeoJSONFeature, *domains.Collection]) *CollectionHandler {
	return &CollectionHandler{Repo: repo, sc: sc}
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

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	serialized, err := serializer.Serialize(ctx, &collection)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
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

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	var serializedCollections []domains.CollectionGeoJSONFeature

	for _, c := range collections {
		serialized, err := serializer.Serialize(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		serializedCollections = append(serializedCollections, serialized)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"collections": serializedCollections,
	})
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

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	serialized, err := serializer.Serialize(ctx, collection)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	json.NewEncoder(w).Encode(serialized)
}

// Add Create, Update, Delete if needed for admin endpoints
