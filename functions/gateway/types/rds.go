package types

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

// RDSDataAPI defines the interface for interacting with the RDS Data API
type RDSDataAPI interface {
    ExecStatement(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error)
}

