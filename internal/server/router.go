package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/nicedavid98/ad-event-receiver/internal/handler"
	customMiddleware "github.com/nicedavid98/ad-event-receiver/internal/handler"
)

// RouterConfig holds dependencies needed to build the router.
type RouterConfig struct {
	EventHandler   *handler.EventHandler
	HealthHandler  *handler.HealthHandler
	Logger         *zap.Logger
	AllowedOrigins []string
}

// NewRouter constructs and returns a chi.Router with all routes and middleware configured.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Built-in chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)

	// Custom middleware
	r.Use(customMiddleware.RequestLogger(cfg.Logger))
	r.Use(customMiddleware.Recoverer(cfg.Logger))
	r.Use(customMiddleware.CORS(cfg.AllowedOrigins))

	// Health probes - no auth or content-type enforcement needed
	r.Get("/healthz", cfg.HealthHandler.Liveness)
	r.Get("/readyz", cfg.HealthHandler.Readiness)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(customMiddleware.ContentTypeJSON)

		r.Route("/events", func(r chi.Router) {
			r.Post("/", cfg.EventHandler.HandleEvent)
			r.Post("/batch", cfg.EventHandler.HandleBatchEvents)
		})
	})

	// 404 handler
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","code":"NOT_FOUND","message":"the requested resource does not exist"}`))
	})

	// 405 handler
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"status":"error","code":"METHOD_NOT_ALLOWED","message":"method not allowed for this endpoint"}`))
	})

	return r
}
