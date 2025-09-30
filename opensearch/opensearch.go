package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/tozny/utils-go/logging"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

// ElasticClient wraps a Client for Elasticsearch interactions. The struct also includes a logger, that will get configured, in instantiation methods.
type OpenSearchClient struct {
	Client *opensearchapi.Client
	logging.Logger
}

// ElasticConfig wraps configuration to create either local or AWS Elasticsearch Client.
type OpenSearchConfig struct {
	UseLocal    bool
	Debug       bool
	Logger      logging.Logger
	Region      string
	URL         string
	AccessKey   string
	SecretKey   string
	ServiceName string
}

// OpenSearchQueryResult wraps results of an OpenSearchQuery
type OpenSearchQueryResult struct {
	Hits      []opensearchapi.SearchHit
	TotalHits int
}

type BulkItem struct {
	Document interface{}
	ID       string
}

type OpenSearchSearchParams struct {
	Size   *int
	From   *int
	SortBy []string
}

const (
	BulkCreate = "create"
	BulkIndex  = "index"
)

// CreateIndex creates OpenSearch Index if it doesn't already exist. Indexes consist of a name and must be provided with a context. The index created has default indexers and tokenizers.
// Unless a non-default settings, such as indexers and tokenizers are explicitly needed this function is preferred over CreateIndexWithSettings
func (osc *OpenSearchClient) CreateIndex(ctx context.Context, name string) error {
	createResponse, err := osc.Client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: name,
		})
	var opensearchError *opensearch.StructError
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "resource_already_exists_exception" {
				return err
			}
		} else {
			return err
		}
	} else if !createResponse.Acknowledged {
		return fmt.Errorf("index was never acknowledged")
	}
	return nil
}

// CreateIndexWithSettings creates OpenSearch Index if it doesn't already exist, atttaching an index body. The settings body can be used to add custom indexers and other options that
// a index may need. In many cases using the CreateIndex function is sufficient.
// https://docs.opensearch.org/latest/api-reference/index-apis/create-index/
func (osc *OpenSearchClient) CreateIndexWithSettings(ctx context.Context, name string, settings string) error {
	settingsReader := strings.NewReader(settings)
	createResponse, err := osc.Client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: name,
			Body:  settingsReader,
		})
	var opensearchError *opensearch.StructError
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "resource_already_exists_exception" {
				return err
			}
		} else {
			return err
		}
	} else if !createResponse.Acknowledged {
		return fmt.Errorf("index was never acknowledged")
	}
	return nil
}

// DeleteIndex deletes OpenSearch Index.
// Should not be used called outside of local environment or without caution and intention.
func (osc *OpenSearchClient) DeleteIndex(ctx context.Context, name string) error {
	deleteResponse, err := osc.Client.Indices.Delete(
		ctx,
		opensearchapi.IndicesDeleteReq{
			Indices: []string{name},
		})
	if err != nil {
		return err
	}
	if !deleteResponse.Acknowledged {
		return fmt.Errorf("index deletion was never acknowledged")
	}
	return nil
}

// AddIndexMapping adds an explicit mapping to an existing recordType within indexName.
// https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping.html
// Most indexes should have an explicit mapping to ensure that records are enforced to a specific schema
func (osc *OpenSearchClient) AddIndexMapping(ctx context.Context, indexName string, mapping string) error {
	mappingReader := strings.NewReader(mapping)
	mappingResp, err := osc.Client.Indices.Mapping.Put(
		ctx,
		opensearchapi.MappingPutReq{
			Indices: []string{indexName},
			Body:    mappingReader,
		})
	if !mappingResp.Acknowledged {
		return fmt.Errorf("add index mapping was never acknowledged")
	}
	return err
}

func (osc *OpenSearchClient) BulkIndexForIndex(ctx context.Context, index string, docs []BulkItem) (map[string]error, error) {
	errorMap := make(map[string]error)
	body, err := bulkFormatForIndex(docs)
	if err != nil {
		return errorMap, err
	}
	bulkResp, err := osc.Client.Bulk(
		ctx,
		opensearchapi.BulkReq{
			Index: index,
			Body:  body,
		})

	indexedActions := indexed(*bulkResp)

	for _, actions := range indexedActions {
		if actions.Error != nil {
			errorMap[actions.ID] = fmt.Errorf("indexed %+v", actions.Error)
			// This error type is the elastic search response when a the bulk update queue is full.
			// When this error occurs upstream callers should backoff and retry
			if actions.Error.Type == "rejected_execution_exception" {
				err = fmt.Errorf("rejected_execution_exception %+v", actions.Error)
				return errorMap, err
			}
		} else if actions.Shards.Successful <= 0 {
			errorMap[actions.ID] = fmt.Errorf("no successful shards for index action %+v", actions.ID)
		}
	}
	return errorMap, err
}

