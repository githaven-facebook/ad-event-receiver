//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	appKafka "github.com/nicedavid98/ad-event-receiver/internal/kafka"
	"github.com/nicedavid98/ad-event-receiver/internal/handler"
	"github.com/nicedavid98/ad-event-receiver/internal/model"
	"github.com/nicedavid98/ad-event-receiver/internal/server"
)

const (
	testKafkaBroker = "localhost:29092" // exposed by docker-compose
	testSSPTopic    = "test-ad-events-ssp"
	testDSPTopic    = "test-ad-events-dsp"
)

// TestEventFlow_SSP verifies that an SSP event posted to the HTTP endpoint
// is routed to the SSP Kafka topic and can be consumed.
func TestEventFlow_SSP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	producer, err := appKafka.NewProducer([]string{testKafkaBroker}, 1, 5*time.Second, 1)
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	logger := zap.NewNop()
	eventHandler := handler.NewEventHandler(producer, testSSPTopic, testDSPTopic, logger)
	healthHandler := handler.NewHealthHandler()

	router := server.NewRouter(server.RouterConfig{
		EventHandler:  eventHandler,
		HealthHandler: healthHandler,
		Logger:        logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	event := model.AdEvent{
		EventID:   fmt.Sprintf("integration-ssp-%d", time.Now().UnixNano()),
		EventType: model.EventTypeSSP,
		Timestamp: time.Now().UTC(),
		Payload: model.AdPayload{
			AdID:        "integration-ad-001",
			CampaignID:  "integration-campaign-001",
			PublisherID: "integration-pub-001",
			Action:      model.ActionImpression,
			Amount:      0.10,
			Metadata: map[string]string{
				"placement": "top_banner",
				"format":    "300x250",
			},
		},
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(
		ts.URL+"/api/v1/events",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	// Consume from Kafka and verify the message arrived
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{testKafkaBroker},
		Topic:     testSSPTopic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
		MaxWait:   2 * time.Second,
	})
	defer reader.Close()

	// Seek to the latest offset just before we published
	if err := reader.SetOffsetAt(ctx, time.Now().Add(-5*time.Second)); err != nil {
		t.Logf("set offset: %v (may be empty topic)", err)
	}

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read from kafka: %v", err)
	}

	var consumed model.AdEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("unmarshal kafka message: %v", err)
	}
	if consumed.EventID != event.EventID {
		t.Errorf("event_id mismatch: want %q, got %q", event.EventID, consumed.EventID)
	}
	if consumed.EventType != model.EventTypeSSP {
		t.Errorf("event_type mismatch: want SSP, got %q", consumed.EventType)
	}
}

// TestEventFlow_DSP verifies end-to-end routing for a DSP event.
func TestEventFlow_DSP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	producer, err := appKafka.NewProducer([]string{testKafkaBroker}, 1, 5*time.Second, 1)
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	logger := zap.NewNop()
	eventHandler := handler.NewEventHandler(producer, testSSPTopic, testDSPTopic, logger)
	healthHandler := handler.NewHealthHandler()

	router := server.NewRouter(server.RouterConfig{
		EventHandler:  eventHandler,
		HealthHandler: healthHandler,
		Logger:        logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	event := model.AdEvent{
		EventID:   fmt.Sprintf("integration-dsp-%d", time.Now().UnixNano()),
		EventType: model.EventTypeDSP,
		Timestamp: time.Now().UTC(),
		Payload: model.AdPayload{
			AdID:         "integration-ad-002",
			CampaignID:   "integration-campaign-002",
			AdvertiserID: "integration-adv-001",
			Action:       model.ActionClick,
			Amount:       2.50,
		},
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(
		ts.URL+"/api/v1/events",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{testKafkaBroker},
		Topic:    testDSPTopic,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	defer reader.Close()

	if err := reader.SetOffsetAt(ctx, time.Now().Add(-5*time.Second)); err != nil {
		t.Logf("set offset: %v", err)
	}

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read from kafka: %v", err)
	}

	var consumed model.AdEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if consumed.EventID != event.EventID {
		t.Errorf("event_id mismatch: want %q, got %q", event.EventID, consumed.EventID)
	}
}

// TestBatchEventFlow verifies batch endpoint routing.
func TestBatchEventFlow(t *testing.T) {
	producer, err := appKafka.NewProducer([]string{testKafkaBroker}, 10, 5*time.Second, 1)
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	logger := zap.NewNop()
	eventHandler := handler.NewEventHandler(producer, testSSPTopic, testDSPTopic, logger)
	healthHandler := handler.NewHealthHandler()

	router := server.NewRouter(server.RouterConfig{
		EventHandler:  eventHandler,
		HealthHandler: healthHandler,
		Logger:        logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	batch := model.BatchAdEvents{
		Events: []model.AdEvent{
			{
				EventID:   fmt.Sprintf("batch-ssp-%d", time.Now().UnixNano()),
				EventType: model.EventTypeSSP,
				Timestamp: time.Now().UTC(),
				Payload:   model.AdPayload{AdID: "ad-b1", CampaignID: "c-b1", Action: model.ActionImpression},
			},
			{
				EventID:   fmt.Sprintf("batch-dsp-%d", time.Now().UnixNano()),
				EventType: model.EventTypeDSP,
				Timestamp: time.Now().UTC(),
				Payload:   model.AdPayload{AdID: "ad-b2", CampaignID: "c-b2", Action: model.ActionClick},
			},
		},
	}

	body, _ := json.Marshal(batch)
	resp, err := http.Post(
		ts.URL+"/api/v1/events/batch",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/events/batch: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var result model.SuccessResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", result.Accepted)
	}
}
