package queryparams

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type spatialBuildResult struct {
	bbox *common_shared.BoundingBox
	geom string
}

func TestSpatialResourceQueryParamsBuildFromRequest(t *testing.T) {
	query := url.Values{
		"bbox": {"-118,32,-117,33"},
		"geom": {"POLYGON((-118 32,-117 32,-117 33,-118 33,-118 32))"},
	}

	builders := map[string]func(*http.Request) (spatialBuildResult, error){
		"systems": func(r *http.Request) (spatialBuildResult, error) {
			params, err := (SystemQueryParams{}).BuildFromRequest(r, 10)
			if err != nil {
				return spatialBuildResult{}, err
			}
			return spatialBuildResult{bbox: params.Bbox, geom: params.Geom}, nil
		},
		"sampling features": func(r *http.Request) (spatialBuildResult, error) {
			params, err := (SamplingFeatureQueryParams{}).BuildFromRequest(r, 10)
			if err != nil {
				return spatialBuildResult{}, err
			}
			return spatialBuildResult{bbox: params.Bbox, geom: params.Geom}, nil
		},
		"deployments": func(r *http.Request) (spatialBuildResult, error) {
			params, err := (DeploymentsQueryParams{}).BuildFromRequest(r, 10)
			if err != nil {
				return spatialBuildResult{}, err
			}
			return spatialBuildResult{bbox: params.Bbox, geom: params.Geom}, nil
		},
	}

	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/resources?"+query.Encode(), nil)
			result, err := build(req)
			require.NoError(t, err)
			require.Equal(t, &common_shared.BoundingBox{MinX: -118, MinY: 32, MaxX: -117, MaxY: 33}, result.bbox)
			require.Equal(t, query.Get("geom"), result.geom)
		})
	}
}

func TestParseBoundingBox(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want *common_shared.BoundingBox
	}{
		{
			name: "2D",
			raw:  "-118,32,-117,33",
			want: &common_shared.BoundingBox{MinX: -118, MinY: 32, MaxX: -117, MaxY: 33},
		},
		{
			name: "3D uses XY bounds",
			raw:  "-118,32,10,-117,33,20",
			want: &common_shared.BoundingBox{MinX: -118, MinY: 32, MaxX: -117, MaxY: 33},
		},
		{
			name: "antimeridian",
			raw:  "170,-10,-170,10",
			want: &common_shared.BoundingBox{MinX: 170, MinY: -10, MaxX: -170, MaxY: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBoundingBox(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSpatialResourceQueryParamsRejectInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		match string
	}{
		{name: "bbox length", query: url.Values{"bbox": {"1,2,3"}}, match: "expected 4 or 6"},
		{name: "bbox number", query: url.Values{"bbox": {"1,two,3,4"}}, match: "coordinate 2"},
		{name: "bbox latitude order", query: url.Values{"bbox": {"-10,20,10,-20"}}, match: "minimum latitude"},
		{name: "bbox vertical order", query: url.Values{"bbox": {"-10,-20,5,10,20,1"}}, match: "minimum vertical"},
		{name: "invalid WKT", query: url.Values{"geom": {"NOT_A_GEOMETRY"}}, match: "invalid geom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/systems?"+tt.query.Encode(), nil)
			_, err := (SystemQueryParams{}).BuildFromRequest(req, 10)
			require.ErrorContains(t, err, tt.match)
		})
	}
}
