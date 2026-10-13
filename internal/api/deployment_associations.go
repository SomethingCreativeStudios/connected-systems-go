package api

import (
	"context"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func deploymentCollection[T any, P any](h *DeploymentHandler, w http.ResponseWriter, r *http.Request, fc *formaters.MultiFormatFormatterCollection[T], parse func(*http.Request, int) (P, error), list func(context.Context, string, P) ([]T, int64, error), base func(P) queryparams.QueryParams) {
	id := chi.URLParam(r, "id")
	if _, err := h.repo.GetByID(r.Context(), id); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Deployment", "Failed to load deployment")
		return
	}
	params, err := parse(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := list(r.Context(), id, params)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Deployment association", "Failed to load deployment association")
		return
	}
	writeCollection(w, r, h.logger, fc, h.cfg.API.BaseURL, items, total, base(params))
}

// deploymentSamplingFeatures serves recursive deployment sampling features.
// @Summary List deployment sampling features
// @Description Sampling features associated with this deployment and its subdeployments.
// @Tags Deployments
// @Produce application/geo+json,application/sml+json
// @Param id path string true "Deployment ID"
// @Param limit query integer false "Maximum number of results"
// @Param cursor query string false "Opaque pagination cursor"
// @Param filter query string false "CQL2 filter"
// @Success 200 {object} map[string]any
// @Failure 400 {object} ValidationErrorResponse
// @Failure 404 {object} map[string]string
// @Router /deployments/{id}/samplingFeatures [get]
func deploymentSamplingFeatures(h *DeploymentHandler, child *SamplingFeatureHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentCollection(h, w, r, child.fc, queryparams.SamplingFeatureQueryParams{}.BuildFromRequest, child.repo.ListDeployment, func(p *queryparams.SamplingFeatureQueryParams) queryparams.QueryParams { return p.QueryParams })
	}
}

// @Summary List deployment features of interest
// @Description Local generic and native features observed or controlled during this deployment and its subdeployments.
// @Tags Deployments
// @Produce application/geo+json
// @Param id path string true "Deployment ID"
// @Param limit query integer false "Maximum number of results"
// @Param cursor query string false "Opaque pagination cursor"
// @Param filter query string false "CQL2 filter over common feature fields"
// @Success 200 {object} map[string]any
// @Failure 400 {object} ValidationErrorResponse
// @Failure 404 {object} map[string]string
// @Router /deployments/{id}/featuresOfInterest [get]
func deploymentFeaturesOfInterest(h *DeploymentHandler, child *FeatureHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentCollection(h, w, r, child.fc, queryparams.FeatureQueryParams{}.BuildFromRequest, child.repo.ListDeployment, func(p *queryparams.FeatureQueryParams) queryparams.QueryParams { return p.QueryParams })
	}
}

// @Summary List deployment datastreams
// @Description Datastreams associated with this deployment and its subdeployments, including streams of deployed subsystems.
// @Tags Deployments
// @Produce json
// @Param id path string true "Deployment ID"
// @Param limit query integer false "Maximum number of results"
// @Param cursor query string false "Opaque pagination cursor"
// @Param filter query string false "CQL2 filter"
// @Success 200 {object} DatastreamCollectionResponse
// @Failure 400 {object} ValidationErrorResponse
// @Failure 404 {object} map[string]string
// @Router /deployments/{id}/datastreams [get]
func deploymentDatastreams(h *DeploymentHandler, child *DatastreamHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentCollection(h, w, r, child.fc, queryparams.DatastreamsQueryParams{}.BuildFromRequest, child.repo.ListDeployment, func(p *queryparams.DatastreamsQueryParams) queryparams.QueryParams { return p.QueryParams })
	}
}

// @Summary List deployment controlstreams
// @Description Controlstreams associated with this deployment and its subdeployments, including streams of deployed subsystems.
// @Tags Deployments
// @Produce json
// @Param id path string true "Deployment ID"
// @Param limit query integer false "Maximum number of results"
// @Param cursor query string false "Opaque pagination cursor"
// @Param filter query string false "CQL2 filter"
// @Success 200 {object} ControlStreamCollectionResponse
// @Failure 400 {object} ValidationErrorResponse
// @Failure 404 {object} map[string]string
// @Router /deployments/{id}/controlstreams [get]
func deploymentControlstreams(h *DeploymentHandler, child *ControlStreamHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentCollection(h, w, r, child.fc, queryparams.ControlStreamsQueryParams{}.BuildFromRequest, child.repo.ListDeployment, func(p *queryparams.ControlStreamsQueryParams) queryparams.QueryParams { return p.QueryParams })
	}
}
