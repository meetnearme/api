package services

import (
	"context"
	"os"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
)

var (
	natsService     interfaces.NatsServiceInterface
	natsServiceOnce sync.Once
)

func GetNatsService(ctx context.Context) (interfaces.NatsServiceInterface, error) {
	natsServiceOnce.Do(func() {
		if os.Getenv("GO_ENV") == "test" {
			// natsService = getMockNatService()
		} else {
			conn, err := GetNatsClient()
			if err != nil {
				panic(err) // Or handle initialization error gracefully
			}
			natsService, err = NewNatsService(ctx, conn)
			if err != nil {
				panic(err) // Or handle initialization error gracefully
			}
		}
	})
	return natsService, nil
}
