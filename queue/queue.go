// Package queue provides definition and implementations of the Queue interface
// for passing messages between services using distributed persistence storage backends
// (e.g. AWS SQS).
package queue

// Message wraps data and metadata for a queue message
type Message struct {
	Body         string            // JSON encoded message content
	ReceiptID    string            // Unique identifier associated with the dequeuing of this message
	ReceiveCount int               // The approximate number of times this message has been dequeued
	Tags         map[string]string // Map of user defined key value pairs associated with this message
}

// Queue is the interface which wraps methods for
// adding and removing message(s), and permanently deleting a message
// from a queue data structure.
type Queue interface {
	DeleteMessage(receiptID string) error
	EnqueueMessage(message Message) error
	DequeueMessage() (Message, error)
	// BatchEnqueueMessages enqueues a batch of messages to the queue
	// returning a list of messages that failed to enqueue and error (if any).
	// Error must always be not nil if any messages failed to enqueue
	BatchEnqueueMessages(messages []Message) ([]Message, error)
	BatchDequeueMessages() ([]Message, error)
}
