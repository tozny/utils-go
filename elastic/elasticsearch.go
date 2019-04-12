package elasticsearch

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/olivere/elastic"
	aws "github.com/olivere/elastic/aws/v4"
)

// ElasticClient wraps a Client for Elasticsearch interactions. The struct also includes a logger, that will get configured, in instantiation methods.
type ElasticClient struct {
	*elastic.Client
	*log.Logger
}

// ElasticAWSConfig wraps necessary configuration for AWS Elasticsearch Client.
type ElasticAWSConfig struct {
	Region      string
	URL         string
	AccessKey   string
	SecretKey   string
	ServiceName string
}

// CreateIndex creates Elasticsearch Index if it doesn't already exist. Indexes consist of a name and must be provided with a context. The index created has default indexers and tokenizers.
// Unless a non-default settings, such as indexers and tokenizers are explicitly needed this function is preferred over CreateIndexWithSettings
func (ec *ElasticClient) CreateIndex(ctx context.Context, name string) error {
	return ec.CreateIndexWithSettings(ctx, name, "")
}

// CreateIndexWithSettings creates Elasticsearch Index if it doesn't already exist, atttaching an index body. The settings body can be used to add custom indexers and other options that
// a index may need. In many cases using the CreateIndex function is sufficient.
// https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html
func (ec *ElasticClient) CreateIndexWithSettings(ctx context.Context, name string, settings string) error {
	exists, err := ec.Client.IndexExists(name).Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		createdIndexResults, err := ec.Client.CreateIndex(name).BodyString(settings).Do(ctx)
		if err != nil {
			return err
		}
		if !createdIndexResults.Acknowledged {
			return fmt.Errorf("index was never acknowledged")
		}
	}
	return err
}

// DeleteIndex deletes Elasticsearch Index.
// Should not be used called outside of local environment or without caution and intention.
func (ec *ElasticClient) DeleteIndex(ctx context.Context, name string) error {
	deleteIndex, err := ec.Client.DeleteIndex(name).Do(ctx)
	if err != nil {
		return err
	}
	if !deleteIndex.Acknowledged {
		return fmt.Errorf("index deletion was never acknowledged")
	}
	return err
}

// AddIndexMapping adds an explicit mapping to an existing recordType within indexName.
// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping.html
// Most indexes should have an explicit mapping to ensure that records are enforced to a specific schema
func (ec *ElasticClient) AddIndexMapping(ctx context.Context, indexName string, recordType string, mapping string) error {
	params := make(url.Values)
	_, err := ec.Client.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "PUT",
		Path:   fmt.Sprintf("/%s/_mapping/%s", indexName, recordType),
		Params: params,
		Body:   mapping,
	})
	if err != nil {
		return err
	}
	return err
}

// NewLocalElasticClient returns a local Elasticsearch client. This client can be used with dockerized elastic search such as
// docker.elastic.co/elasticsearch/elasticsearch
func NewLocalElasticClient(localElasticURL string, serviceName string) (ElasticClient, error) {
	client, err := elastic.NewSimpleClient(
		elastic.SetURL(localElasticURL),
		// DEBUG: Uncomment the line below to enable full tracing of all http requests and responses
		// elastic.SetTraceLog(log.New(os.Stdout, "ElasticSearchClient:", log.LstdFlags)),
	)
	return ElasticClient{
		client,
		log.New(os.Stdout, fmt.Sprintf("%v-search: ", serviceName), log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}, err
}

// NewElasticClient returns a new client for AWS Elasticsearch. This client can be used with elastic search clusters in AWS, both managed and hosted.
func NewElasticClient(config ElasticAWSConfig) (ElasticClient, error) {
	signingClient := aws.NewV4SigningClient(credentials.NewStaticCredentials(
		config.AccessKey,
		config.SecretKey,
		"",
	), config.Region)
	client, err := elastic.NewClient(
		elastic.SetURL(config.URL),
		elastic.SetScheme("https"),
		elastic.SetSniff(false),
		elastic.SetHttpClient(signingClient),
	)
	return ElasticClient{
		client,
		log.New(os.Stdout, fmt.Sprintf("%v-queue: ", config.ServiceName), log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}, err
}
