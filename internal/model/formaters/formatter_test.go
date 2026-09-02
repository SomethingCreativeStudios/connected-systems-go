package formaters

import (
	"context"
	"io"
	"strings"
	"testing"

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
	collection := NewMultiFormatFormatterCollection[string]("application/test+json")
	RegisterFormatterTyped(collection, "application/test+json", replayTestFormatter{})
	RegisterFormatterTypedDefault(collection, replayTestFormatter{}, "application/test+json")

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
