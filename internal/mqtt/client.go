package mqtt

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// Config holds MQTT connection configuration.
type Config struct {
	Broker   string
	ClientID string
	Username string
	Password string
	QoS      byte
	Retained bool
}

// Manager wraps an MQTT client connection and provides publish/subscribe helpers
// for OGC Connected Systems Pub/Sub topics.
type Manager struct {
	cfg             Config
	logger          *zap.Logger
	client          mqtt.Client
	mu              sync.Mutex
	publishedEchoes map[string]publishedEcho
	subscriptions   []string
}

type publishedEcho struct {
	count     int
	expiresAt time.Time
}

const publishedEchoTTL = 30 * time.Second

// NewManager creates a new MQTT Manager. Call Connect() to establish the connection.
func NewManager(cfg Config, logger *zap.Logger) *Manager {
	return &Manager{
		cfg:             cfg,
		logger:          logger,
		publishedEchoes: make(map[string]publishedEcho),
	}
}

// Connect establishes the MQTT connection. Safe to call once.
func (m *Manager) Connect() error {
	opts := mqtt.NewClientOptions().
		AddBroker(m.cfg.Broker).
		SetClientID(m.cfg.ClientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			m.logger.Warn("MQTT connection lost", zap.Error(err))
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			m.logger.Info("MQTT connected", zap.String("broker", m.cfg.Broker))
		})

	if m.cfg.Username != "" {
		opts.SetUsername(m.cfg.Username)
	}
	if m.cfg.Password != "" {
		opts.SetPassword(m.cfg.Password)
	}

	m.client = mqtt.NewClient(opts)
	token := m.client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return fmt.Errorf("mqtt connect: timed out waiting for broker %s", m.cfg.Broker)
	}
	if token.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", token.Error())
	}

	return nil
}

// Disconnect gracefully closes the MQTT connection.
func (m *Manager) Disconnect() {
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(500)
		m.logger.Info("MQTT disconnected")
	}
}

// IsConnected returns whether the MQTT client is currently connected.
func (m *Manager) IsConnected() bool {
	return m.client != nil && m.client.IsConnected()
}

// Publish sends a message to the given topic. Non-blocking — errors are logged.
func (m *Manager) Publish(topic string, payload []byte) {
	if !m.IsConnected() {
		m.logger.Warn("MQTT not connected, skipping publish", zap.String("topic", topic))
		return
	}

	if m.hasMatchingSubscription(topic) {
		m.rememberPublishedMessage(topic, payload)
	}
	token := m.client.Publish(topic, m.cfg.QoS, m.cfg.Retained, payload)
	// Fire-and-forget: don't block the caller. Log errors asynchronously.
	go func() {
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			m.logger.Error("MQTT publish failed",
				zap.String("topic", topic),
				zap.Error(token.Error()),
			)
		}
	}()
}

// Subscribe registers a message handler for the given topic.
func (m *Manager) Subscribe(topic string, handler mqtt.MessageHandler) error {
	if !m.IsConnected() {
		return fmt.Errorf("mqtt not connected")
	}

	wrapper := func(client mqtt.Client, message mqtt.Message) {
		if m.consumePublishedEcho(message.Topic(), message.Payload()) {
			m.logger.Debug("Ignored locally published MQTT echo", zap.String("topic", message.Topic()))
			return
		}
		handler(client, message)
	}
	token := m.client.Subscribe(topic, m.cfg.QoS, wrapper)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt subscribe %s: timed out waiting for broker", topic)
	}
	if token.Error() != nil {
		return fmt.Errorf("mqtt subscribe %s: %w", topic, token.Error())
	}
	m.mu.Lock()
	m.subscriptions = append(m.subscriptions, topic)
	m.mu.Unlock()

	m.logger.Info("MQTT subscribed", zap.String("topic", topic))
	return nil
}

func (m *Manager) rememberPublishedMessage(topic string, payload []byte) {
	key := publishedMessageKey(topic, payload)
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prunePublishedEchoes(now)
	echo := m.publishedEchoes[key]
	echo.count++
	echo.expiresAt = now.Add(publishedEchoTTL)
	m.publishedEchoes[key] = echo
}

func (m *Manager) consumePublishedEcho(topic string, payload []byte) bool {
	key := publishedMessageKey(topic, payload)
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prunePublishedEchoes(now)
	echo, exists := m.publishedEchoes[key]
	if !exists {
		return false
	}
	echo.count--
	if echo.count <= 0 {
		delete(m.publishedEchoes, key)
	} else {
		m.publishedEchoes[key] = echo
	}
	return true
}

func (m *Manager) prunePublishedEchoes(now time.Time) {
	for key, echo := range m.publishedEchoes {
		if !echo.expiresAt.After(now) {
			delete(m.publishedEchoes, key)
		}
	}
}

func publishedMessageKey(topic string, payload []byte) string {
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%s\x00%x", topic, digest)
}

// Unsubscribe removes a subscription for the given topic.
func (m *Manager) Unsubscribe(topic string) error {
	if !m.IsConnected() {
		return nil
	}

	token := m.client.Unsubscribe(topic)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("mqtt unsubscribe %s: timed out waiting for broker", topic)
	}
	if token.Error() != nil {
		return fmt.Errorf("mqtt unsubscribe %s: %w", topic, token.Error())
	}
	m.mu.Lock()
	for i, subscription := range m.subscriptions {
		if subscription == topic {
			m.subscriptions = append(m.subscriptions[:i], m.subscriptions[i+1:]...)
			break
		}
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) hasMatchingSubscription(topic string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, filter := range m.subscriptions {
		if topicMatchesFilter(topic, filter) {
			return true
		}
	}
	return false
}

func topicMatchesFilter(topic, filter string) bool {
	topicParts := strings.Split(topic, "/")
	filterParts := strings.Split(filter, "/")
	for i, filterPart := range filterParts {
		if filterPart == "#" {
			return i == len(filterParts)-1
		}
		if i >= len(topicParts) || (filterPart != "+" && filterPart != topicParts[i]) {
			return false
		}
	}
	return len(topicParts) == len(filterParts)
}
