package rds_service

// import (
// 	"context"
// 	"encoding/json"
// 	"testing"
// 	"time"

// 	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
// 	"github.com/google/uuid"
// 	internal_types "github.com/meetnearme/api/functions/gateway/types"
// 	"github.com/meetnearme/api/functions/gateway/test_helpers"
// )

// // Helper function to create a test user
// func createTestUser() internal_types.UserInsert {
// 	return internal_types.UserInsert{
// 		ID:                  uuid.NewString(),
// 		Name:                "John Doe",
// 		Email:               "john.doe@example.com",
// 		Address:             "123 Main St",
// 		Phone:               "555-1234",
// 		ProfilePictureURL:   "http://example.com/profile.jpg",
// 		CategoryPreferences: []string{"Sports", "Music"},
// 		Role:                "User",
// 	}
// }

// // Helper function to create mock ExecStatement output
// func createMockExecStatementOutput(records [][]types.Field) *types.ExecuteStatementOutput {
// 	return &types.ExecuteStatementOutput{
// 		FormattedRecords: &records,
// 	}
// }

// // Helper function to create JSON records for mock data
// func createJSONRecords(user internal_types.User) []map[string]interface{} {
// 	return []map[string]interface{}{
// 		{
// 			"ID":                  user.ID,
// 			"Name":                user.Name,
// 			"Email":               user.Email,
// 			"Address":             user.Address,
// 			"Phone":               user.Phone,
// 			"ProfilePictureURL":   user.ProfilePictureURL,
// 			"CategoryPreferences": user.CategoryPreferences,
// 			"Role":                user.Role,
// 		},
// 	}
// }

// // Custom mock RDSDataClient
// type MockRdsDataClient struct {
// 	ExecStatementFunc func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error)
// }

// func (m *MockRdsDataClient) ExecStatement(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 	if m.ExecStatementFunc != nil {
// 		return m.ExecStatementFunc(ctx, statement, parameters)
// 	}
// 	return nil, nil
// }

// // Test InsertUser
// func TestInsertUser(t *testing.T) {
// 	mockRDS := &MockRdsDataClient{}
// 	userService := NewUserService()

// 	testUser := createTestUser()
// 	expectedUser := &internal_types.User{
// 		ID:                  testUser.ID,
// 		Name:                testUser.Name,
// 		Email:               testUser.Email,
// 		Address:             testUser.Address,
// 		Phone:               testUser.Phone,
// 		ProfilePictureURL:   testUser.ProfilePictureURL,
// 		CategoryPreferences: testUser.CategoryPreferences,
// 		Role:                testUser.Role,
// 		CreatedAt:           time.Now().UTC(),
// 		UpdatedAt:           time.Now().UTC(),
// 	}

// 	mockRDS.ExecStatementFunc = func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 		return createMockExecStatementOutput(createJSONRecords(*expectedUser)), nil
// 	}

// 	result, err := userService.InsertUser(context.Background(), mockRDS, testUser)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// 	if !equalUsers(result, expectedUser) {
// 		t.Errorf("expected %v, got %v", expectedUser, result)
// 	}
// }

// // Test GetUserByID
// func TestGetUserByID(t *testing.T) {
// 	mockRDS := &MockRdsDataClient{}
// 	userService := NewUserService()

// 	testUserID := uuid.NewString()
// 	expectedUser := &internal_types.User{
// 		ID:                  testUserID,
// 		Name:                "John Doe",
// 		Email:               "john.doe@example.com",
// 		Address:             "123 Main St",
// 		Phone:               "555-1234",
// 		ProfilePictureURL:   "http://example.com/profile.jpg",
// 		CategoryPreferences: []string{"Sports", "Music"},
// 		Role:                "User",
// 		CreatedAt:           time.Now().UTC(),
// 		UpdatedAt:           time.Now().UTC(),
// 	}

// 	mockRDS.ExecStatementFunc = func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 		return createMockExecStatementOutput(createJSONRecords(*expectedUser)), nil
// 	}

// 	result, err := userService.GetUserByID(context.Background(), mockRDS, testUserID)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// 	if !equalUsers(result, expectedUser) {
// 		t.Errorf("expected %v, got %v", expectedUser, result)
// 	}
// }

// // Test GetUsers
// func TestGetUsers(t *testing.T) {
// 	mockRDS := &MockRdsDataClient{}
// 	userService := NewUserService()

