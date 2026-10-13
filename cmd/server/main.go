// @title           OGC API - Connected Systems
// @version         1.0.0
// @description     OGC API Connected Systems implementation (Part 1 + Part 2)
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/api"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mqtt"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	paho "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Usage:
//
//	server          start the API server (applying migrations when database.auto_migrate is set)
//	server migrate  apply pending database migrations and exit
func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	err = run(logger, os.Args[1:])
	if err != nil {
		logger.Error("Server stopped", zap.Error(err))
	}
	_ = logger.Sync()
	if err != nil {
		os.Exit(1)
	}
}

// run returns instead of exiting so deferred cleanup (MQTT disconnect,
// publisher flush, database close) always happens.
func run(logger *zap.Logger, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	migrateOnly := len(args) > 0 && args[0] == "migrate"
	if len(args) > 0 && !migrateOnly {
		return fmt.Errorf("unknown command %q (expected migrate)", args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	db, err := openDatabase(cfg.Database)
	if err != nil {
		return err
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	if migrateOnly || cfg.Database.AutoMigrate {
		if err := repository.Migrate(ctx, db); err != nil {
			return err
		}
		logger.Info("Database migrated")
	}
	if migrateOnly {
		return nil
	}

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
			return fmt.Errorf("connect to MQTT broker: %w", err)
		}
		defer mqttManager.Disconnect()

		logger.Info("MQTT pub/sub enabled", zap.String("broker", cfg.MQTT.Broker))
	}

	pubSubPublisher := pubsub.NewPublisher(cfg.API.BaseURL, cfg.PubSub, mqttManager, logger)
	defer pubSubPublisher.Close()
	if mqttManager != nil && cfg.PubSub.ResourceData.Enabled {
		// Set up strict Resource Data Message ingestion.
		ingestion := mqtt.NewIngestionHandlers(logger, repos, pubSubPublisher)

		if err := mqttManager.Subscribe(mqtt.ObservationsWildcardTopic(), func(client paho.Client, msg paho.Message) {
			ingestion.HandleObservation(msg.Topic(), msg.Payload())
		}); err != nil {
			return fmt.Errorf("subscribe to observation topics: %w", err)
		}

		if err := mqttManager.Subscribe(mqtt.CommandStatusWildcardTopic(), func(client paho.Client, msg paho.Message) {
			ingestion.HandleCommandStatus(msg.Topic(), msg.Payload())
		}); err != nil {
			return fmt.Errorf("subscribe to command status topics: %w", err)
		}
	}

	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      api.NewRouter(cfg, logger, repos, pubSubPublisher),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("Starting server",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down server: %w", err)
	}
	logger.Info("Server exited")
	return nil
}

func openDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	return db, nil
}
