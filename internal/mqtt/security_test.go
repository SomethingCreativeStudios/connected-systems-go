package mqtt

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestBrokerResumedPermissionsAreChecked(t *testing.T) {
	dir := t.TempDir()
	m := testManager(t, dir, nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "revoked", "", "", 300, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	c.disconnect(t)
	m.Publish("node1/systems:data", []byte(`{"id":"private"}`), "application/geo+json")
	m.Disconnect()
	denied := newTestIngress()
	denied.disabled = true
	m = testManager(t, dir, nil, denied)
	c, code, _ = dialWire(t, m, 5, "revoked", "", "", 300, false)
	require.Zero(t, code)
	require.NoError(t, c.send(0xc0, nil))
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0xd0), kind)
	require.GreaterOrEqual(t, c.subscribe(t, "node1/systems:data"), byte(128))
}

func TestBrokerRemovedAccountDoesNotRestoreQueue(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "users.json")
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 4)
	require.NoError(t, err)
	writeUsers := func(names ...string) {
		var users []account
		for _, name := range names {
			users = append(users, account{Username: name, PasswordHash: string(hash)})
		}
		b, _ := json.Marshal(users)
		require.NoError(t, os.WriteFile(file, b, 0600))
	}
	writeUsers("alice", "bob")
	edit := func(c *config.MQTTConfig) { c.AllowAnonymous = false; c.UsersFile = file }
	state := filepath.Join(dir, "state")
	m := testManager(t, state, edit, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "same-client", "alice", "password", 300, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	c.disconnect(t)
	m.Publish("node1/systems:data", []byte(`{"id":"alice-private"}`), "application/geo+json")
	m.Disconnect()
	writeUsers("bob")
	m = testManager(t, state, edit, newTestIngress())
	_, code, _ = dialWire(t, m, 5, "invalid", "alice", "password", 0, false)
	require.NotZero(t, code)
	c, code, present := dialWire(t, m, 5, "same-client", "bob", "password", 300, false)
	require.Zero(t, code)
	require.False(t, present)
	require.NoError(t, c.send(0xc0, nil))
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0xd0), kind)
}

func TestBrokerSimultaneousSessionOwners(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "users.json")
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 4)
	require.NoError(t, err)
	users, _ := json.Marshal([]account{{Username: "alice", PasswordHash: string(hash)}, {Username: "bob", PasswordHash: string(hash)}})
	require.NoError(t, os.WriteFile(file, users, 0600))
	m := testManager(t, filepath.Join(dir, "state"), func(c *config.MQTTConfig) { c.AllowAnonymous = false; c.UsersFile = file }, newTestIngress())
	var wg sync.WaitGroup
	codes := make(chan byte, 2)
	for _, user := range []string{"alice", "bob"} {
		wg.Go(func() { _, code, _ := dialWire(t, m, 5, "contested", user, "password", 300, false); codes <- code })
	}
	wg.Wait()
	close(codes)
	accepted := 0
	for code := range codes {
		if code == 0 {
			accepted++
		}
	}
	require.Equal(t, 1, accepted)
}

func TestBrokerNoLocalAndPacketLimit(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "no-local", "", "", 0, false)
	require.Zero(t, code)
	b := append([]byte{0, 1, 0}, wireString("node1/systems:data")...)
	b = append(b, 6)
	require.NoError(t, c.send(0x82, b))
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x90), kind)
	kind, _ = c.create(t, "local-create", 1, 7)
	require.Equal(t, byte(0x40), kind)
	require.NoError(t, c.send(0xc0, nil))
	kind, _, err = c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0xd0), kind)
	oversized, code, _ := dialWire(t, m, 5, "oversized", "", "", 0, false)
	require.Zero(t, code)
	_, err = oversized.Write(append([]byte{0x30}, wireLength(MaxPacketBytes+1)...))
	require.NoError(t, err)
	_, _, err = oversized.packet()
	require.Error(t, err)
	var timeout net.Error
	if errors.As(err, &timeout) {
		require.False(t, timeout.Timeout(), "oversized header should disconnect before reading its body")
	}
}

func TestBrokerStorageAndUserFileFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-directory")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	cfg := config.MQTTConfig{AllowAnonymous: true, NodeID: "node1", TCPAddress: "127.0.0.1:0", WebsocketAddress: "127.0.0.1:0", DataDir: file, IngestTimeout: time.Second, MaxMessageBytes: 10 << 20}
	m := NewManager(cfg, zap.NewNop())
	require.Error(t, m.Start(newTestIngress()))
	m.Disconnect()
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 4)
	require.NoError(t, err)
	users, _ := json.Marshal([]account{{Username: "same", PasswordHash: string(hash)}, {Username: "same", PasswordHash: string(hash)}})
	file = filepath.Join(dir, "users.json")
	require.NoError(t, os.WriteFile(file, users, 0600))
	_, err = loadAccounts(file, false)
	require.ErrorContains(t, err, "duplicate")
}
