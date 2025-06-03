package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
)

const vectorizer = "text2vec-transformers"
const eventClassName = "EventStrict" //

func GetWeaviateClient() (*weaviate.Client, error) {
	weaviateHost := os.Getenv("WEAVIATE_HOST")
	weaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	weaviatePort := os.Getenv("WEAVIATE_PORT")
	weaviateApiKey := os.Getenv("WEAVIATE_API_KEY_ALLOWED_KEYS")

	if weaviateHost == "" {
		weaviateHost = "localhost"
	}

	if weaviateScheme == "" {
		weaviateScheme = "http"
	}

	if weaviatePort == "" {
		weaviatePort = "8080"
	}

	if weaviateApiKey == "" {
		log.Printf("Please add a weaviate API Key")
	}

	weaviateHostPort := weaviateHost + ":" + weaviatePort

	cfg := weaviate.Config{
		Host:       weaviateHostPort,
		Scheme:     weaviateScheme,
		AuthConfig: auth.ApiKey{Value: weaviateApiKey},
		Headers:    nil,
		// May need AuthConfig, need to look at Marqo impl
		// what do we want our time out to be
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Weaviate client: %w", err)
	}

	return client, nil
}

type WeaviateServiceInterface interface {
	UpsertEventToWeaviate(client *weaviate.Client, event types.Event) (*models.Object, error)
}

type WeaviateService struct{}

func NewWeaviateService() *WeaviateService {
	return &WeaviateService{}
}

func InsertSimpleTestDoc(client *weaviate.Client, className string, docID strfmt.UUID, message string) error {

	testDoc := map[string]interface{}{
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	_, err := client.Data().Creator().
		WithClassName(className).
		WithID(string(docID)).
		WithProperties(testDoc).
		Do(context.Background())

	return err
}

// GetSimpleTestDoc retrieves a single document by its ID.
func GetSimpleTestDoc(client *weaviate.Client, className string, docID strfmt.UUID) (*models.Object, error) {
	data, err := client.Data().ObjectsGetter().
		WithClassName(className).
		WithID(string(docID)).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("document not found")
	}
	return data[0], nil
}
