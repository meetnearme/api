package rds_service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestInsertUser(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":                "test-id",
			"name":              "John Doe",
			"email":             "john@example.com",
			"address":           "123 Main St",
			"phone":             "1234567890",
			"profile_picture_url": "http://example.com/pic.jpg",
			"role":              "user",
			"created_at":        time.Now().Format(time.RFC3339),
			"updated_at":        time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewUserService()

	user := internal_types.UserInsert{
		Name:                "John Doe",
		Email:               "john@example.com",
		Address:             "123 Main St",
		Phone:               "1234567890",
		ProfilePictureURL:   "http://example.com/pic.jpg",
		Role:                "user",
		CreatedAt:           time.Now().Format(time.RFC3339),
		UpdatedAt:           time.Now().Format(time.RFC3339),
	}

	// Test
	result, err := service.InsertUser(context.Background(), rdsClient, user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "test-id" {
		t.Errorf("expected id 'test-id', got '%v'", result.ID)
	}
}

func TestGetUserByID(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":                "test-id",
			"name":              "John Doe",
			"email":             "john@example.com",
			"address":           "123 Main St",
			"phone":             "1234567890",
			"profile_picture_url": "http://example.com/pic.jpg",
			"role":              "user",
			"created_at":        time.Now().Format(time.RFC3339),
			"updated_at":        time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewUserService()

	// Test
	result, err := service.GetUserByID(context.Background(), rdsClient, "test-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "test-id" {
		t.Errorf("expected id 'test-id', got '%v'", result.ID)
	}
	if result.Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got '%v'", result.Email)
	}
}

