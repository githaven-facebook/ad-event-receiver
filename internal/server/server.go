package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/nicedavid98/ad-event-receiver/internal/config"
)

// Server wraps an http.Server and manages its lifecycle.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
	cfg        config.ServerConfig
}

// New creates a new Server with the given handler and configuration.
func New(handler http.Handler, cfg config.ServerConfig, logger *zap.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
		cfg:    cfg,
	}
}

// Start begins accepting connections. It blocks until the server encounters a
// non-ErrServerClosed error or is stopped via Shutdown.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server",
		zap.String("addr", s.httpServer.Addr),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

// Shutdown performs a graceful shutdown, waiting up to ShutdownTimeout for
// in-flight requests to complete before forcefully closing connections.
func (s *Server) Shutdown() error {
	s.logger.Info("shutting down HTTP server",
		zap.Duration("timeout", s.cfg.ShutdownTimeout),
	)

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("HTTP server stopped cleanly")
	return nil
}

// Addr returns the listen address of the server.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// MetricsServer creates a minimal HTTP server for the Prometheus metrics endpoint.
func MetricsServer(port int, metricsHandler http.Handler, logger *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Info("metrics server configured", zap.String("addr", srv.Addr))
	return srv
}
