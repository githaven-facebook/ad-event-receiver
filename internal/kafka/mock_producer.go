package kafka

import (
	"context"
	"sync"
)

// MockProducer is a thread-safe in-memory Producer implementation for testing.
type MockProducer struct {
	mu       sync.Mutex
	messages []MockMessage
	closed   bool

	// PublishErr, if non-nil, is returned by every Publish call.
	PublishErr error
	// CloseErr, if non-nil, is returned by Close.
	CloseErr error
}

// MockMessage records a single call to Publish.
type MockMessage struct {
	Topic string
	Key   string
	Value []byte
}

// Publish records the message and optionally returns a preconfigured error.
func (m *MockProducer) Publish(_ context.Context, topic, key string, value []byte) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, MockMessage{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	return nil
}

// Close marks the mock as closed and returns any preconfigured error.
func (m *MockProducer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.CloseErr
}

// Messages returns a copy of all messages published so far.
func (m *MockProducer) Messages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockMessage, len(m.messages))
	copy(result, m.messages)
	return result
}

// Reset clears the recorded messages.
func (m *MockProducer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
	m.closed = false
}

// IsClosed reports whether Close has been called.
func (m *MockProducer) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// MessagesForTopic returns all messages published to a given topic.
func (m *MockProducer) MessagesForTopic(topic string) []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []MockMessage
	for _, msg := range m.messages {
		if msg.Topic == topic {
			result = append(result, msg)
		}
	}
	return result
}
