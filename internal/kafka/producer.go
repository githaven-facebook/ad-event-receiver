package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer defines the interface for publishing messages to Kafka.
// Using an interface allows easy mocking in tests.
type Producer interface {
	// Publish sends a message to the specified topic with the given key and value.
	Publish(ctx context.Context, topic, key string, value []byte) error

	// Close flushes any buffered messages and closes the underlying connection.
	Close() error
}

// producerConfig holds tuning parameters for the Kafka writer.
type producerConfig struct {
	brokers      []string
	batchSize    int
	writeTimeout time.Duration
	requiredAcks kafka.RequiredAcks
}

// kafkaProducer wraps kafka-go's Writer to implement the Producer interface.
type kafkaProducer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer connected to the given brokers.
// It uses async batching for high throughput with configurable write timeout.
func NewProducer(brokers []string, batchSize int, writeTimeout time.Duration, requiredAcks int) (Producer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("at least one broker address is required")
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if writeTimeout <= 0 {
		writeTimeout = 10 * time.Second
	}

	acks := kafka.RequireOne
	switch requiredAcks {
	case 0:
		acks = kafka.RequireNone
	case -1:
		acks = kafka.RequireAll
	default:
		acks = kafka.RequireOne
	}

	cfg := producerConfig{
		brokers:      brokers,
		batchSize:    batchSize,
		writeTimeout: writeTimeout,
		requiredAcks: acks,
	}

	return newKafkaProducer(cfg), nil
}

func newKafkaProducer(cfg producerConfig) *kafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.batchSize,
		WriteTimeout: cfg.writeTimeout,
		RequiredAcks: cfg.requiredAcks,
		// Async = false ensures we get delivery confirmation before returning
		Async:        false,
		// Compress messages to reduce network overhead
		Compression:  kafka.Snappy,
	}

	return &kafkaProducer{writer: w}
}

// Publish sends a single message to the given Kafka topic.
// The key is used for partition routing (same key => same partition).
func (p *kafkaProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Value: value,
	}
	if key != "" {
		msg.Key = []byte(key)
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish to topic %q: %w", topic, err)
	}
	return nil
}

// Close flushes pending writes and closes the Kafka writer.
func (p *kafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("kafka producer close: %w", err)
	}
	return nil
}
