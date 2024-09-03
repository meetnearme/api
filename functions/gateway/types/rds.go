package types

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
)

// RDSAPI defines the interface for interacting with RDS
// type RDSAPI interface {
// 	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
// 	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
// 	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
// }

// RDSDataAPI defines the interface for interacting with the RDS Data API
type RDSDataAPI interface {
	ExecStatement(ctx context.Context, sql string) (*rdsdata.ExecuteStatementOutput, error)
}
