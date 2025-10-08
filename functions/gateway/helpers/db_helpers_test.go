package helpers

import (
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
)

func init() {
	os.Setenv("GO_ENV", constants.GO_TEST_ENV)
}

func TestIsRemoteDB(t *testing.T) {
	tests := []struct {
		name           string
		useRemoteDB    string
		sstStage       string
		expectedResult bool
	}{
		{"Use remote DB", "true", "", true},
		{"Production stage", "", "prod", true},
		{"Empty environment", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("USE_REMOTE_DB", tt.useRemoteDB)
			os.Setenv("ACT_STAGE", tt.sstStage)
			result := IsRemoteDB()
			if result != tt.expectedResult {
				t.Errorf("IsRemoteDB() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestGetDbTableName(t *testing.T) {
	t.Run("Local DB", func(t *testing.T) {
		os.Setenv("USE_REMOTE_DB", "false")
		os.Setenv("ACT_STAGE", "")
		result := GetDbTableName("TestTable")
		if result != "TestTable" {
			t.Errorf("GetDbTableName(\"TestTable\") = %s, want TestTable", result)
		}
	})

	t.Run("Remote DB", func(t *testing.T) {
		os.Setenv("USE_REMOTE_DB", "true")
		os.Setenv("SST_Table_tableName_TestTable", "RemoteTestTable")
		result := GetDbTableName("TestTable")
		if result != "RemoteTestTable" {
			t.Errorf("GetDbTableName(\"TestTable\") = %s, want RemoteTestTable", result)
		}
	})
}
