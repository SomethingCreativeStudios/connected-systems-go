package mediatype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAccept(t *testing.T) {
	require.Empty(t, ParseAccept(""))
	require.Equal(t, []Range{
		{Type: "application/sml+json", Q: 1},
		{Type: "application/geo+json", Q: 0.9},
		{Type: "*/*", Q: 0.1},
		{Type: "text/html", Q: 0},
	}, ParseAccept("*/*;q=0.1, application/geo+json; q=0.9, Application/SML+JSON;charset=utf-8, text/html;q=0"))
	require.Equal(t, []Range{{Type: "application/json", Q: 1}}, ParseAccept("application/json;q=bogus"))
}

func TestFromFormatParam(t *testing.T) {
	require.Equal(t, GeoJSON, FromFormatParam("geojson"))
	require.Equal(t, SensorML, FromFormatParam("SMLJSON"))
	require.Equal(t, JSON, FromFormatParam("json"))
	require.Equal(t, GeoJSON, FromFormatParam("application/geo+json"))
	require.Equal(t, "xml", FromFormatParam("xml"))
}
