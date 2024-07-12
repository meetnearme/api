package helpers

import (
	"os"
	"strings"
	"testing"
)

func init() {
    os.Setenv("GO_ENV", "test")
}

func TestFormatDate(t *testing.T) {
    tests := []struct {
        name string
        input string
        expected string
    }{
        {"Valid date", "2099-05-01T12:00:00Z", "May 1, 2099 (Fri)"},
        {"Invalid date", "invalid-date", "Invalid date"},
        {"Empty string", "", "Invalid date"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatDate(tt.input)
            if result != tt.expected {
                t.Errorf("FormatDate(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}

func TestFormatTime(t *testing.T) {
    tests := []struct {
        name string
        input string
        expected string
    }{
        {"Valid time", "2023-05-01T14:30:00Z", "2:30pm"},
        {"Invalid time", "invalid-time", "Invalid time"},
        {"Empty string", "", "Invalid time"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatTime(tt.input)
            if result != tt.expected {
                t.Errorf("FormatTime(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}


func TestIsRemoteDB(t *testing.T) {
    tests := []struct {
        name string
        useRemoteDB string
        sstStage string
        expectedResult bool
    }{
        {"Use remote DB", "true", "", true},
        {"Production stage", "", "prod", true},
        {"Feature branch", "", "feature-test", true},
        {"Local development", "", "dev", false},
        {"Empty environment", "", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            os.Setenv("USE_REMOTE_DB", tt.useRemoteDB)
            os.Setenv("SST_STAGE", tt.sstStage)
            result := IsRemoteDB()
            if result != tt.expectedResult {
                t.Errorf("IsRemoteDB() = %v, want %v", result, tt.expectedResult)
            }
        })
    }
}

func TestGetDbTableName(t *testing.T) {
    originalEnv := os.Environ()
    defer func() {
        for _, pair := range originalEnv {
            parts := strings.SplitN(pair, "=", 2)
            os.Setenv(parts[0], parts[1])
        }
    }()

    tests := []struct {
        name string
        useRemoteDB string
        sstStage string
        tableName string
        remoteTableEnv string
        expectedResult string
    }{
        {"Local DB", "false", "", "TestTable", "RemoteTestTable", "TestTable"},
        {"Remote DB", "true", "prod", "TestTable", "RemoteTestTable", "RemoteTestTable"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            os.Clearenv()
            os.Setenv("USE_REMOTE_DB", tt.useRemoteDB)
            os.Setenv("SST_STAGE", tt.sstStage)
            os.Setenv("SST_Table_tableName_"+EventsTablePrefix, "RemoteTestTable")
            os.Setenv("GO_ENV", "test")

            result := GetDbTableName(tt.tableName)
            if result != tt.expectedResult {
                t.Errorf("GetDbTableName(%q) = %q, want %q", tt.tableName, result, tt.expectedResult)
            }
        })
    }
}
