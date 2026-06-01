package sse_test

import (
	"testing"
	"time"

	"github.com/jsanca/go-folio/gateway-service/internal/sse"
)

func TestBroker_PublishDelivers(t *testing.T) {
	b := sse.NewBroker()
	ch1 := b.Subscribe()
	ch2 := b.Subscribe()

	now := time.Now().UTC().Truncate(time.Second)
	event := sse.StockEvent{EventType: "stock.updated", SKU: "BAG-001", Available: 10, Reserved: 0, Status: "IN_STOCK", OccurredAt: now}
	b.Publish(event)

	for _, ch := range []chan sse.StockEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got != event {
				t.Errorf("want %+v, got %+v", event, got)
			}
		default:
			t.Error("expected event in channel, got none")
		}
	}

	b.Unsubscribe(ch1)
	b.Unsubscribe(ch2)
}

func TestBroker_PublishNonBlocking(t *testing.T) {
	b := sse.NewBroker()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	event := sse.StockEvent{EventType: "stock.updated", SKU: "BAG-001", Status: "IN_STOCK", OccurredAt: time.Now().UTC()}
	// Fill the buffer completely (capacity 10).
	for range 10 {
		b.Publish(event)
	}

	// This 11th publish must return immediately even though the channel is full.
	done := make(chan struct{})
	go func() {
		b.Publish(event)
		close(done)
	}()

	select {
	case <-done:
		// expected — publish returned without blocking
	case <-time.After(time.Second):
		t.Error("Publish blocked on a full channel")
	}
}

func TestBroker_Unsubscribe(t *testing.T) {
	b := sse.NewBroker()
	ch := b.Subscribe()
	b.Unsubscribe(ch)

	// Channel must be closed after unsubscribe.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected closed channel, got value")
		}
	default:
		t.Error("channel not closed after Unsubscribe")
	}

	// Publish must not reach an unsubscribed channel (and must not panic).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Publish panicked after Unsubscribe: %v", r)
		}
	}()
	b.Publish(sse.StockEvent{EventType: "stock.out", SKU: "BAG-001", OccurredAt: time.Now().UTC()})
}
