package mqtt

import (
	"fmt"
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
// for OGC Connected Systems Part 3 topics.
type Manager struct {
	cfg    Config
	logger *zap.Logger
	client mqtt.Client
	mu     sync.RWMutex
}

// NewManager creates a new MQTT Manager. Call Connect() to establish the connection.
func NewManager(cfg Config, logger *zap.Logger) *Manager {
	return &Manager{
		cfg:    cfg,
		logger: logger,
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
	if token.WaitTimeout(15 * time.Second); token.Error() != nil {
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

	token := m.client.Subscribe(topic, m.cfg.QoS, handler)
	if token.WaitTimeout(10 * time.Second); token.Error() != nil {
		return fmt.Errorf("mqtt subscribe %s: %w", topic, token.Error())
	}

	m.logger.Info("MQTT subscribed", zap.String("topic", topic))
	return nil
}

// Unsubscribe removes a subscription for the given topic.
func (m *Manager) Unsubscribe(topic string) error {
	if !m.IsConnected() {
		return nil
	}

	token := m.client.Unsubscribe(topic)
	if token.WaitTimeout(5 * time.Second); token.Error() != nil {
		return fmt.Errorf("mqtt unsubscribe %s: %w", topic, token.Error())
	}

	return nil
}
