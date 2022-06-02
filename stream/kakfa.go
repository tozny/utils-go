package stream

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudevent "github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"github.com/tozny/utils-go/logging"
	"log"
)

const (
	AnyPartitionPublishFlag = -1  // Value to use to signal Kafka client to publish messages to any partition
	SubscribeBufferSize     = 256 // Max number of messages to buffer when subscribing to a stream
	defaultReceiverGroupId  = "tozny-cloudevents"
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
	BrokerEndpoints []string               // List of broker endpoints used to publish and or subscribe to this Kafka stream
	logger          logging.Logger         // Logger to use for stream trace logs
	config          KafkaStreamConfig      // Private and static configuration for this Kafka stream
	producer        sarama.SyncProducer    // Private Kafka client for synchronous publishing of messages to a Kafka stream
	consumer        sarama.Consumer        // Private Kafka client for consuming messages from a Kafka stream
	sender          *kafka_sarama.Sender   // Private Kafka client for sending CloudEvents from a Kafka stream
	receiver        *kafka_sarama.Consumer // Private Kafka client for consuming CloudEvents from a Kafka stream
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
				ks.logger.Errorf("Subscribe: Error %s to starting consumer for partition %d", err, partition)
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

	// Initialize a CloudEvents Sender Client and add it to the KafkaStream
	sender, err := kafka_sarama.NewSenderFromSyncProducer(config.Topic, kafkaProducer)
	if err != nil {
		log.Fatalf("Failed to create sender: %s", err.Error())
	}
	kafkaStream.sender = sender

	receiver, err := kafka_sarama.NewConsumer(config.BrokerEndpoints, kafkaConfig, defaultReceiverGroupId, config.Topic)
	if err != nil {
		log.Fatalf("Failed to create receiver: %s", err.Error())
	}
	kafkaStream.receiver = receiver

	return kafkaStream, nil
}

// Send accepts an event, translates it to a CloudEvent and publishes it to the underlying Kafka stream,
// returns an error (if any).
func (ks *KafkaStream) Send(event CloudEvent) error {
	//defer ks.sender.Close(context.Background())
	client, err := cloudevents.NewClient(ks.sender, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		log.Fatalf("Failed to create CloudEvents client, %v", err)
		return err
	}

	cloudEvent := createCloudEventFromEvent(event)
	if result := client.Send(
		kafka_sarama.WithMessageKey(context.Background(), sarama.StringEncoder(event.Tag)),
		cloudEvent,
	); cloudevents.IsUndelivered(result) {
		log.Printf("Failed to send: %v", result)
	} else {
		log.Printf("Message accepted: %t", cloudevents.IsACK(result))
	}
	return nil
}

// Receive starts a kafka CloudEvents receiver for consuming messages from the kafka stream
// accepts a channel that receives a connection close signal
// returns a channel on which the received messages are pushed and an error (if any)
func (ks *KafkaStream) Receive(close chan struct{}) (<-chan CloudEvent, error) {
	events := make(chan CloudEvent)
	client, err := cloudevents.NewClient(ks.receiver, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		log.Fatalf("Failed to create receiver client, %v", err)
		return events, err
	}
	// Start the receiver
	go func() {
		log.Printf("Listening to consuming topic %s\n", ks.config.Topic)
		err = client.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
			events <- createEventFromCloudEvent(event)
		})
		if err != nil {
			log.Fatalf("Failed to start receiver: %s", err)
		} else {
			log.Printf("Receiver stopped\n")
		}
	}()
	// Start goroutine to run until the close channel is closed by the caller
	go func(receiver *kafka_sarama.Consumer) {
		<-close
		ks.logger.Debug("Receiver: close signal")
		err := receiver.Close(context.Background())
		if err != nil {
			log.Fatalf("Failed to close receiver")
			return
		}
	}(ks.receiver)
	return events, nil
}

func createCloudEventFromEvent(event CloudEvent) cloudevent.Event {
	e := cloudevents.NewEvent()
	e.SetID(uuid.New().String())
	e.SetType(event.Type)
	e.SetSource(event.Source)
	e.SetTime(event.Timestamp)
	_ = e.SetData(event.ContentType, event.Data)
	return e
}

func createEventFromCloudEvent(event cloudevents.Event) CloudEvent {
	return CloudEvent{
		Type:        event.Type(),
		Source:      event.Source(),
		ContentType: event.DataContentType(),
		Data:        event.Data(),
		Timestamp:   event.Time(),
	}
}
