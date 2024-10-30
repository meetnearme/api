package helpers

import (
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
		return tableName
	}
	if SST_Table_tableName == "" {
		return ""
	}
	return SST_Table_tableName
}
