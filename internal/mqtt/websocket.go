package mqtt

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// Bind during Init, unlike Mochi's built-in WebSocket listener, so startup
// cannot report HTTP readiness before discovering a WebSocket bind failure.
type websocketListener struct {
	address   string
	listener  net.Listener
	server    *http.Server
	establish listeners.EstablishFn
	fail      func(error)
	wrap      func(net.Conn) net.Conn
	closing   sync.Once
}

func newWebsocketListener(address string, fail func(error), wrap func(net.Conn) net.Conn) *websocketListener {
	return &websocketListener{address: address, fail: fail, wrap: wrap}
}
func (*websocketListener) ID() string       { return "websocket" }
func (*websocketListener) Protocol() string { return "ws" }
func (l *websocketListener) Address() string {
	if l.listener != nil {
		return l.listener.Addr().String()
	}
	return l.address
}
func (l *websocketListener) Init(*slog.Logger) error {
	var err error
	l.listener, err = net.Listen("tcp", l.address)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /mqtt", l.upgrade)
	l.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5e9}
	return nil
}
func (l *websocketListener) Serve(establish listeners.EstablishFn) {
	l.establish = establish
	if err := l.server.Serve(l.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.fail(err)
	}
}
func (l *websocketListener) Close(closeClients listeners.CloseFn) {
	l.closing.Do(func() { _ = l.server.Close(); closeClients(l.ID()) })
}
func (l *websocketListener) upgrade(w http.ResponseWriter, r *http.Request) {
	// Any origin is accepted: credentials travel in MQTT CONNECT rather than in
	// cookies, so a cross-origin page gains nothing over a direct MQTT client.
	// Deployments that need an origin policy enforce it at the TLS proxy.
	upgrader := websocket.Upgrader{Subprotocols: []string{"mqtt"}, CheckOrigin: func(*http.Request) bool { return true }}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c.SetReadLimit(MaxPacketBytes)
	defer c.Close()
	_ = l.establish(l.ID(), l.wrap(&websocketConn{Conn: c.UnderlyingConn(), socket: c}))
}

type websocketConn struct {
	net.Conn
	socket *websocket.Conn
	reader io.Reader
	mu     sync.Mutex
}

func (c *websocketConn) Read(b []byte) (int, error) {
	for {
		if c.reader == nil {
			kind, r, err := c.socket.NextReader()
			if err != nil {
				return 0, err
			}
			if kind != websocket.BinaryMessage {
				return 0, errors.New("MQTT WebSocket requires binary frames")
			}
			c.reader = r
		}
		n, err := c.reader.Read(b)
		if errors.Is(err, io.EOF) {
			c.reader = nil
			if n == 0 {
				continue
			}
			return n, nil
		}
		return n, err
	}
}
func (c *websocketConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.socket.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}
