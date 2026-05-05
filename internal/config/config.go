package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	API      APIConfig      `mapstructure:"api"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
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
}

// APIConfig holds API-specific configuration
type APIConfig struct {
	BaseURL      string `mapstructure:"base_url"`
	Title        string `mapstructure:"title"`
	Description  string `mapstructure:"description"`
	Version      string `mapstructure:"version"`
	DefaultLimit int    `mapstructure:"default_limit"`
}

// MQTTConfig holds MQTT broker connection configuration.
type MQTTConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Broker   string `mapstructure:"broker"`
	ClientID string `mapstructure:"client_id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	QoS      byte   `mapstructure:"qos"`
	Retained bool   `mapstructure:"retained"`
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
	viper.SetDefault("api.title", "OGC Connected Systems API")
	viper.SetDefault("api.version", "1.0.0")
	viper.SetDefault("api.description", "OGC API - Connected Systems - Part 1: Feature Resources")
	viper.SetDefault("api.default_limit", 10)
	viper.SetDefault("mqtt.enabled", false)
	viper.SetDefault("mqtt.broker", "tcp://localhost:1883")
	viper.SetDefault("mqtt.client_id", "cs-api-server")
	viper.SetDefault("mqtt.qos", 1)
	viper.SetDefault("mqtt.retained", false)

	// Read from environment — replace "." with "_" so database.host → DATABASE_HOST
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
