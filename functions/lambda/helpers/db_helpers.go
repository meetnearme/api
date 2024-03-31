package helpers

import (
	"os"
	"strings"

	"github.com/meetnearme/api/functions/lambda/shared"
)

func IsDeployed() bool {
	sstStage := os.Getenv("SST_STAGE")
	// `.github/workflows/deploy-feature.yml` deploys any branch that begins with `feature/*` to aws as `feature-*`
	return sstStage == "prod" || strings.HasPrefix(sstStage, "feature-")
}

func GetDbTableName(tableName string) string {
	var SST_Table_tableName_Events = os.Getenv("SST_Table_tableName_" + shared.EventsTablePrefix)

	if !IsDeployed() {
		return tableName
	}
	return SST_Table_tableName_Events
}
