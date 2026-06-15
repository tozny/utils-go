package queue

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/tozny/utils-go/logging"
)

const (
	SQSBatchEnqueueLimit = 10 // Max number of messages SQS will let you enqueue at once
)

var (
	BatchSizeExceededError = errors.New(fmt.Sprintf("can not batch enqueue more than %d messages", SQSBatchEnqueueLimit))
)

// SQSQueueConfig wraps configuration for an SQS queue
type SQSQueueConfig struct {
	QueueName                string         // The name of the queue to configure
	SQSEndpoint              string         // Which SQS service endpoint to use for queue interactions
	SQSRegion                string         // Which AWS region the queue is located in e.g. us-west-2
	APIKeyID                 string         // AWS API Secret Key ID for IAM user with sqs permissions
	APIKeySecret             string         // AWS API Secret Key for IAM user with sqs permissions
	VisibilityTimeoutSeconds int64          // How long a message should be invisible after being dequeued
	DequeueBatchSize         int64          // Max number of messages that can be dequeued
	PollSeconds              int64          // How long to poll for dequeueable messages when dequeing messages from the queue
	Logger                   logging.Logger // Logger to use for queue trace logs
}

// Queue wraps a concrete(AWS SQS) distributed queue for
// enqueuing and dequeuing messages across a network.
type SQSQueue struct {
	Name                     string
	url                      string
	sqsClient                *sqs.Client
	visibilityTimeoutSeconds int64
	dequeueBatchSize         int64
	pollSeconds              int64
	logger                   logging.Logger
}

// DeleteMessage deletes the message with messageID from the queue
// returning error (if any).
func (q *SQSQueue) DeleteMessage(messageID string) error {
	_, err := q.sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.url),
		ReceiptHandle: aws.String(messageID),
	})
	return err
}

// EnqueueMessage enqueues a single message to the queue, returning error (if any).
func (q *SQSQueue) EnqueueMessage(message Message) error {
	sendMessageRequest := &sqs.SendMessageInput{
		MessageAttributes: convertTagsToSQSMessageAttributes(message.Tags),
		MessageBody:       aws.String(message.Body),
		QueueUrl:          aws.String(q.url),
	}
	_, err := q.sqsClient.SendMessage(context.Background(), sendMessageRequest)
	return err
}

// BatchEnqueueMessages enqueues a batch of messages to the queue,
// returning the messages that failed to enqueue and error (if any).
// BatchEnqueueMessages will fail immediately if more
// than `BatchEnqueueLimit` messages are passed.
func (q *SQSQueue) BatchEnqueueMessages(messages []Message) ([]Message, error) {
	if len(messages) > SQSBatchEnqueueLimit {
		return messages, BatchSizeExceededError
	}
	// Create lookup table for tracking and returning
	// messages that failed to enqueue
	var messageToSQSLookup = map[string]*Message{}
	var sqsBatchRequestEntries []types.SendMessageBatchRequestEntry
	for messageIndex, message := range messages {
		messageID := uuid.New().String()
		// Populate lookup table in case this message
		// fails as part of the batch enqueue request
		messageToSQSLookup[messageID] = &messages[messageIndex]
		sqsBatchRequestEntries = append(sqsBatchRequestEntries, types.SendMessageBatchRequestEntry{
			Id:                aws.String(messageID),
			MessageAttributes: convertTagsToSQSMessageAttributes(message.Tags),
			MessageBody:       aws.String(message.Body),
		})
	}
	// Construct SendMessageBatch request
	sendMessageBatchRequest := &sqs.SendMessageBatchInput{
		Entries:  sqsBatchRequestEntries,
		QueueUrl: aws.String(q.url),
	}
	sendMessageBatchResponse, err := q.sqsClient.SendMessageBatch(context.Background(), sendMessageBatchRequest)
	if err != nil {
		q.logger.Printf("BatchEnqueueMessages error %s for batch %+v\n", err, sqsBatchRequestEntries)
	}
	// Return any messages that failed to enqueue
	failedToEnqueueMessages := []Message{}
	if sendMessageBatchResponse != nil {
		for _, failure := range sendMessageBatchResponse.Failed {
			failedToEnqueueMessages = append(failedToEnqueueMessages, *messageToSQSLookup[*failure.Id])
		}
	}
	return failedToEnqueueMessages, err
}

