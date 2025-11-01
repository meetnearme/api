package helpers

import (
	"os"
)

func IsRemoteDB() bool {
	remoteDbFlag := os.Getenv("USE_REMOTE_DB")

	if remoteDbFlag == "true" {
		return true
	}
	actStage := os.Getenv("ACT_STAGE")
	return actStage == "prod" || actStage == "dev"
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
