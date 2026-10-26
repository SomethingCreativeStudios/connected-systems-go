package mqtt

import (
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/mochi-mqtt/server/v2/listeners"
)

type tcpListener struct {
	address string
	net.Listener
	log       *slog.Logger
	wrap      func(net.Conn) net.Conn
	closeOnce sync.Once
}

func (*tcpListener) ID() string       { return "tcp" }
func (*tcpListener) Protocol() string { return "tcp" }
func (l *tcpListener) Address() string {
	if l.Listener != nil {
		return l.Addr().String()
	}
	return l.address
}
func (l *tcpListener) Init(log *slog.Logger) error {
	l.log = log
	if l.log == nil {
		l.log = slog.Default()
	}
	var err error
	l.Listener, err = net.Listen("tcp", l.address)
	return err
}
func (l *tcpListener) Serve(establish listeners.EstablishFn) {
	var delay time.Duration
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Accept errors such as EMFILE are usually transient. Back off and
			// retry, as net/http does, rather than stopping the whole API.
			delay = min(max(2*delay, 5*time.Millisecond), time.Second)
			l.log.Warn("MQTT accept failed; retrying", "error", err, "delay", delay)
			time.Sleep(delay)
			continue
		}
		delay = 0
		conn = l.wrap(conn)
		go func() { _ = establish(l.ID(), conn) }()
	}
}
func (l *tcpListener) Close(closeClients listeners.CloseFn) {
	l.closeOnce.Do(func() { _ = l.Listener.Close(); closeClients(l.ID()) })
}
