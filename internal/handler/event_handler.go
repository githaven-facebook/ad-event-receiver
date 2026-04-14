package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/nicedavid98/ad-event-receiver/internal/kafka"
	"github.com/nicedavid98/ad-event-receiver/internal/model"
)

const (
	maxBodyBytes = 1 << 20 // 1 MB
)

// EventHandler handles ad event ingestion requests.
type EventHandler struct {
	producer kafka.Producer
	sspTopic string
	dspTopic string
	logger   *zap.Logger
}

// NewEventHandler creates a new EventHandler with the given dependencies.
func NewEventHandler(producer kafka.Producer, sspTopic, dspTopic string, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		producer: producer,
		sspTopic: sspTopic,
		dspTopic: dspTopic,
		logger:   logger,
	}
}

// HandleEvent handles POST /api/v1/events for a single ad event.
func (h *EventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetReqID(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var event model.AdEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.logger.Warn("failed to decode event payload",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		writeJSON(w, http.StatusBadRequest, model.NewErrorResponse(
			"INVALID_PAYLOAD",
			"request body is not valid JSON or exceeds size limit",
			requestID,
		))
		return
	}

	if errs := validateEvent(&event); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, model.NewErrorResponse(
			"VALIDATION_FAILED",
			"one or more fields failed validation",
			requestID,
			errs...,
		))
		return
	}

	topic := h.topicForEvent(&event)

	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal event for kafka",
			zap.String("request_id", requestID),
			zap.String("event_id", event.EventID),
			zap.Error(err),
		)
		writeJSON(w, http.StatusInternalServerError, model.NewErrorResponse(
			"INTERNAL_ERROR",
			"failed to process event",
			requestID,
		))
		return
	}

	if err := h.producer.Publish(r.Context(), topic, event.EventID, payload); err != nil {
		h.logger.Error("failed to publish event to kafka",
			zap.String("request_id", requestID),
			zap.String("event_id", event.EventID),
			zap.String("topic", topic),
			zap.Error(err),
		)
		writeJSON(w, http.StatusServiceUnavailable, model.NewErrorResponse(
			"KAFKA_UNAVAILABLE",
			"failed to publish event, please retry",
			requestID,
		))
		return
	}

	h.logger.Info("event published",
		zap.String("request_id", requestID),
		zap.String("event_id", event.EventID),
		zap.String("event_type", string(event.EventType)),
		zap.String("topic", topic),
	)

	writeJSON(w, http.StatusAccepted, model.NewSuccessResponse("event accepted", event.EventID, requestID))
}

// HandleBatchEvents handles POST /api/v1/events/batch for multiple events.
func (h *EventHandler) HandleBatchEvents(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetReqID(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes*10)

	var batch model.BatchAdEvents
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.logger.Warn("failed to decode batch payload",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		writeJSON(w, http.StatusBadRequest, model.NewErrorResponse(
			"INVALID_PAYLOAD",
			"request body is not valid JSON or exceeds size limit",
			requestID,
		))
		return
	}

	if len(batch.Events) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, model.NewErrorResponse(
			"VALIDATION_FAILED",
			"batch must contain at least one event",
			requestID,
		))
		return
	}

	if len(batch.Events) > 1000 {
		writeJSON(w, http.StatusRequestEntityTooLarge, model.NewErrorResponse(
			"BATCH_TOO_LARGE",
			"batch may not exceed 1000 events",
			requestID,
		))
		return
	}

	accepted := 0
	for i := range batch.Events {
		event := &batch.Events[i]

		if errs := validateEvent(event); len(errs) > 0 {
			h.logger.Warn("skipping invalid event in batch",
				zap.String("request_id", requestID),
				zap.Int("index", i),
				zap.String("event_id", event.EventID),
			)
			continue
		}

		topic := h.topicForEvent(event)
		payload, err := json.Marshal(event)
		if err != nil {
			h.logger.Error("failed to marshal batch event",
				zap.String("request_id", requestID),
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		if err := h.producer.Publish(r.Context(), topic, event.EventID, payload); err != nil {
			h.logger.Error("failed to publish batch event",
				zap.String("request_id", requestID),
				zap.Int("index", i),
				zap.String("event_id", event.EventID),
				zap.String("topic", topic),
				zap.Error(err),
			)
			continue
		}
		accepted++
	}

	h.logger.Info("batch processed",
		zap.String("request_id", requestID),
		zap.Int("total", len(batch.Events)),
		zap.Int("accepted", accepted),
	)

	writeJSON(w, http.StatusAccepted, model.NewBatchSuccessResponse(accepted, requestID))
}

func (h *EventHandler) topicForEvent(event *model.AdEvent) string {
	if event.IsSSP() {
		return h.sspTopic
	}
	return h.dspTopic
}

func validateEvent(event *model.AdEvent) []model.FieldError {
	var errs []model.FieldError

	if event.EventID == "" {
		errs = append(errs, model.FieldError{Field: "event_id", Message: "required"})
	}
	if event.EventType != model.EventTypeSSP && event.EventType != model.EventTypeDSP {
		errs = append(errs, model.FieldError{Field: "event_type", Message: "must be SSP or DSP"})
	}
	if event.Timestamp.IsZero() {
		errs = append(errs, model.FieldError{Field: "timestamp", Message: "required"})
	}
	if event.Payload.AdID == "" {
		errs = append(errs, model.FieldError{Field: "payload.ad_id", Message: "required"})
	}
	if event.Payload.CampaignID == "" {
		errs = append(errs, model.FieldError{Field: "payload.campaign_id", Message: "required"})
	}
	validActions := map[model.Action]bool{
		model.ActionImpression: true,
		model.ActionClick:      true,
		model.ActionConversion: true,
		model.ActionViewThrough: true,
	}
	if !validActions[event.Payload.Action] {
		errs = append(errs, model.FieldError{
			Field:   "payload.action",
			Message: "must be one of: impression, click, conversion, view_through",
		})
	}

	return errs
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent, nothing we can do
		return
	}
}