func TestUpdateUser(t *testing.T) {
	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Setup
	records := []map[string]interface{}{
		{
			"id":                "test-id",
			"name":              "Jane Doe",
			"email":             "jane@example.com",
			"address":           "456 Another St",
			"phone":             "0987654321",
			"profile_picture_url": "http://example.com/newpic.jpg",
			"role":              "admin",
			"created_at":        time.Now().Format(rdsTimeFormat),
			"updated_at":        time.Now().Format(rdsTimeFormat),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewUserService()

	userUpdate := internal_types.UserUpdate{
		Name:              "Jane Doe",
		Email:             "jane@example.com",
		Address:           "456 Another St",
		Phone:             "0987654321",
		ProfilePictureURL: "http://example.com/newpic.jpg",
		Role:              "admin",
	}

	// Test
	result, err := service.UpdateUser(context.Background(), rdsClient, "test-id", userUpdate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.Role != "admin" {
		t.Errorf("expected role 'admin', got '%v'", result.Role)
	}
}

func TestDeleteUser(t *testing.T) {
    // Initialize mock RDS client
    rdsClient := &test_helpers.MockRdsDataClient{
        ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
            fmt.Printf("SQL: %s\n", sql)
            fmt.Printf("Params: %v\n", params)

            switch sql {
            case "DELETE FROM users WHERE id = :id":
                // Simulate successful delete
                return &rdsdata.ExecuteStatementOutput{
                    NumberOfRecordsUpdated: 1, // Simulate that one record was deleted
                }, nil
            case "SELECT * FROM users WHERE id = :id":
                // Simulate item not found after deletion
                return &rdsdata.ExecuteStatementOutput{
                    FormattedRecords: aws.String("[]"), // Simulate no records found
                }, nil
            default:
                return nil, fmt.Errorf("unexpected SQL query")
            }
        },
    }

    service := NewUserService()

    // Test deletion
    err := service.DeleteUser(context.Background(), rdsClient, "test-id")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    // Verify deletion by trying to retrieve the item
    result, err := rdsClient.ExecStatement(context.Background(), "SELECT * FROM users WHERE id = :id", []rds_types.SqlParameter{
        {
            Name: aws.String("id"),
            Value: &rds_types.FieldMemberStringValue{
                Value: "test-id",
            },
        },
    })

    if err != nil {
        t.Fatalf("failed to get item after deletion: %v", err)
    }

    if result.FormattedRecords == nil || *result.FormattedRecords == "[]" {
        // Pass the test if no records are found
        return
    }

    t.Fatalf("expected no records, got %v", *result.FormattedRecords)
}

func TestGetUsers(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":                "test-id",
			"name":              "John Doe",
			"email":             "john@example.com",
			"address":           "123 Main St",
			"phone":             "1234567890",
			"profile_picture_url": "http://example.com/pic.jpg",
			"role":              "user",
			"created_at":        time.Now().Format(time.RFC3339),
			"updated_at":        time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewUserService()

	// Test
	results, err := service.GetUsers(context.Background(), rdsClient)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if len(results) == 0 {
		t.Fatalf("expected results, got none")
	}
	if results[0].ID != "test-id" {
		t.Errorf("expected id 'test-id', got '%v'", results[0].ID)
	}
	if results[0].Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got '%v'", results[0].Email)
	}
}

func TestBuildSqlUserParams(t *testing.T) {
	tests := []struct {
		name      string
		parameters map[string]interface{}
		expected  []rds_types.SqlParameter
		expectErr bool
	}{
		{
			name: "valid parameters",
			parameters: map[string]interface{}{
				"id":                  "1234",
				"name":                "John Doe",
				"email":               "john@example.com",
				"role":                "admin",
				"created_at":          "2024-01-01 12:00:00",
				"updated_at":          "2024-01-02 12:00:00",
				"address":             "123 Main St",
				"phone":               "555-5555",
				"profile_picture_url": "http://example.com/pic.jpg",
			},
			expected: []rds_types.SqlParameter{
				{
					Name:     aws.String("id"),
					TypeHint: "UUID",
					Value: &rds_types.FieldMemberStringValue{
						Value: "1234",
					},
				},
				{
					Name: aws.String("name"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "John Doe",
					},
				},
				{
					Name: aws.String("email"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "john@example.com",
					},
				},
				{
					Name: aws.String("role"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "admin",
					},
				},
				{
					Name:     aws.String("created_at"),
					TypeHint: "TIMESTAMP",
					Value: &rds_types.FieldMemberStringValue{
						Value: "2024-01-01 12:00:00",
					},
				},
				{
					Name:     aws.String("updated_at"),
					TypeHint: "TIMESTAMP",
					Value: &rds_types.FieldMemberStringValue{
						Value: "2024-01-02 12:00:00",
					},
				},
				{
					Name: aws.String("address"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "123 Main St",
					},
				},
				{
					Name: aws.String("phone"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "555-5555",
					},
				},
				{
					Name: aws.String("profile_picture_url"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "http://example.com/pic.jpg",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "missing id",
			parameters: map[string]interface{}{
				"name": "John Doe",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "non-string id",
			parameters: map[string]interface{}{
				"id": 1234,
				"name": "John Doe",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "missing name",
			parameters: map[string]interface{}{
				"id": "1234",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "non-string name",
			parameters: map[string]interface{}{
				"id":   "1234",
				"name": 5678,
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSqlUserParams(tt.parameters)
			if (err != nil) != tt.expectErr {
				t.Errorf("buildSqlUserParams() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("buildSqlUserParams() = %v, want %v", got, tt.expected)
				return
			}
			for i, g := range got {
				e := tt.expected[i]
				if *g.Name != *e.Name || g.TypeHint != e.TypeHint || g.Value.(*rds_types.FieldMemberStringValue).Value != e.Value.(*rds_types.FieldMemberStringValue).Value {
					t.Errorf("buildSqlUserParams() = %v, want %v", got, tt.expected)
					return
				}
			}
		})
	}
}


func parseTime(value string, t *testing.T) time.Time {
    layout := "2006-01-02 15:04:05" // RDS SQL accepted time format
    parsedTime, err := time.Parse(layout, value)
    if err != nil {
        t.Fatalf("error parsing time for key: %s, error: %v", value, err)
    }
    return parsedTime
}


func TestExtractAndMapSingleUserFromJSON(t *testing.T) {
    tests := []struct {
        name      string
        jsonInput string
        expected  *internal_types.User
        expectErr bool
    }{
        {
            name: "valid JSON",
            jsonInput: `[
                {
                    "id": "1234",
                    "name": "John Doe",
                    "email": "john@example.com",
                    "address": "123 Main St",
                    "phone": "555-5555",
                    "profile_picture_url": "http://example.com/pic.jpg",
                    "role": "admin",
                    "created_at": "2024-01-01 12:00:00",
                    "updated_at": "2024-01-02 12:00:00"
                }
            ]`,
            expected: &internal_types.User{
                ID:                  "1234",
                Name:                "John Doe",
                Email:               "john@example.com",
                Address:             "123 Main St",
                Phone:               "555-5555",
                ProfilePictureURL:   "http://example.com/pic.jpg",
                Role:                "admin",
                CreatedAt:           parseTime("2024-01-01 12:00:00", t),
                UpdatedAt:           parseTime("2024-01-02 12:00:00", t),
            },
            expectErr: false,
        },
        // other test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := extractAndMapSingleUserFromJSON(tt.jsonInput)
            if (err != nil) != tt.expectErr {
                t.Errorf("extractAndMapSingleUserFromJSON() error = %v, expectErr %v", err, tt.expectErr)
                return
            }
            if tt.expected == nil {
                if got != nil {
                    t.Errorf("extractAndMapSingleUserFromJSON() = %v, want %v", got, tt.expected)
                }
                return
            }
            if *got != *tt.expected {
                t.Errorf("extractAndMapSingleUserFromJSON() = %v, want %v", got, tt.expected)
            }
        })
    }
}

func TestExtractUsersFromJson(t *testing.T) {
    tests := []struct {
        name      string
        jsonInput string
        expected  []internal_types.User
        expectErr bool
    }{
        {
            name: "valid JSON array",
            jsonInput: `[
                {
                    "id": "1234",
                    "name": "John Doe",
                    "email": "john@example.com",
                    "address": "123 Main St",
                    "phone": "555-5555",
                    "profile_picture_url": "http://example.com/pic.jpg",
                    "role": "admin",
                    "created_at": "2024-01-01 12:00:00",
                    "updated_at": "2024-01-02 12:00:00"
                },
                {
                    "id": "5678",
                    "name": "Jane Doe",
                    "email": "jane@example.com",
                    "address": "456 Elm St",
                    "phone": "555-1234",
                    "profile_picture_url": "http://example.com/pic2.jpg",
                    "role": "user",
                    "created_at": "2024-01-01 12:00:00",
                    "updated_at": "2024-01-02 12:00:00"
                }
            ]`,
            expected: []internal_types.User{
                {
                    ID:                  "1234",
                    Name:                "John Doe",
                    Email:               "john@example.com",
                    Address:             "123 Main St",
                    Phone:               "555-5555",
                    ProfilePictureURL:   "http://example.com/pic.jpg",
                    Role:                "admin",
					CreatedAt:           parseTime("2024-01-01 12:00:00", t),
					UpdatedAt:           parseTime("2024-01-02 12:00:00", t),
                },
                {
                    ID:                  "5678",
                    Name:                "Jane Doe",
                    Email:               "jane@example.com",
                    Address:             "456 Elm St",
                    Phone:               "555-1234",
                    ProfilePictureURL:   "http://example.com/pic2.jpg",
                    Role:                "user",
					CreatedAt:           parseTime("2024-01-01 12:00:00", t),
					UpdatedAt:           parseTime("2024-01-02 12:00:00", t),
                },
            },
            expectErr: false,
        },
        {
            name: "empty JSON array",
            jsonInput: `[]`,
            expected:  []internal_types.User{},
            expectErr: false,
        },
        {
            name: "invalid JSON",
            jsonInput: `[{invalid json}]`,
            expected:  nil,
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := extractUsersFromJson(tt.jsonInput)
            if (err != nil) != tt.expectErr {
                t.Errorf("extractUsersFromJson() error = %v, expectErr %v", err, tt.expectErr)
                return
            }
            if len(got) != len(tt.expected) {
                t.Errorf("extractUsersFromJson() = %v, want %v", got, tt.expected)
                return
            }
            for i, g := range got {
                e := tt.expected[i]
                if g != e {
                    t.Errorf("extractUsersFromJson() = %v, want %v", g, e)
                }
            }
        })
    }
}

