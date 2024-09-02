package helpers

import (
	"log"
	"os"
	"strings"
)

func IsRemoteDB() bool {
    remoteDbFlag := os.Getenv("USE_REMOTE_DB")

    if remoteDbFlag == "true" {
        return true
    }
	sstStage := os.Getenv("SST_STAGE")
	// `.github/workflows/deploy-feature.yml` deploys any branch that begins with `feature/*` to aws as `feature-*`
	return sstStage == "prod" || sstStage == "dev" || strings.HasPrefix(sstStage, "feature-")
}

func GetDbTableName(tableName string) string {
	// this must be added in stacks/ApiStack.ts and stacks/StorageStack.ts
	var SST_Table_tableName = os.Getenv("SST_Table_tableName_" + tableName)
	if !IsRemoteDB() {
        if os.Getenv("GO_ENV") != "test" {
            log.Printf("Log Get Db Table: %v", tableName)
        }
		return tableName
	}
    if os.Getenv("GO_ENV") != "test" {
        log.Printf("Log Get Db Table: %v", SST_Table_tableName)
    }
    if SST_Table_tableName == "" {
        return ""
    }
	return SST_Table_tableName
}

func GetMarqoEndpoint() string {
    marqoEndpoint := os.Getenv("MARQO_API_BASE_URL")
    return marqoEndpoint
}