func bulkFormatForIndex(bulkItems []BulkItem) (io.Reader, error) {
	body := &bytes.Buffer{}
	enc := json.NewEncoder(body)

	for _, bulkItem := range bulkItems {
		// metadata line
		meta := map[string]map[string]string{
			"index": {},
		}
		if len(bulkItem.ID) > 0 {
			meta["index"] = map[string]string{"_id": bulkItem.ID}
		}
		if err := enc.Encode(meta); err != nil {
			return nil, fmt.Errorf("encode meta for ID %s: %w", bulkItem.ID, err)
		}

		// document line
		if err := enc.Encode(bulkItem.Document); err != nil {
			return nil, fmt.Errorf("encode doc for ID %s: %w", bulkItem.ID, err)
		}
	}
	return body, nil
}

// Indexed returns all bulk request results of "index" actions.
func indexed(bulkResp opensearchapi.BulkResp) []*opensearchapi.BulkRespItem {
	return byAction(bulkResp, "index")
}

// Created returns all bulk request results of "create" actions.
func created(bulkResp opensearchapi.BulkResp) []*opensearchapi.BulkRespItem {
	return byAction(bulkResp, "create")
}

// Updated returns all bulk request results of "update" actions.
func updated(bulkResp opensearchapi.BulkResp) []*opensearchapi.BulkRespItem {
	return byAction(bulkResp, "update")
}

// Deleted returns all bulk request results of "delete" actions.
func deleted(bulkResp opensearchapi.BulkResp) []*opensearchapi.BulkRespItem {
	return byAction(bulkResp, "delete")
}

// ByAction returns all bulk request results of a certain action,
// e.g. "index" or "delete".
func byAction(bulkResp opensearchapi.BulkResp, action string) []*opensearchapi.BulkRespItem {
	if bulkResp.Items == nil {
		return nil
	}
	var items []*opensearchapi.BulkRespItem
	for _, item := range bulkResp.Items {
		if result, found := item[action]; found {
			items = append(items, &result)
		}
	}
	return items
}

// InsertDocument inserts a json compatable interface as a document into OpenSearch. Returns true if no error occurred and false if not, as well as the error
func (osc *OpenSearchClient) InsertDocument(ctx context.Context, index string, id string, document interface{}) (bool, error) {
	docJson, err := json.Marshal(document)
	if err != nil {
		return false, fmt.Errorf("insert document marshal error for doc ID %s: %w", id, err)
	}

	insertResp, err := osc.Client.Document.Create(
		ctx,
		opensearchapi.DocumentCreateReq{
			Index:      index,
			DocumentID: id,
			Body:       bytes.NewReader(docJson),
		})
	if err != nil {
		osc.Logger.Printf("OpenSearch Document Failed to insert: %s", docJson)
		return false, err
	}
	if insertResp.Shards.Successful <= 0 {
		return false, fmt.Errorf("openSearch Document of index %s, id %s and body %+v, did not error but failed to load", index, id, document)
	}
	return true, err
}

// IndexDocument indexes (overrides if the document is already created) a json compatable interface as a document into OpenSearch. Returns true if no error occurred and false if not, as well as the error
func (osc *OpenSearchClient) IndexDocument(ctx context.Context, index string, id string, document interface{}) (bool, error) {
	docJson, err := json.Marshal(document)
	if err != nil {
		return false, fmt.Errorf("indexing document marshal error for doc ID %s: %w", id, err)
	}

	insertResp, err := osc.Client.Index(
		ctx,
		opensearchapi.IndexReq{
			Index:      index,
			DocumentID: id,
			Body:       bytes.NewReader(docJson),
		})
	if err != nil {
		osc.Logger.Printf("OpenSearch Document Failed to index: %s", docJson)
		return false, err
	}
	if insertResp.Shards.Successful <= 0 {
		return false, fmt.Errorf("openSearch Document of index %s, id %s and body %+v, did not error but failed to load", index, id, document)
	}
	return true, err
}

