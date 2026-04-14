package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/nicedavid98/ad-event-receiver/internal/kafka"
	"github.com/nicedavid98/ad-event-receiver/internal/model"
)

func newTestEventHandler(producer kafka.Producer) *EventHandler {
	return NewEventHandler(producer, "ad-events-ssp", "ad-events-dsp", zap.NewNop())
}

func validSSPEvent() model.AdEvent {
	return model.AdEvent{
		EventID:   "550e8400-e29b-41d4-a716-446655440000",
		EventType: model.EventTypeSSP,
		Timestamp: time.Now().UTC(),
		Payload: model.AdPayload{
			AdID:        "ad-001",
			CampaignID:  "campaign-001",
			PublisherID: "pub-001",
			Action:      model.ActionImpression,
			Amount:      0.05,
		},
	}
}

func validDSPEvent() model.AdEvent {
	return model.AdEvent{
		EventID:   "660e8400-e29b-41d4-a716-446655440001",
		EventType: model.EventTypeDSP,
		Timestamp: time.Now().UTC(),
		Payload: model.AdPayload{
			AdID:         "ad-002",
			CampaignID:   "campaign-002",
			AdvertiserID: "adv-001",
			Action:       model.ActionClick,
			Amount:       1.25,
		},
	}
}

func postEvent(t *testing.T, handler http.HandlerFunc, event interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestHandleEvent_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		event        interface{}
		publishErr   error
		wantStatus   int
		wantTopic    string
		wantAccepted bool
	}{
		{
			name:         "valid SSP event",
			event:        validSSPEvent(),
			wantStatus:   http.StatusAccepted,
			wantTopic:    "ad-events-ssp",
			wantAccepted: true,
		},
		{
			name:         "valid DSP event",
			event:        validDSPEvent(),
			wantStatus:   http.StatusAccepted,
			wantTopic:    "ad-events-dsp",
			wantAccepted: true,
		},
		{
			name: "missing event_id",
			event: model.AdEvent{
				EventType: model.EventTypeSSP,
				Timestamp: time.Now().UTC(),
				Payload: model.AdPayload{
					AdID: "ad-001", CampaignID: "c-001",
					Action: model.ActionImpression,
				},
			},
			wantStatus:   http.StatusUnprocessableEntity,
			wantAccepted: false,
		},
		{
			name: "invalid event_type",
			event: map[string]interface{}{
				"event_id":   "550e8400-e29b-41d4-a716-446655440000",
				"event_type": "UNKNOWN",
				"timestamp":  time.Now().UTC(),
				"payload": map[string]interface{}{
					"ad_id": "ad-001", "campaign_id": "c-001", "action": "impression",
				},
			},
			wantStatus:   http.StatusUnprocessableEntity,
			wantAccepted: false,
		},
		{
			name: "invalid action",
			event: model.AdEvent{
				EventID:   "550e8400-e29b-41d4-a716-446655440000",
				EventType: model.EventTypeSSP,
				Timestamp: time.Now().UTC(),
				Payload: model.AdPayload{
					AdID: "ad-001", CampaignID: "c-001",
					Action: model.Action("unknown"),
				},
			},
			wantStatus:   http.StatusUnprocessableEntity,
			wantAccepted: false,
		},
		{
			name:         "kafka publish error",
			event:        validSSPEvent(),
			publishErr:   context.DeadlineExceeded,
			wantStatus:   http.StatusServiceUnavailable,
			wantAccepted: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &kafka.MockProducer{PublishErr: tc.publishErr}
			h := newTestEventHandler(mock)

			w := postEvent(t, h.HandleEvent, tc.event)

			if w.Code != tc.wantStatus {
				t.Errorf("status: want %d, got %d (body: %s)", tc.wantStatus, w.Code, w.Body.String())
			}

			if tc.wantAccepted {
				msgs := mock.Messages()
				if len(msgs) != 1 {
					t.Fatalf("expected 1 published message, got %d", len(msgs))
				}
				if msgs[0].Topic != tc.wantTopic {
					t.Errorf("topic: want %q, got %q", tc.wantTopic, msgs[0].Topic)
				}
			}
		})
	}
}

func TestHandleEvent_MalformedJSON(t *testing.T) {
	mock := &kafka.MockProducer{}
	h := newTestEventHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
	if len(mock.Messages()) != 0 {
		t.Error("no messages should be published for malformed JSON")
	}
}

func TestHandleBatchEvents(t *testing.T) {
	t.Run("valid batch with SSP and DSP events", func(t *testing.T) {
		mock := &kafka.MockProducer{}
		h := newTestEventHandler(mock)

		batch := model.BatchAdEvents{
			Events: []model.AdEvent{validSSPEvent(), validDSPEvent()},
		}
		body, _ := json.Marshal(batch)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/events/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.HandleBatchEvents(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d (body: %s)", w.Code, w.Body.String())
		}
		if len(mock.Messages()) != 2 {
			t.Errorf("expected 2 published messages, got %d", len(mock.Messages()))
		}
		var resp model.SuccessResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Accepted != 2 {
			t.Errorf("expected accepted=2, got %d", resp.Accepted)
		}
	})

	t.Run("empty batch returns 422", func(t *testing.T) {
		mock := &kafka.MockProducer{}
		h := newTestEventHandler(mock)

		batch := model.BatchAdEvents{Events: []model.AdEvent{}}
		body, _ := json.Marshal(batch)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/events/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.HandleBatchEvents(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", w.Code)
		}
	})

	t.Run("batch skips invalid events", func(t *testing.T) {
		mock := &kafka.MockProducer{}
		h := newTestEventHandler(mock)

		valid := validSSPEvent()
		invalid := model.AdEvent{} // missing required fields

		batch := model.BatchAdEvents{Events: []model.AdEvent{valid, invalid}}
		body, _ := json.Marshal(batch)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/events/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.HandleBatchEvents(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d", w.Code)
		}
		if len(mock.Messages()) != 1 {
			t.Errorf("expected 1 valid message published, got %d", len(mock.Messages()))
		}
	})
}
