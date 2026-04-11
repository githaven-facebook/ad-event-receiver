package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "ad_event_receiver"
)

var (
	// EventsReceivedTotal counts all ad events received over HTTP, labeled by event type (SSP/DSP).
	EventsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_received_total",
			Help:      "Total number of ad events received via HTTP, partitioned by event type.",
		},
		[]string{"event_type"},
	)

	// EventsPublishedTotal counts events successfully published to Kafka, labeled by topic.
	EventsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_published_total",
			Help:      "Total number of ad events successfully published to Kafka, partitioned by topic.",
		},
		[]string{"topic"},
	)

	// EventProcessingDuration records the end-to-end processing latency per event type.
	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "event_processing_duration_seconds",
			Help:      "End-to-end processing latency in seconds for ad events.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	// KafkaPublishErrorsTotal counts Kafka publish failures by topic.
	KafkaPublishErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "kafka_publish_errors_total",
			Help:      "Total number of errors encountered when publishing to Kafka, partitioned by topic.",
		},
		[]string{"topic"},
	)
)

// Register initializes all metrics with their zero values to ensure they appear
// in /metrics output even before any events are processed.
func Register() {
	// Pre-initialize label combinations so dashboards always see the series.
	for _, eventType := range []string{"SSP", "DSP"} {
		EventsReceivedTotal.WithLabelValues(eventType)
		EventProcessingDuration.WithLabelValues(eventType)
	}
}