// Dequeue dequeues a single messages from the queue,
// returning dequeued messages and error (if any).
func (q *SQSQueue) DequeueMessage() (Message, error) {
	var message Message
	previousDequeueBatchSize := q.dequeueBatchSize
	q.dequeueBatchSize = 1
	defer func() { q.dequeueBatchSize = previousDequeueBatchSize }()
	messages, err := q.BatchDequeueMessages()
	if err != nil {
		return message, err
	}
	if len(messages) == 0 {
		return message, err
	}
	message = messages[0]
	return message, err
}

// BatchDequeue dequeues ups to `q.DequeueBatchSize` messages from the queue,
// returning dequeued messages and error (if any).
func (q *SQSQueue) BatchDequeueMessages() ([]Message, error) {
	var dequeuedMessages []Message
	// construct ReceiveMessage request
	receiveMessageRequest := sqs.ReceiveMessageInput{
		MessageSystemAttributeNames: []types.MessageSystemAttributeName{
			types.MessageSystemAttributeNameSentTimestamp,
			types.MessageSystemAttributeNameApproximateReceiveCount,
		},
		MessageAttributeNames: []string{"All"},
		QueueUrl:              aws.String(q.url),
		MaxNumberOfMessages:   int32(q.dequeueBatchSize),
		VisibilityTimeout:     int32(q.visibilityTimeoutSeconds),
		WaitTimeSeconds:       int32(q.pollSeconds),
	}

	// make ReceiveMessage request
	receiveMessageResponse, err := q.sqsClient.ReceiveMessage(context.Background(), &receiveMessageRequest)
	if err != nil {
		return dequeuedMessages, err
	}
	// Convert from SQS message to Queue message
	for _, receivedMessage := range receiveMessageResponse.Messages {
		message, err := convertSQSMessageToQueueMessage(receivedMessage)
		if err != nil {
			q.logger.Printf("error %s converting %+v to Message type\n", err, receivedMessage)
			continue
		}
		dequeuedMessages = append(dequeuedMessages, *message)
	}
	return dequeuedMessages, err
}

// convertSQSMessageToQueueMessage converts data from the SQSMessage type
// to the generic Message type, returning the converted message and error (if any).
func convertSQSMessageToQueueMessage(sqsMessage types.Message) (*Message, error) {
	var message *Message
	approximateReceiveCount := sqsMessage.Attributes["ApproximateReceiveCount"]
	receiveCount, err := strconv.Atoi(approximateReceiveCount)
	if err != nil {
		return message, err
	}
	message = &Message{
		Body:         aws.ToString(sqsMessage.Body),
		ReceiptID:    aws.ToString(sqsMessage.ReceiptHandle),
		ReceiveCount: receiveCount,
		Tags:         map[string]string{},
	}
	for messageAttribute, messageAttributeValue := range sqsMessage.MessageAttributes {
		message.Tags[messageAttribute] = aws.ToString(messageAttributeValue.StringValue)
	}
	return message, err
}

// convertTagsToSQSMessageAttributes converts a message's tag(s) to a map of tag key
// to a SQS MessageAttributeValue.
func convertTagsToSQSMessageAttributes(tags map[string]string) map[string]types.MessageAttributeValue {
	var messageAttributes map[string]types.MessageAttributeValue
	if len(tags) == 0 {
		return messageAttributes
	}
	messageAttributes = map[string]types.MessageAttributeValue{}
	for key, value := range tags {
		messageAttributes[key] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(value),
		}
	}
	return messageAttributes
}

// NewSQSQueue idempotently creates a SQS queue using the provided configuration,
// returning a queue interface wrapping the sqs queue connection and error (if any).
func NewSQSQueue(config SQSQueueConfig) (Queue, error) {
	var sqsQueue *SQSQueue
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.SQSRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.APIKeyID,
			config.APIKeySecret,
			"" /*AWS_SESSION_TOKEN*/,
		)),
	)
	if err != nil {
		return sqsQueue, err
	}
	sqsClient := sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
		if config.SQSEndpoint != "" {
			o.BaseEndpoint = aws.String(config.SQSEndpoint)
		}
	})
	// Create the queue using params from config
	createQueueResponse, err := sqsClient.CreateQueue(context.Background(),
		&sqs.CreateQueueInput{
			QueueName: aws.String(config.QueueName),
		})
	if err != nil {
		return sqsQueue, err
	}
	sqsQueue = &SQSQueue{
		Name:                     config.QueueName,
		url:                      aws.ToString(createQueueResponse.QueueUrl),
		sqsClient:                sqsClient,
		visibilityTimeoutSeconds: config.VisibilityTimeoutSeconds,
		dequeueBatchSize:         config.DequeueBatchSize,
		pollSeconds:              config.PollSeconds,
		logger:                   config.Logger,
	}
	return sqsQueue, err
}
