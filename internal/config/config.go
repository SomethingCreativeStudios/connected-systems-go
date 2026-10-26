package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	API      APIConfig      `mapstructure:"api"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	PubSub   PubSubConfig   `mapstructure:"pubsub"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslmode"`

	// Connection pool limits; zero keeps the database/sql default.
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

	// AutoMigrate applies pending migrations at startup. When disabled, run
	// `server migrate` before starting the server.
	AutoMigrate bool `mapstructure:"auto_migrate"`
}

// DSN is the connection URL for the database. Building it as a URL escapes
// credentials that contain spaces, quotes or other special characters.
func (d DatabaseConfig) DSN() string {
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:     "/" + d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	return dsn.String()
}

// APIConfig holds API-specific configuration
type APIConfig struct {
	BaseURL      string `mapstructure:"base_url"`
	Title        string `mapstructure:"title"`
	Description  string `mapstructure:"description"`
	Version      string `mapstructure:"version"`
	DefaultLimit int    `mapstructure:"default_limit"`
	// MaxBodyBytes caps request bodies; 0 means the 10 MiB default.
	MaxBodyBytes int64 `mapstructure:"max_body_bytes"`
}

// MQTTConfig configures the optional embedded CS API MQTT endpoint.
type MQTTConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	AllowAnonymous   bool          `mapstructure:"allow_anonymous"`
	NodeID           string        `mapstructure:"node_id"`
	PublicBrokers    string        `mapstructure:"public_brokers"`
	TCPAddress       string        `mapstructure:"tcp_address"`
	WebsocketAddress string        `mapstructure:"websocket_address"`
	DataDir          string        `mapstructure:"data_dir"`
	UsersFile        string        `mapstructure:"users_file"`
	IngestTimeout    time.Duration `mapstructure:"ingest_timeout"`
	MaxMessageBytes  int64         `mapstructure:"max_message_bytes"`
}

// PubSubConfig controls the independently optional publish/subscribe message classes.
// MQTT remains the transport-level master switch; these flags decide which
// message classes are active once an MQTT transport is available.
type PubSubConfig struct {
	ResourceData        PubSubFeatureConfig       `mapstructure:"resource_data"`
	ResourceEvents      PubSubFeatureConfig       `mapstructure:"resource_events"`
	BatchResourceEvents BatchResourceEventsConfig `mapstructure:"batch_resource_events"`
}

// PubSubFeatureConfig is intentionally nested so class-specific settings can
// be added later without changing the environment-variable shape.
type PubSubFeatureConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// BatchResourceEventsConfig controls aggregation of high-volume lifecycle
// events. Window is a clock-aligned tumbling-window duration.
type BatchResourceEventsConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Window  time.Duration `mapstructure:"window"`
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "connected_systems")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.auto_migrate", true)
	viper.SetDefault("api.title", "OGC Connected Systems API")
	viper.SetDefault("api.version", "1.0.0")
	viper.SetDefault("api.description", "OGC API - Connected Systems - Part 1: Feature Resources")
	viper.SetDefault("api.base_url", "http://localhost:8080")
	viper.SetDefault("api.default_limit", 10)
	viper.SetDefault("api.max_body_bytes", 10<<20)
	viper.SetDefault("mqtt.enabled", false)
	viper.SetDefault("mqtt.allow_anonymous", false)
	viper.SetDefault("mqtt.node_id", "")
	viper.SetDefault("mqtt.public_brokers", "")
	viper.SetDefault("mqtt.tcp_address", "127.0.0.1:1883")
	viper.SetDefault("mqtt.websocket_address", "127.0.0.1:9001")
	viper.SetDefault("mqtt.data_dir", "./data/mqtt")
	viper.SetDefault("mqtt.users_file", "")
	viper.SetDefault("mqtt.ingest_timeout", 2*time.Second)
	viper.SetDefault("mqtt.max_message_bytes", 10<<20)
	viper.SetDefault("pubsub.resource_data.enabled", true)
	viper.SetDefault("pubsub.resource_events.enabled", true)
	viper.SetDefault("pubsub.batch_resource_events.enabled", true)
	viper.SetDefault("pubsub.batch_resource_events.window", time.Minute)

	// Read from environment — replace "." with "_" so database.host → DATABASE_HOST
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, notFound := errors.AsType[viper.ConfigFileNotFoundError](err); !notFound {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if config.PubSub.BatchResourceEvents.Window <= 0 {
		return nil, fmt.Errorf("pubsub.batch_resource_events.window must be a positive duration")
	}
	if err := config.ValidateMQTT(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) ValidateMQTT() error {
	if !c.MQTT.Enabled {
		return nil
	}
	m := c.MQTT
	apiRoot, err := url.Parse(c.API.BaseURL)
	if err != nil || apiRoot.Host == "" || apiRoot.User != nil || apiRoot.RawQuery != "" || apiRoot.Fragment != "" || (apiRoot.Scheme != "http" && apiRoot.Scheme != "https") {
		return fmt.Errorf("MQTT requires an absolute canonical HTTP api.base_url")
	}
	if m.NodeID == "" || !utf8.ValidString(m.NodeID) || strings.ContainsAny(m.NodeID, "/+#\x00") {
		return fmt.Errorf("mqtt.node_id must be a nonempty single MQTT level")
	}
	if !m.AllowAnonymous && m.UsersFile == "" {
		return fmt.Errorf("mqtt.users_file is required when anonymous access is disabled")
	}
	for _, address := range []string{m.TCPAddress, m.WebsocketAddress} {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return fmt.Errorf("invalid MQTT listener %q: %w", address, err)
		}
	}
	if m.DataDir == "" || m.MaxMessageBytes <= 0 || m.MaxMessageBytes > 10<<20 || m.IngestTimeout <= 0 {
		return fmt.Errorf("MQTT requires a data directory, positive ingest timeout and resource limit between 1 byte and 10 MiB")
	}
	if c.PubSub.BatchResourceEvents.Enabled && !c.PubSub.ResourceEvents.Enabled {
		return fmt.Errorf("batch Resource Events require Resource Events")
	}
	if m.PublicBrokers == "" {
		return fmt.Errorf("mqtt.public_brokers must advertise client connection addresses")
	}
	for _, address := range strings.Split(m.PublicBrokers, ",") {
		u, err := url.Parse(strings.TrimSpace(address))
		if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "mqtt" && u.Scheme != "mqtts" && u.Scheme != "ws" && u.Scheme != "wss") {
			return fmt.Errorf("invalid public MQTT address %q", address)
		}
	}
	return nil
}