// 	expectedUsers := []internal_types.User{
// 		{
// 			ID:                  uuid.NewString(),
// 			Name:                "John Doe",
// 			Email:               "john.doe@example.com",
// 			Address:             "123 Main St",
// 			Phone:               "555-1234",
// 			ProfilePictureURL:   "http://example.com/profile.jpg",
// 			CategoryPreferences: []string{"Sports", "Music"},
// 			Role:                "User",
// 			CreatedAt:           time.Now().UTC(),
// 			UpdatedAt:           time.Now().UTC(),
// 		},
// 	}

// 	records := []map[string]interface{}{}
// 	for _, user := range expectedUsers {
// 		records = append(records, createJSONRecords(user)[0])
// 	}

// 	mockRDS.ExecStatementFunc = func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 		return createMockExecStatementOutput(records), nil
// 	}

// 	result, err := userService.GetUsers(context.Background(), mockRDS)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// 	if len(result) != len(expectedUsers) {
// 		t.Fatalf("expected %v users, got %v", len(expectedUsers), len(result))
// 	}
// 	for i, user := range expectedUsers {
// 		if !equalUsers(&result[i], &user) {
// 			t.Errorf("expected %v, got %v", user, result[i])
// 		}
// 	}
// }

// // Test UpdateUser
// func TestUpdateUser(t *testing.T) {
// 	mockRDS := &MockRdsDataClient{}
// 	userService := NewUserService()

// 	testUserID := uuid.NewString()
// 	updateUser := internal_types.UserUpdate{
// 		Name:                "Jane Doe",
// 		Email:               "jane.doe@example.com",
// 		Address:             "456 Elm St",
// 		Phone:               "555-5678",
// 		ProfilePictureURL:   "http://example.com/profile-new.jpg",
// 		CategoryPreferences: []string{"Tech", "Art"},
// 		Role:                "Admin",
// 	}

// 	expectedUser := &internal_types.User{
// 		ID:                  testUserID,
// 		Name:                updateUser.Name,
// 		Email:               updateUser.Email,
// 		Address:             updateUser.Address,
// 		Phone:               updateUser.Phone,
// 		ProfilePictureURL:   updateUser.ProfilePictureURL,
// 		CategoryPreferences: updateUser.CategoryPreferences,
// 		Role:                updateUser.Role,
// 		CreatedAt:           time.Now().UTC(),
// 		UpdatedAt:           time.Now().UTC(),
// 	}

// 	mockRDS.ExecStatementFunc = func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 		return createMockExecStatementOutput(createJSONRecords(*expectedUser)), nil
// 	}

// 	result, err := userService.UpdateUser(context.Background(), mockRDS, testUserID, updateUser)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// 	if !equalUsers(result, expectedUser) {
// 		t.Errorf("expected %v, got %v", expectedUser, result)
// 	}
// }

// // Test DeleteUser
// func TestDeleteUser(t *testing.T) {
// 	mockRDS := &MockRdsDataClient{}
// 	userService := NewUserService()

// 	testUserID := uuid.NewString()

// 	mockRDS.ExecStatementFunc = func(ctx context.Context, statement string, parameters map[string]interface{}) (*types.ExecuteStatementOutput, error) {
// 		return &types.ExecuteStatementOutput{
// 			NumberOfRecordsUpdated: 1,
// 		}, nil
// 	}

// 	err := userService.DeleteUser(context.Background(), mockRDS, testUserID)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// }

// // Helper function to compare two User objects for equality
// func equalUsers(a, b *internal_types.User) bool {
// 	if a == nil || b == nil {
// 		return a == b
// 	}
// 	return a.ID == b.ID &&
// 		a.Name == b.Name &&
// 		a.Email == b.Email &&
// 		a.Address == b.Address &&
// 		a.Phone == b.Phone &&
// 		a.ProfilePictureURL == b.ProfilePictureURL &&
// 		equalStringSlices(a.CategoryPreferences, b.CategoryPreferences) &&
// 		a.Role == b.Role &&
// 		a.CreatedAt.Equal(b.CreatedAt) &&
// 		a.UpdatedAt.Equal(b.UpdatedAt)
// }

// // Helper function to compare two slices of strings
// func equalStringSlices(a, b []string) bool {
// 	if len(a) != len(b) {
// 		return false
// 	}
// 	for i, v := range a {
// 		if v != b[i] {
// 			return false
// 		}
// 	}
// 	return true
// }

