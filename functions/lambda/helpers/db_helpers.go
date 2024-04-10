package helpers

import (
	"os"
	"strings"
)

func IsDeployed() bool {
	sstStage := os.Getenv("SST_STAGE")
	// `.github/workflows/deploy-feature.yml` deploys any branch that begins with `feature/*` to aws as `feature-*`
	return sstStage == "prod" || strings.HasPrefix(sstStage, "feature-")
}

func GetDbTableName(tableName string) string {
	var SST_Table_tableName_Events = os.Getenv("SST_Table_tableName_" + EventsTablePrefix)

	if !IsDeployed() {
		return tableName
	}
	return SST_Table_tableName_Events
}
