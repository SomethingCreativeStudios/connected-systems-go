package formaters

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/stretchr/testify/require"
)

type replayTestFormatter struct{}

func (replayTestFormatter) Serialize(context.Context, string) (string, error) { return "", nil }
func (replayTestFormatter) SerializeAll(context.Context, []string) ([]string, error) {
	return nil, nil
}
func (replayTestFormatter) Deserialize(_ context.Context, reader io.Reader) (string, error) {
	body, err := io.ReadAll(reader)
	return string(body), err
}
func (replayTestFormatter) ContentType() string { return "application/test+json" }

func TestDeserializeValidatesRawBodyAndReplaysItForFormatter(t *testing.T) {
	collection := NewFormatterCollection(Adapt[string, string](replayTestFormatter{}))

	var validatedContentType string
	var validatedBody []byte
	collection.SetRequestValidator(func(contentType string, body []byte) error {
		validatedContentType = contentType
		validatedBody = append([]byte(nil), body...)
		return nil
	})

	decoded, err := collection.Deserialize("application/test+json; charset=utf-8", strings.NewReader(`{"value":42}`))
	require.NoError(t, err)
	require.Equal(t, `{"value":42}`, decoded)
	require.Equal(t, "application/test+json", validatedContentType)
	require.Equal(t, []byte(`{"value":42}`), validatedBody)
}

// A nil resource is an error rather than an index-out-of-range panic.
func TestSerializeOneRejectsNil(t *testing.T) {
	called := false
	_, err := SerializeOne(context.Background(), (*string)(nil), func(context.Context, []*string) ([]string, error) {
		called = true
		return nil, nil
	})
	require.Error(t, err)
	require.False(t, called)

	value := "x"
	out, err := SerializeOne(context.Background(), &value, func(_ context.Context, items []*string) ([]string, error) {
		return []string{*items[0] + "!"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "x!", out)
}

type contentTypeFormatter struct {
	replayTestFormatter
	contentType string
}

func (f contentTypeFormatter) ContentType() string { return f.contentType }

func TestNegotiate(t *testing.T) {
	collection := NewFormatterCollection(
		Adapt[string, string](contentTypeFormatter{contentType: mediatype.GeoJSON}),
		Adapt[string, string](contentTypeFormatter{contentType: mediatype.SensorML}),
	)
	cases := map[string]struct {
		accept string
		want   string
		ok     bool
	}{
		"no accept header":                        {"", mediatype.GeoJSON, true},
		"exact":                                   {"application/sml+json", mediatype.SensorML, true},
		"wildcard":                                {"*/*", mediatype.GeoJSON, true},
		"plain json is the default":               {"application/json", mediatype.GeoJSON, true},
		"quality order":                           {"application/geo+json;q=0.5, application/sml+json", mediatype.SensorML, true},
		"named beats fallback":                    {"application/json, application/sml+json", mediatype.SensorML, true},
		"lower-q fallback":                        {"application/xml, */*;q=0.1", mediatype.GeoJSON, true},
		"nothing acceptable":                      {"application/xml, text/html", "", false},
		"explicit exclusion":                      {"application/geo+json;q=0, */*", mediatype.SensorML, true},
		"all excluded":                            {"application/geo+json;q=0, application/sml+json;q=0, */*", "", false},
		"alias cannot revive excluded default":    {"application/geo+json;q=0, application/json", "", false},
		"specific quality beats wildcard quality": {"application/geo+json;q=0.2, */*;q=0.8", mediatype.SensorML, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := collection.Negotiate(mediatype.ParseAccept(tc.accept))
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
	require.Equal(t, []string{mediatype.GeoJSON, mediatype.SensorML}, collection.ContentTypes())
}
