package transport

import "context"

// Event represents a message/event in the system
type Event struct {
	Topic   string
	Payload []byte
	Headers map[string]string
}

// Publisher defines the interface for publishing events
type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

// Subscriber defines the interface for subscribing to events
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}

// Handler is a function that processes incoming events
type Handler func(ctx context.Context, event Event) error

// EventBus combines both publishing and subscribing capabilities
type EventBus interface {
	Publisher
	Subscriber
}
