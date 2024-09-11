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

func init() {
	rdsDataClient = CreateRDSClient()
}

// CreateRDSClient initializes and returns an RDS Data API client
func CreateRDSClient() internal_types.RDSDataAPI {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v", err)
	}

	client := rdsdata.NewFromConfig(cfg)

	clusterArn := os.Getenv("RDS_CLUSTER_ARN")
	secretArn := os.Getenv("RDS_SECRET_ARN")
	databaseName := os.Getenv("DATABASE_NAME")

	if clusterArn == "" || secretArn == "" {
		log.Fatalf("Missing RDS cluster or secret ARN in environment variables")
	}

	log.Println("Create RDS Data API client successful.")

	// Verify the connection by running a simple query

	verifyQuery := `SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema NOT IN ('information_schema', 'pg_catalog')
  AND table_type = 'BASE TABLE'
ORDER BY table_schema, table_name;`
	exOutput, err := client.ExecuteStatement(context.TODO(), &rdsdata.ExecuteStatementInput{
		ResourceArn: &clusterArn,
		SecretArn:   &secretArn,
		Database:    &databaseName,
		Sql:         &verifyQuery,
	})

	if err != nil {
		log.Fatalf("Error executing verify query: %v", err)
	}

	// Print the table names
	fmt.Println("User-created tables in the database:")
	for _, record := range exOutput.Records {
		var schemaName, tableName string
		for j, field := range record {
			switch v := field.(type) {
			case *rds_types.FieldMemberStringValue:
				if j == 0 {
					schemaName = v.Value
				} else if j == 1 {
					tableName = v.Value
				}
			}
		}
		if tableName != "" {
			fmt.Printf(" - %s.%s\n", schemaName, tableName)
		}
	}

	log.Println("Database connection verified successfully.")

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
	// Build and execute the SQL statement
	input := &rdsdata.ExecuteStatementInput{
		ResourceArn: &r.clusterArn,
		SecretArn:   &r.secretArn,
		Database:    &r.database,
		Sql:         &sql,
		FormatRecordsAs: "JSON",
		Parameters:  params,
	}

	log.Printf("params in exec: %v", params)

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
			testRdsData = &test_helpers.MockRdsDataClient{} // Implement MockRdsDataClient for testing
		}
		log.Println("Returning mock RDS Data API client for testing")
		return testRdsData
	}
	onceRds.Do(func() {
		rdsDataClient = CreateRDSClient()
	})
	return rdsDataClient
}

