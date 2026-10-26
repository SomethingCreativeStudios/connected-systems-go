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

	if len(args) == 2 && args[0] == "mqtt" && args[1] == "hash-password" {
		return mqtt.HashPassword(os.Stdin, os.Stdout)
	}
	migrateOnly := len(args) > 0 && args[0] == "migrate"
	if len(args) > 0 && !migrateOnly {
		return fmt.Errorf("unknown command %q (expected migrate or mqtt hash-password)", args[0])
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

	var mqttManager *mqtt.Manager
	if cfg.MQTT.Enabled {
		mqttManager = mqtt.NewManager(cfg.MQTT, logger)
		defer mqttManager.Disconnect()
	}
	pubSubPublisher := pubsub.NewPublisher(cfg.API.BaseURL, cfg.MQTT.NodeID, cfg.PubSub, mqttManager, logger)
	pubSubPublisher.SetMessageLimit(cfg.MQTT.MaxMessageBytes)
	defer func() {
		if mqttManager != nil {
			mqttManager.Quiesce()
		}
		pubSubPublisher.Close()
	}()
	var brokerErr <-chan error
	if mqttManager != nil {
		if err := mqttManager.Start(api.NewMQTTIngress(cfg, repos, pubSubPublisher, logger)); err != nil {
			return fmt.Errorf("start embedded MQTT: %w", err)
		}
		brokerErr = mqttManager.Fatal()
	}

	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      api.NewRouter(cfg, logger, repos, pubSubPublisher),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	defer server.Close()
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
	case err := <-brokerErr:
		return fmt.Errorf("embedded MQTT: %w", err)
	case <-ctx.Done():
	}

	logger.Info("Shutting down server...")
	if mqttManager != nil {
		mqttManager.Quiesce()
	}
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
