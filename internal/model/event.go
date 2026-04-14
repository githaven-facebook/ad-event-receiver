package model

import "time"

// EventType classifies an ad event as either SSP or DSP.
type EventType string

const (
	// EventTypeSSP represents a Supply-Side Platform event (publisher/inventory side).
	EventTypeSSP EventType = "SSP"
	// EventTypeDSP represents a Demand-Side Platform event (advertiser/buyer side).
	EventTypeDSP EventType = "DSP"
)

// Action represents the user interaction captured by the event.
type Action string

const (
	ActionImpression  Action = "impression"
	ActionClick       Action = "click"
	ActionConversion  Action = "conversion"
	ActionViewThrough Action = "view_through"
)

// AdEvent is the primary domain model representing a single ad system event.
type AdEvent struct {
	// EventID is a globally unique identifier for this event (UUID v4).
	EventID string `json:"event_id" validate:"required,uuid4"`

	// EventType classifies the event as SSP or DSP.
	EventType EventType `json:"event_type" validate:"required,oneof=SSP DSP"`

	// Timestamp is when the event occurred (RFC3339 format).
	Timestamp time.Time `json:"timestamp" validate:"required"`

	// Payload contains the ad-specific data for the event.
	Payload AdPayload `json:"payload" validate:"required"`
}

// AdPayload carries the detailed ad event attributes.
type AdPayload struct {
	// AdID identifies the specific ad creative being served.
	AdID string `json:"ad_id" validate:"required"`

	// CampaignID links the event to an advertising campaign.
	CampaignID string `json:"campaign_id" validate:"required"`

	// CreativeID identifies the specific creative asset (banner, video, etc).
	CreativeID string `json:"creative_id"`

	// PublisherID identifies the publisher/inventory source (SSP side).
	PublisherID string `json:"publisher_id"`

	// AdvertiserID identifies the advertiser placing the ad (DSP side).
	AdvertiserID string `json:"advertiser_id"`

	// UserID is the anonymized/hashed user identifier.
	UserID string `json:"user_id"`

	// Action describes what the user did (impression, click, conversion).
	Action Action `json:"action" validate:"required,oneof=impression click conversion view_through"`

	// Amount represents the monetary value associated with the event (e.g., bid price, CPC).
	Amount float64 `json:"amount,omitempty"`

	// Metadata holds arbitrary key-value pairs for extensibility.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BatchAdEvents wraps a slice of events for batch ingestion.
type BatchAdEvents struct {
	Events []AdEvent `json:"events" validate:"required,min=1,max=1000,dive"`
}

// IsSSP returns true if the event originated from a supply-side platform.
func (e *AdEvent) IsSSP() bool {
	return e.EventType == EventTypeSSP
}

// IsDSP returns true if the event originated from a demand-side platform.
func (e *AdEvent) IsDSP() bool {
	return e.EventType == EventTypeDSP
}
