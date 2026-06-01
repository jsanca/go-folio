// Package sse implements a fan-out SSE broker for real-time stock events.
package sse

import (
	"sync"
	"time"
)

// StockEvent is the payload pushed to SSE subscribers when stock changes.
// EventType values: "stock.updated", "stock.low", "stock.out", "stock.restocked".
type StockEvent struct {
	EventType  string    `json:"eventType"`
	SKU        string    `json:"sku"`
	Available  int32     `json:"available"`
	Reserved   int32     `json:"reserved"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurredAt"`
}

// Broker distributes StockEvents to all active SSE subscribers.
type Broker struct {
	clients map[chan StockEvent]struct{}
	mu      sync.RWMutex
}

// NewBroker creates an empty Broker ready for use.
func NewBroker() *Broker {
	return &Broker{clients: make(map[chan StockEvent]struct{})}
}

// Subscribe registers a new subscriber and returns its event channel.
// The channel has a buffer of 10 to absorb short bursts without blocking Publish.
func (b *Broker) Subscribe() chan StockEvent {
	ch := make(chan StockEvent, 10)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the broker and closes it.
func (b *Broker) Unsubscribe(ch chan StockEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// Publish sends event to every subscriber. Slow or full channels are skipped
// to avoid blocking the caller.
func (b *Broker) Publish(event StockEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}
}
