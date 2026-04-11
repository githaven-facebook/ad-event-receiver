package model

import "time"

// SuccessResponse is returned when an event (or batch) is accepted successfully.
type SuccessResponse struct {
	// Status is always "ok" for success responses.
	Status string `json:"status"`

	// Message provides a human-readable description.
	Message string `json:"message,omitempty"`

	// EventID echoes back the event ID for single-event responses.
	EventID string `json:"event_id,omitempty"`

	// Accepted is the count of events accepted in a batch response.
	Accepted int `json:"accepted,omitempty"`

	// RequestID ties the response back to the original HTTP request.
	RequestID string `json:"request_id,omitempty"`

	// Timestamp records when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse is returned when request processing fails.
type ErrorResponse struct {
	// Status is always "error" for error responses.
	Status string `json:"status"`

	// Code is a machine-readable error code (e.g., "INVALID_PAYLOAD", "KAFKA_UNAVAILABLE").
	Code string `json:"code"`

	// Message provides a human-readable description of the error.
	Message string `json:"message"`

	// Details holds field-level validation errors when applicable.
	Details []FieldError `json:"details,omitempty"`

	// RequestID ties the response back to the original HTTP request.
	RequestID string `json:"request_id,omitempty"`

	// Timestamp records when the error occurred.
	Timestamp time.Time `json:"timestamp"`
}

// FieldError represents a validation error on a specific request field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NewSuccessResponse constructs a SuccessResponse with the current timestamp.
func NewSuccessResponse(message, eventID, requestID string) *SuccessResponse {
	return &SuccessResponse{
		Status:    "ok",
		Message:   message,
		EventID:   eventID,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// NewBatchSuccessResponse constructs a SuccessResponse for batch ingestion.
func NewBatchSuccessResponse(accepted int, requestID string) *SuccessResponse {
	return &SuccessResponse{
		Status:    "ok",
		Message:   "batch accepted",
		Accepted:  accepted,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorResponse constructs an ErrorResponse with the current timestamp.
func NewErrorResponse(code, message, requestID string, details ...FieldError) *ErrorResponse {
	return &ErrorResponse{
		Status:    "error",
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}
