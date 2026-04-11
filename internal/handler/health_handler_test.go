package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeChecker implements ReadinessChecker for testing.
type fakeChecker struct {
	name string
	err  error
}

func (f *fakeChecker) Name() string { return f.name }
func (f *fakeChecker) Check() error { return f.err }

func TestLiveness(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.Liveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestReadiness(t *testing.T) {
	t.Run("all checks pass returns 200", func(t *testing.T) {
		h := NewHealthHandler(
			&fakeChecker{name: "kafka", err: nil},
			&fakeChecker{name: "db", err: nil},
		)
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		h.Readiness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp HealthStatus
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.Status != "ok" {
			t.Errorf("expected status ok, got %q", resp.Status)
		}
		if resp.Checks["kafka"] != "ok" {
			t.Errorf("expected kafka check ok, got %q", resp.Checks["kafka"])
		}
	})

	t.Run("any check fails returns 503", func(t *testing.T) {
		h := NewHealthHandler(
			&fakeChecker{name: "kafka", err: errors.New("connection refused")},
			&fakeChecker{name: "db", err: nil},
		)
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		h.Readiness(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", w.Code)
		}

		var resp HealthStatus
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp.Status != "degraded" {
			t.Errorf("expected status degraded, got %q", resp.Status)
		}
		if resp.Checks["kafka"] == "ok" {
			t.Error("expected kafka check to reflect error")
		}
		if resp.Checks["db"] != "ok" {
			t.Errorf("expected db check ok, got %q", resp.Checks["db"])
		}
	})

	t.Run("no checkers registered returns 200", func(t *testing.T) {
		h := NewHealthHandler()
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		h.Readiness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 with no checkers, got %d", w.Code)
		}
	})
}
