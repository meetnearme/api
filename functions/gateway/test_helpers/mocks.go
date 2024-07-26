package test_helpers


import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/types"
)

type MockDynamoDBClient struct {
    ScanFunc func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
    PutItemFunc func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
    GetItemFunc func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
    UpdateItemFunc func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

func (m *MockDynamoDBClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
    return m.ScanFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
    if m.PutItemFunc != nil {
        return m.PutItemFunc(ctx, params, optFns...)
    }
    return &dynamodb.PutItemOutput{}, nil
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
    return m.GetItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
    if m.UpdateItemFunc != nil {
        return m.UpdateItemFunc(ctx, params, optFns...)
    }
    return &dynamodb.UpdateItemOutput{}, nil
}


// MockGeoService
type MockGeoService struct{
    GetGeoFunc func(location, baseUrl string) (string, string, string, error)
}

func (m *MockGeoService) GetGeo(location, baseUrl string) (string, string, string, error) {
    return "40.7128", "-74.0060", "New York, NY 10001, USA", nil
}

// MochSeshuService mocks the UpdateSeshuSession function
type MockSeshuService struct{}

func (m *MockSeshuService) UpdateSeshuSession(ctx context.Context, db types.DynamoDBAPI, update types.SeshuSessionUpdate) (*types.SeshuSessionUpdate, error) {
    return &update, nil
}

func (m *MockSeshuService) GetSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSession) (*types.SeshuSession, error) {
    // Return mock data
    return &types.SeshuSession{
        OwnerId: "mockOwner",
        Url: seshuPayload.Url,
        Status: "draft",
        // Fill in other fields as needed
    }, nil
}

func (m *MockSeshuService) InsertSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionInput) (*types.SeshuSessionInsert, error) {
    // Return mock data
    return &types.SeshuSessionInsert{
        OwnerId: seshuPayload.OwnerId,
        Url: seshuPayload.Url,
        Status: "draft",
        // Fill in other fields as needed
    }, nil
}

// MockTemplateRenderer mocks the template rendering process
type MockTemplateRenderer struct{}

func (m *MockTemplateRenderer) Render(ctx context.Context, buf *bytes.Buffer) error {
    // Simulate rendering by writing a mock HTML string to the buffer
    _, err := buf.WriteString("<div>Mock rendered template</div>")
    return err
}


