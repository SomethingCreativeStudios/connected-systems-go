package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"
)

// Opt-in check against the isolated Compose API and a real Traefik container.
// TLS hostname and CA verification stay enabled, with only TCP dialing redirected
// to loopback so this test never connects to the actual dev-server hostname.
func TestBrokerExternalTraefikWSS(t *testing.T) {
	address := os.Getenv("MQTT_TEST_WSS_URL")
	if address == "" {
		t.Skip("set MQTT_TEST_WSS_URL, MQTT_TEST_CA, MQTT_TEST_DIAL_ADDRESS and MQTT_TEST_API_URL for live Traefik verification")
	}
	pem, err := os.ReadFile(os.Getenv("MQTT_TEST_CA"))
	require.NoError(t, err)
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(pem))
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, os.Getenv("MQTT_TEST_DIAL_ADDRESS"))
	}
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, NetDialContext: dial, Subprotocols: []string{"mqtt"}}
	socket, _, err := dialer.Dial(address, nil)
	require.NoError(t, err)
	defer socket.Close()
	c := &wireClient{Conn: &websocketConn{Conn: socket.UnderlyingConn(), socket: socket}, version: 5}
	flags := byte(2)
	user := os.Getenv("MQTT_TEST_USERNAME")
	password := os.Getenv("MQTT_TEST_PASSWORD")
	if user != "" {
		flags |= 192
	}
	body := append(wireString("MQTT"), 5, flags, 0, 30, 0)
	body = append(body, wireString("wss-"+uuid.NewString())...)
	if user != "" {
		body = append(body, wireString(user)...)
		body = append(body, wireString(password)...)
	}
	require.NoError(t, c.send(0x10, body))
	kind, ack, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x20), kind)
	require.Zero(t, ack[1])
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	name := "Traefik WSS " + uuid.NewString()
	payload, _ := json.Marshal(map[string]any{"type": "Feature", "geometry": nil, "properties": map[string]any{"uid": "urn:uuid:" + uuid.NewString(), "name": name, "featureType": "http://www.w3.org/ns/sosa/Sensor", "assetType": "Equipment"}})
	props := append([]byte{3}, wireString("application/geo+json")...)
	body = append(wireString("node1/systems:data"), 0, 7)
	body = append(body, wireLength(len(props))...)
	body = append(body, props...)
	body = append(body, payload...)
	require.NoError(t, c.send(0x32, body))
	var resource map[string]any
	for range 2 {
		kind, body, err = c.packet()
		require.NoError(t, err)
		if kind == 0x40 {
			require.True(t, len(body) == 2 || body[2] < 128)
			continue
		}
		require.Equal(t, byte(3), kind>>4)
		pk := packets.Packet{ProtocolVersion: 5, FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: (kind >> 1) & 3}}
		require.NoError(t, pk.PublishDecode(body))
		require.Equal(t, "application/geo+json", pk.Properties.ContentType)
		require.NoError(t, json.Unmarshal(pk.Payload, &resource))
	}
	require.NotEmpty(t, resource["id"])
	apiURL := os.Getenv("MQTT_TEST_API_URL") + "/systems/" + resource["id"].(string)
	response, err := http.Get(apiURL)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	got, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Contains(t, string(got), name)
	request, err := http.NewRequest(http.MethodDelete, apiURL, nil)
	require.NoError(t, err)
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, 204, response.StatusCode)
}
