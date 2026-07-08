// @title           OGC API - Connected Systems
// @version         1.0.0
// @description     OGC API Connected Systems implementation (Part 1 + Part 2)
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/yourusername/connected-systems-go/internal/api"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/mqtt"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Auto-migrate models
	if err := repository.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize MQTT manager (if enabled)
	var mqttManager *mqtt.Manager
	if cfg.MQTT.Enabled {
		mqttManager = mqtt.NewManager(mqtt.Config{
			Broker:   cfg.MQTT.Broker,
			ClientID: cfg.MQTT.ClientID,
			Username: cfg.MQTT.Username,
			Password: cfg.MQTT.Password,
			QoS:      cfg.MQTT.QoS,
			Retained: cfg.MQTT.Retained,
		}, logger)

		if err := mqttManager.Connect(); err != nil {
			logger.Fatal("Failed to connect to MQTT broker", zap.Error(err))
		}
		defer mqttManager.Disconnect()

		logger.Info("MQTT pub/sub enabled", zap.String("broker", cfg.MQTT.Broker))

		// Set up MQTT → DB ingestion handlers
		ingestion := mqtt.NewIngestionHandlers(logger, repos)

		// Subscribe to observations from external sources
		if err := mqttManager.Subscribe(mqtt.ObservationsWildcardTopic(), func(client paho.Client, msg paho.Message) {
			ingestion.HandleObservation(msg.Topic(), msg.Payload())
		}); err != nil {
			logger.Fatal("Failed to subscribe to observation topics", zap.Error(err))
		}

		// Subscribe to command status updates from external sources
		if err := mqttManager.Subscribe(mqtt.CommandStatusWildcardTopic(), func(client paho.Client, msg paho.Message) {
			ingestion.HandleCommandStatus(msg.Topic(), msg.Payload())
		}); err != nil {
			logger.Fatal("Failed to subscribe to command status topics", zap.Error(err))
		}
	}

	// Initialize API router
	router := api.NewRouter(cfg, logger, repos, mqttManager)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
