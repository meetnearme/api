package services

import (
	"context"
	"os"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
)

var (
	postgresService     interfaces.PostgresServiceInterface
	postgresServiceOnce sync.Once
)

func GetPostgresService(ctx context.Context) interfaces.PostgresServiceInterface {
	postgresServiceOnce.Do(func() {
		if os.Getenv("GO_ENV") == "test" {
			// postgresService = getMockPostgresService()
		} else {
			db, err := GetPostgresClient(ctx)
			if err != nil {
				panic(err) // Or handle initialization error gracefully
			}
			postgresService = NewPostgresService(db)
		}
	})
	return postgresService
}
