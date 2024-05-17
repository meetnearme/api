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
	return sstStage == "prod" || strings.HasPrefix(sstStage, "feature-")
}

func GetDbTableName(tableName string) string {
	var SST_Table_tableName_Events = os.Getenv("SST_Table_tableName_" + EventsTablePrefix)

	if !IsRemoteDB() {
        log.Printf("Log Get Db Table: %v", tableName)
		return tableName
	}
    log.Printf("Log Get Db Table: %v", SST_Table_tableName_Events)
	return SST_Table_tableName_Events
}

