// Package stream provides definition and implementations of a Stream
// for publishing and subscribing to events using a distributed and persistent
// streaming message processing backend (e.g. Apache Kafka).
package stream

import (
	"time"

	"github.com/tozny/utils-go/auth"
)

// Event wraps information and metadata about an event published to a stream
type Event struct {
	Topic     string    // The Stream topic this event was published to
	Tag       string    // Publisher defined value associated with this event
	Message   string    // Publisher provided content for the Event
	Timestamp time.Time // The timestamp for when the event was first published to the stream
	Partition string    // The server side resource this event is stored or has been subscribed from
	SortKey   string    // Server defined unique and monotonic key for ordering of published events
}

// CloudEvent wraps information and metadata about a cloud event published to a stream
type CloudEvent struct {
	Topic       string      // The Stream topic this event was published to
	Tag         string      // Publisher defined value associated with this event
	Type        string      // Event type
	Source      string      // Source from where the event was triggered
	ContentType string      // ContentType of Data (Eg: application/json)
	Data        interface{} // Publisher provided content for the Event
	Timestamp   time.Time   // The timestamp for when the event was first published to the stream
	Partition   string      // The server side resource this event is stored or has been subscribed from
	SortKey     string      // Server defined unique and monotonic key for ordering of published events
}

// ReadOnlyStream wraps functionality for
// subscribing to event(s) published to a stream
type ReadOnlyStream interface {
	Subscribe(close chan struct{}) (<-chan Event, error)
	Receive(close chan struct{}) (<-chan CloudEvent, error)
}

// WriteOnlyStream wraps functionality for
// publishing event(s) to a stream
type WriteOnlyStream interface {
	Publish(events []Event) ([]Event, error)
	Send(event CloudEvent) error
}

// Stream wraps functionality for publishing and subscribing to
// event(s) sent to a streaming message processing backend
type Stream interface {
	ReadOnlyStream
	WriteOnlyStream
}

// EventPublisher wraps functionality for publishing tagged event data
// TODO moq mocking!
type EventPublisher interface {
	// Publish publishes an event in the topic with a particular tag & string message
	Publish(tag string, message string) error
	// PublishData is responsible for converting arbitrary data to a string & publishing it as an event
	PublishData(tag string, data auth.Claims) error
}

// NoOpEventClient is an EventPublisher that ignores all events and doesn't publish them
type NoOpEventClient struct{}

func (c *NoOpEventClient) Publish(tag string, message string) error {
	// do nothing! event gets dropped.
	return nil
}

func (c *NoOpEventClient) PublishData(tag string, data auth.Claims) error {
	// do nothing! event gets dropped.
	return nil
}
