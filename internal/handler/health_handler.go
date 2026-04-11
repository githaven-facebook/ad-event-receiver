package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthStatus represents the overall service health.
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// ReadinessChecker is implemented by components that can report their readiness.
type ReadinessChecker interface {
	// Name returns a human-readable identifier for this check.
	Name() string
	// Check returns nil if the component is ready, or an error describing why it is not.
	Check() error
}

// HealthHandler serves liveness and readiness probe endpoints.
type HealthHandler struct {
	checkers []ReadinessChecker
}

// NewHealthHandler creates a HealthHandler with the given readiness checkers.
func NewHealthHandler(checkers ...ReadinessChecker) *HealthHandler {
	return &HealthHandler{checkers: checkers}
}

// Liveness handles GET /healthz.
// Returns 200 as long as the process is running; used by the container runtime
// to decide whether to restart the pod.
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}

// Readiness handles GET /readyz.
// Runs all registered readiness checks and returns 200 only when all pass.
// Used by Kubernetes to decide whether to send traffic to this pod.
func (h *HealthHandler) Readiness(w http.ResponseWriter, _ *http.Request) {
	checks := make(map[string]string, len(h.checkers))
	allHealthy := true

	for _, checker := range h.checkers {
		if err := checker.Check(); err != nil {
			checks[checker.Name()] = err.Error()
			allHealthy = false
		} else {
			checks[checker.Name()] = "ok"
		}
	}

	status := HealthStatus{
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		status.Status = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		status.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(status)
}
