package kafka

import (
	"context"
	"errors"
	"testing"
)

func TestMockProducer_Publish(t *testing.T) {
	ctx := context.Background()

	t.Run("publish single message", func(t *testing.T) {
		mock := &MockProducer{}
		err := mock.Publish(ctx, "test-topic", "key1", []byte(`{"event":"test"}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		msgs := mock.Messages()
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Topic != "test-topic" {
			t.Errorf("expected topic %q, got %q", "test-topic", msgs[0].Topic)
		}
		if msgs[0].Key != "key1" {
			t.Errorf("expected key %q, got %q", "key1", msgs[0].Key)
		}
	})

	t.Run("publish multiple messages", func(t *testing.T) {
		mock := &MockProducer{}
		topics := []string{"ssp-topic", "dsp-topic", "ssp-topic"}
		for i, topic := range topics {
			if err := mock.Publish(ctx, topic, "", []byte(`{}`)); err != nil {
				t.Fatalf("publish %d: unexpected error: %v", i, err)
			}
		}
		if len(mock.Messages()) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(mock.Messages()))
		}
		sspMsgs := mock.MessagesForTopic("ssp-topic")
		if len(sspMsgs) != 2 {
			t.Errorf("expected 2 SSP messages, got %d", len(sspMsgs))
		}
	})

	t.Run("publish returns configured error", func(t *testing.T) {
		sentinel := errors.New("kafka unavailable")
		mock := &MockProducer{PublishErr: sentinel}
		err := mock.Publish(ctx, "topic", "key", []byte("value"))
		if !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel error, got %v", err)
		}
		if len(mock.Messages()) != 0 {
			t.Error("no messages should be recorded when Publish errors")
		}
	})

	t.Run("reset clears messages", func(t *testing.T) {
		mock := &MockProducer{}
		_ = mock.Publish(ctx, "topic", "k", []byte("v"))
		mock.Reset()
		if len(mock.Messages()) != 0 {
			t.Error("expected messages to be cleared after Reset")
		}
	})
}

func TestMockProducer_Close(t *testing.T) {
	t.Run("close marks producer closed", func(t *testing.T) {
		mock := &MockProducer{}
		if mock.IsClosed() {
			t.Fatal("producer should not be closed initially")
		}
		if err := mock.Close(); err != nil {
			t.Fatalf("unexpected error on close: %v", err)
		}
		if !mock.IsClosed() {
			t.Error("producer should be marked closed after Close()")
		}
	})

	t.Run("close returns configured error", func(t *testing.T) {
		sentinel := errors.New("close error")
		mock := &MockProducer{CloseErr: sentinel}
		if err := mock.Close(); !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel error, got %v", err)
		}
	})
}

func TestNewProducer_Validation(t *testing.T) {
	t.Run("empty brokers returns error", func(t *testing.T) {
		_, err := NewProducer(nil, 100, 0, 1)
		if err == nil {
			t.Fatal("expected error for empty broker list")
		}
	})

	t.Run("valid config returns producer", func(t *testing.T) {
		p, err := NewProducer([]string{"localhost:9092"}, 50, 0, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Don't actually write to Kafka; just verify construction succeeds and Close works
		if err := p.Close(); err != nil {
			// Close may fail without a real broker; just ensure it doesn't panic
			t.Logf("close error (expected without broker): %v", err)
		}
	})
}
