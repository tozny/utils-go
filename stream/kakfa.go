package stream

import (
	"fmt"

	"github.com/tozny/utils-go/logging"

	"github.com/Shopify/sarama"
)

const (
	AnyPartitionPublishFlag = -1  // Value to use to signal Kafka client to publish messages to any partition
	SubscribeBufferSize     = 256 // Max number of messages to buffer when subscribing to a stream
)

// KafkaStreamConfig wraps configuration for a Kafka stream
type KafkaStreamConfig struct {
	BrokerEndpoints     []string       // List of broker endpoints used to publish and or subscribe to this Kafka stream
	Topic               string         // Which Kafka service endpoint to use for stream interactions
	Logger              logging.Logger // Logger to use for stream trace logs
	Partition           int32          // Kafka server defined shard of the stream to consume and publish messages from
	Offset              int64          // Offset to use for determining where in the stream to start consuming and subscribing to messages
	SubscribeBufferSize int            // Max Number of messages to buffer when subscribing to a stream
}

// KafkaStream wraps a concrete (Apacha Kafka) distributed stream
// processing backend for publishing and subscribing to events.
type KafkaStream struct {
	BrokerEndpoints []string            // List of broker endpoints used to publish and or subscribe to this Kafka stream
	logger          logging.Logger      // Logger to use for stream trace logs
	config          KafkaStreamConfig   // Private and static configuration for this Kafka stream
	producer        sarama.SyncProducer // Private Kafka client for synchronous publishing of messages to a Kafka stream
	consumer        sarama.Consumer     // Private Kafka client for consuming messages from a Kafka stream
}

func convertEventToMessage(event Event, partition int32) *sarama.ProducerMessage {
	message := &sarama.ProducerMessage{
		Topic:     event.Topic,
		Partition: partition,
	}
	if event.Tag != "" {
		message.Key = sarama.StringEncoder(event.Tag)
	}
	if event.Message != "" {
		message.Value = sarama.StringEncoder(event.Message)
	}
	return message
}

// Publish publishes N events to the underlying Kafka stream,
// returning the published events and error (if any).
func (ks *KafkaStream) Publish(events []Event) ([]Event, error) {
	for index, event := range events {
		message := convertEventToMessage(event, ks.config.Partition)
		partition, offset, err := ks.producer.SendMessage(message)
		if err != nil {
			return events, err
		}
		events[index].Partition = string(partition)
		events[index].SortKey = fmt.Sprint(offset)
		ks.logger.Debugf("Publish: published event %+v", events[index])
	}
	return events, nil
}

func convertMessageToEvent(message *sarama.ConsumerMessage, topic string) Event {
	return Event{
		Topic:     topic,
		Tag:       string(message.Key),
		Message:   string(message.Value),
		Timestamp: message.Timestamp,
		Partition: fmt.Sprint(message.Partition),
		SortKey:   fmt.Sprint(message.Offset),
	}
}

// Subscribe opens a connection to a Kafka stream, returning a channel
// upon which messages published to the topic will be delivered on
// and error (if any) opening the connection.
// The caller can cancel the subscription at anytime and close the connection
// by closing the provided close channel.
func (ks *KafkaStream) Subscribe(close chan struct{}) (<-chan Event, error) {
	// Capture current state of stream for use throughout this connection
	topic := ks.config.Topic
	offset := ks.config.Offset
	streamPartitions, err := ks.consumer.Partitions(topic)
	// Set up return channel for subscription events
	events := make(chan Event, ks.config.SubscribeBufferSize)
	if err != nil {
		return events, err
	}
	// Start subscription to stream in background
	go func() {
		// For each partition in the stream set up a consumer to subscribe to messages
		// published to that partition
		for _, partition := range streamPartitions {
			partitionConsumer, err := ks.consumer.ConsumePartition(topic, partition, offset)
			if err != nil {
				ks.logger.Errorf("Subscribe: Error %s to starting consumer for partition %d", partition, err)
				continue
			}
			// Start goroutine to run until the close channel is closed by the caller
			go func(partitionConsumer sarama.PartitionConsumer) {
				<-close
				ks.logger.Debug("Subscribe: Received close signal")
				// at which point the connection to this partition consumer should be closed
				partitionConsumer.AsyncClose()
			}(partitionConsumer)
			// Start goroutine to run until the close channel is closed by the caller
			go func(partitionConsumer sarama.PartitionConsumer) {
				// to consume and convert messages for the subscriber to receive
				for message := range partitionConsumer.Messages() {
					event := convertMessageToEvent(message, topic)
					ks.logger.Debugf("Subscribe: Received event %+v", event)
					events <- event
				}
			}(partitionConsumer)
		}
	}()

	return events, nil
}

// NewKafkaStream idempotently creates a Kafka stream using the provided configuration,
// returning a stream interface wrapping the Kafka stream connection and error (if any).
func NewKafkaStream(config KafkaStreamConfig) (Stream, error) {
	kafkaStream := &KafkaStream{}

	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Partitioner = sarama.NewHashPartitioner

	kafkaProducer, err := sarama.NewSyncProducer(config.BrokerEndpoints, kafkaConfig)
	if err != nil {
		return kafkaStream, err
	}
	kafkaStream.producer = kafkaProducer

	kafkaConsumer, err := sarama.NewConsumer(config.BrokerEndpoints, kafkaConfig)
	if err != nil {
		return kafkaStream, err
	}
	kafkaStream.consumer = kafkaConsumer

	if config.Partition == 0 {
		// By default publish to any partition for the given stream and topic
		config.Partition = AnyPartitionPublishFlag
	}

	kafkaStream.config = config
	kafkaStream.logger = config.Logger

	return kafkaStream, nil
}
