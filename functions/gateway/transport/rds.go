package transport

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var (
	rdsDataClient internal_types.RDSDataAPI
	onceRds       sync.Once
	testRdsData   internal_types.RDSDataAPI
)

// CreateRDSClient initializes and returns an RDS Data API client
func CreateRDSClient() internal_types.RDSDataAPI {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v", err)
	}

	clusterArn := os.Getenv("RDS_CLUSTER_ARN")
	secretArn := os.Getenv("RDS_SECRET_ARN")
	databaseName := os.Getenv("DATABASE_NAME")

	if clusterArn == "" || secretArn == "" || databaseName == "" {
		log.Fatalf("Missing RDS cluster, secret ARN, or database name in environment variables")
	}

	client := rdsdata.NewFromConfig(cfg)

	return &RDSDataClient{
		client:     client,
		clusterArn: clusterArn,
		secretArn:  secretArn,
		database:   databaseName,
	}
}

// RDSDataClient wraps around *rdsdata.Client and implements the RDSDataAPI interface
type RDSDataClient struct {
	client     *rdsdata.Client
	clusterArn string
	secretArn  string
	database   string
}

func (r *RDSDataClient) ExecStatement(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
	input := &rdsdata.ExecuteStatementInput{
		ResourceArn: &r.clusterArn,
		SecretArn:   &r.secretArn,
		Database:    &r.database,
		Sql:         &sql,
		Parameters:  params,
		FormatRecordsAs: "JSON",
	}
	return r.client.ExecuteStatement(ctx, input)
}


// SetTestRdsDB sets a mock RDS Data API client for testing
func SetTestRdsDB(db internal_types.RDSDataAPI) {
	testRdsData = db
}

// GetRdsDB returns the singleton RDS Data API client
func GetRdsDB() internal_types.RDSDataAPI {
	if os.Getenv("GO_ENV") == "test" {
		if testRdsData == nil {
			log.Println("Creating mock RDS Data API client for testing")
			testRdsData = &test_helpers.MockRdsDataClient{
				ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
					return &rdsdata.ExecuteStatementOutput{}, nil
				},
			}
		}
		log.Println("Returning mock RDS Data API client for testing")
		return testRdsData
	}
	onceRds.Do(func() {
		rdsDataClient = CreateRDSClient()
	})
	return rdsDataClient
}
