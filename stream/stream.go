// Package stream provides definition and implementations of a Stream
// for publishing and subscribing to events using a distributed and persistent
// streaming message processing backend (e.g. Apache Kafka).
package stream

import (
	"time"
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

// ReadOnlyStream wraps functionality for
// subscribing to event(s) published to a stream
type ReadOnlyStream interface {
	Subscribe(close chan struct{}) (<-chan Event, error)
}

// WriteOnlyStream wraps functionality for
// publishing event(s) to a stream
type WriteOnlyStream interface {
	Publish(events []Event) ([]Event, error)
}

// Stream wraps functionality for publishing and subscribing to
// event(s) sent to a streaming message processing backend
type Stream interface {
	ReadOnlyStream
	WriteOnlyStream
}
