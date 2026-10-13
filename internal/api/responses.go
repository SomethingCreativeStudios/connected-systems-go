package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"go.uber.org/zap"
)

// negotiate resolves the response representation from the f query
// parameter, or else the Accept header. It answers 406 when the resource has
// none of the requested representations.
func negotiate[T any](w http.ResponseWriter, r *http.Request, fc *formaters.MultiFormatFormatterCollection[T]) (string, bool) {
	w.Header().Add("Vary", "Accept")
	accepted := mediatype.ParseAccept(r.Header.Get("Accept"))
	if f := r.URL.Query().Get("f"); f != "" {
		accepted = []mediatype.Range{{Type: mediatype.FromFormatParam(f), Q: 1}}
	}
	contentType, ok := fc.Negotiate(accepted)
	if !ok {
		writeError(w, r, http.StatusNotAcceptable, "none of the requested media types is available; this resource is served as "+strings.Join(fc.ContentTypes(), ", "))
	}
	return contentType, ok
}

// writeResource answers with one resource in the representation the request
// accepts.
func writeResource[T any](w http.ResponseWriter, r *http.Request, logger *zap.Logger, fc *formaters.MultiFormatFormatterCollection[T], item T) {
	response, ok := prepareResource(w, r, logger, fc, item)
	if ok {
		response.write(w, http.StatusOK)
	}
}

type preparedResponse struct {
	contentType string
	body        []byte
}

func (response preparedResponse) write(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", response.contentType)
	w.WriteHeader(status)
	_, _ = w.Write(response.body)
}

func prepareResource[T any](w http.ResponseWriter, r *http.Request, logger *zap.Logger, fc *formaters.MultiFormatFormatterCollection[T], item T) (preparedResponse, bool) {
	contentType, ok := negotiate(w, r, fc)
	if !ok {
		return preparedResponse{}, false
	}
	body, err := fc.SerializeContext(r.Context(), contentType, item)
	if err != nil {
		writeSerializeError(w, r, logger, err)
		return preparedResponse{}, false
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		writeSerializeError(w, r, logger, err)
		return preparedResponse{}, false
	}
	return preparedResponse{contentType: contentType, body: encoded}, true
}

// writeCollection answers with one page of resources in the representation
// the request accepts, with pagination links built from the request URL.
func writeCollection[T any](w http.ResponseWriter, r *http.Request, logger *zap.Logger, fc *formaters.MultiFormatFormatterCollection[T], baseURL string, items []T, total int64, params queryparams.QueryParams) {
	contentType, ok := negotiate(w, r, fc)
	if !ok {
		return
	}
	collection, err := fc.BuildCollectionContext(r.Context(), contentType, items, baseURL+r.URL.Path, int(total), r.URL.Query(), params)
	if err != nil {
		writeSerializeError(w, r, logger, err)
		return
	}
	encoded, err := json.Marshal(collection)
	if err != nil {
		writeSerializeError(w, r, logger, err)
		return
	}
	(preparedResponse{contentType: contentType, body: encoded}).write(w, http.StatusOK)
}
