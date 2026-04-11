package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nicedavid98/ad-event-receiver/internal/config"
	"github.com/nicedavid98/ad-event-receiver/internal/handler"
	"github.com/nicedavid98/ad-event-receiver/internal/kafka"
	"github.com/nicedavid98/ad-event-receiver/internal/metrics"
	"github.com/nicedavid98/ad-event-receiver/internal/server"
)

func main() {
	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		// Logger not yet available; write directly to stderr
		_, _ = os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize structured logger
	logger, err := buildLogger(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("configuration loaded",
		zap.Strings("kafka_brokers", cfg.Kafka.Brokers),
		zap.String("ssp_topic", cfg.Kafka.SSPTopic),
		zap.String("dsp_topic", cfg.Kafka.DSPTopic),
		zap.Int("server_port", cfg.Server.Port),
	)

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.BatchSize,
		cfg.Kafka.WriteTimeout,
		cfg.Kafka.RequiredAcks,
	)
	if err != nil {
		logger.Fatal("failed to create kafka producer", zap.Error(err))
	}
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Error("failed to close kafka producer", zap.Error(err))
		}
	}()

	// Register Prometheus metrics
	metrics.Register()

	// Build HTTP handlers
	eventHandler := handler.NewEventHandler(
		producer,
		cfg.Kafka.SSPTopic,
		cfg.Kafka.DSPTopic,
		logger,
	)
	healthHandler := handler.NewHealthHandler()

	// Build router
	router := server.NewRouter(server.RouterConfig{
		EventHandler:  eventHandler,
		HealthHandler: healthHandler,
		Logger:        logger,
	})

	// Wrap router with metrics middleware
	instrumentedRouter := metrics.WrapHandler(router)

	// Start metrics server in background
	metricsSrv := server.MetricsServer(cfg.Metrics.Port, promhttp.Handler(), logger)
	go func() {
		logger.Info("metrics server starting", zap.Int("port", cfg.Metrics.Port))
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	// Start main HTTP server in background
	srv := server.New(instrumentedRouter, cfg.Server, logger)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	// Wait for shutdown signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	case err := <-serverErr:
		logger.Error("server error", zap.Error(err))
	}

	// Graceful shutdown
	logger.Info("initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown error", zap.Error(err))
	}
	if err := srv.Shutdown(); err != nil {
		logger.Error("http server shutdown error", zap.Error(err))
	}

	logger.Info("shutdown complete")
}

func buildLogger(level, format string) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(lvl)

	if format == "console" {
		logCfg.Encoding = "console"
		logCfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	return logCfg.Build()
}