func (osc *OpenSearchClient) DeleteDocument(ctx context.Context, index string, id string) (bool, error) {
	_, err := osc.Client.Document.Delete(
		ctx,
		opensearchapi.DocumentDeleteReq{
			Index:      index,
			DocumentID: id,
		})
	if err != nil {
		osc.Errorf("openSearch document failed to delete: %s", err)
		return false, err
	}
	return true, err
}

func (osc *OpenSearchClient) DeleteDocumentByQuery(ctx context.Context, index string, query interface{}) error {
	queryJson, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("delete document by query marshal error: %w", err)
	}
	_, err = osc.Client.Document.DeleteByQuery(
		ctx,
		opensearchapi.DocumentDeleteByQueryReq{
			Indices: []string{index},
			Body:    bytes.NewReader(queryJson),
			Params:  opensearchapi.DocumentDeleteByQueryParams{Conflicts: "proceed"},
		})
	if err != nil {
		osc.Errorf("openSearch document failed to delete: %s", err)
		return err
	}
	return nil
}

// QueryForMetrics queries elastic search cluster for metrics documents that meet the search param criteria and return them
func (osc *OpenSearchClient) SearchQuery(ctx context.Context, index string, query interface{}, params OpenSearchSearchParams) (OpenSearchQueryResult, error) {
	queryJson, err := json.Marshal(query)
	if err != nil {
		return OpenSearchQueryResult{}, fmt.Errorf("search query marshal error for query: %s: %w", query, err)
	}

	searchReq := &opensearchapi.SearchReq{
		Indices: []string{index},
		Body:    bytes.NewReader(queryJson),
	}

	if params.Size != nil || params.From != nil || len(params.SortBy) > 0 {
		searchParams := opensearchapi.SearchParams{}

		if params.Size != nil {
			searchParams.Size = params.Size
		}
		if params.From != nil {
			searchParams.From = params.From
		}
		if len(params.SortBy) > 0 {
			searchParams.Sort = params.SortBy
		}

		searchReq.Params = searchParams
	}

	searchResp, err := osc.Client.Search(
		ctx,
		searchReq)
	if err != nil {
		return OpenSearchQueryResult{}, err
	}

	return OpenSearchQueryResult{
		Hits:      searchResp.Hits.Hits,
		TotalHits: searchResp.Hits.Total.Value,
	}, nil
}

// NewOpenSearchClient returns a new client for Opensearch, local or hosted through AWS.
// The UseLocal flag determines which client is created.
// With AWS Config this client can be used with elastic search clusters in AWS, both managed and hosted.
func NewOpenSearchClient(ctx context.Context, osConfig OpenSearchConfig) (*OpenSearchClient, error) {
	var client *opensearchapi.Client
	var err error
	if osConfig.UseLocal {
		client, err = opensearchapi.NewClient(
			opensearchapi.Config{
				Client: opensearch.Config{
					Addresses: []string{osConfig.URL},
				},
			})

	} else {
		awsCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(osConfig.Region),
			config.WithCredentialsProvider(
				getCredentialProvider(osConfig.AccessKey, osConfig.SecretKey, ""),
			),
		)
		if err != nil {
			return nil, err
		}

		// Create an AWS request Signer and load AWS configuration using default config folder or env vars.
		signer, err := requestsigner.NewSignerWithService(awsCfg, "es") // Use "aoss" for Amazon OpenSearch Serverless
		if err != nil {
			return nil, err
		}
		// Create an opensearch client and use the request-signer.
		client, err = opensearchapi.NewClient(
			opensearchapi.Config{
				Client: opensearch.Config{
					Addresses: []string{osConfig.URL},
					Signer:    signer,
				},
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return &OpenSearchClient{
		Client: client,
		Logger: osConfig.Logger,
	}, err
}

func getCredentialProvider(accessKey, secretAccessKey, token string) aws.CredentialsProviderFunc {
	return func(ctx context.Context) (aws.Credentials, error) {
		c := &aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretAccessKey,
			SessionToken:    token,
		}
		return *c, nil
	}
}
