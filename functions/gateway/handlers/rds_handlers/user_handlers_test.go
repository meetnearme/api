package rds_handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var mockUser = &internal_types.User{
	ID:                "123",
	Name:              "Test User",
	Email:             "test@example.com",
	Address:           "123 Test St",
	Phone:             "555-555-5555",
	ProfilePictureURL: "https://example.com/pic.jpg",
	Role: "standard_user",
}

func TestCreateUser_Success(t *testing.T) {
	// Setup mock user service
	mockUserService := &rds_service.MockUserService{}
	mockUserService.InsertUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error) {
		return mockUser, nil
	}

	// Create a UserHandler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a valid request body
	createUser := internal_types.UserInsert{
		Name:              "Test User",
		Email:             "test@example.com",
		CategoryPreferences: "[\"category1\", \"category2\"]",
		Role: "standard_user",
	}
	body, _ := json.Marshal(createUser)

	// Create a new HTTP request for creating the user
	req := httptest.NewRequest("POST", "/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the CreateUser handler
	userHandler.CreateUser(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check the response body (should contain the created user data)
	expected, _ := json.Marshal(mockUser)
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expected))
	}
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	// Setup mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create a UserHandler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create an invalid request body (malformed JSON)
	body := []byte(`{"name": "Test User", "email": "test@example.com", "role": "standard_user", "category_preferences": ["category1", "category2"`)

	// Create a new HTTP request for creating the user
	req := httptest.NewRequest("POST", "/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the CreateUser handler
	userHandler.CreateUser(rr, req)

	// Check the status code (should be 422 Unprocessable Entity)
	if status := rr.Code; status != http.StatusUnprocessableEntity {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
	}

	// Check the response body for the expected error message
	expected := "Invalid JSON payload"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestCreateUser_MissingRequiredFields(t *testing.T) {
	// Setup mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create a UserHandler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request body with missing required fields (e.g., no 'Name' or 'Role')
	createUser := internal_types.UserInsert{
		Email: "test@example.com", // Missing Name and Role
	}
	body, _ := json.Marshal(createUser)

	// Create a new HTTP request for creating the user
	req := httptest.NewRequest("POST", "/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the CreateUser handler
	userHandler.CreateUser(rr, req)

	// Check the status code (should be 400 Bad Request)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Check the response body for the expected validation error message
	expected := "Invalid body"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}


func TestCreateUser_DBInsertError(t *testing.T) {
	// Setup mock user service with an insert error
	mockUserService := &rds_service.MockUserService{}
	mockUserService.InsertUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error) {
		return nil, fmt.Errorf("database error")
	}

	// Create a UserHandler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a valid request body
	createUser := internal_types.UserInsert{
		Name:              "Test User",
		Email:             "test@example.com",
		Role:              "standard_user",
		CategoryPreferences: "[\"category1\", \"category2\"]",
	}
	body, _ := json.Marshal(createUser)

	// Create a new HTTP request for creating the user
	req := httptest.NewRequest("POST", "/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the CreateUser handler
	userHandler.CreateUser(rr, req)

	// Check the status code (should be 500 Internal Server Error)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// Check the response body for the expected error message
	expected := "Failed to create user"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestGetUser_Success(t *testing.T) {
	// Mock user service with a successful response
	mockUserService := &rds_service.MockUserService{}
	mockUser := &internal_types.User{
		ID:                "123",
		Name:              "Test User",
		Email:             "test@example.com",
		CategoryPreferences: []string{"category1", "category2"},
		Role:              "standard_user",
	}
	mockUserService.GetUserByIDFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error) {
		return mockUser, nil
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID
	req := httptest.NewRequest("GET", "/user/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the GetUser handler
	userHandler.GetUser(rr, req)

	// Check the status code (should be 200 OK)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body for the user data
	expected, _ := json.Marshal(mockUser)
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expected))
	}
}

func TestGetUser_MissingUserID(t *testing.T) {
	// Mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with no user ID
	req := httptest.NewRequest("GET", "/user", nil)

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the GetUser handler
	userHandler.GetUser(rr, req)

	// Check the status code (should be 400 Bad Request)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Check the response body for the error message
	expected := "Missing user ID"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestGetUser_UserNotFound(t *testing.T) {
	// Mock user service with no user found
	mockUserService := &rds_service.MockUserService{}
	mockUserService.GetUserByIDFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error) {
		return nil, nil
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID
	req := httptest.NewRequest("GET", "/user/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the GetUser handler
	userHandler.GetUser(rr, req)

	// Check the status code (should be 404 Not Found)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Check the response body for the error message
	expected := "User not found"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestGetUser_DBError(t *testing.T) {
	// Mock user service with a DB error
	mockUserService := &rds_service.MockUserService{}
	mockUserService.GetUserByIDFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error) {
		return nil, fmt.Errorf("mock db error")
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID
	req := httptest.NewRequest("GET", "/user/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the GetUser handler
	userHandler.GetUser(rr, req)

	// Check the status code (should be 500 Internal Server Error)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// Check the response body for the error message
	expected := "Failed to get user: mock db error"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}
func TestUpdateUser_Success(t *testing.T) {
	// Mock user service with a successful update
	mockUserService := &rds_service.MockUserService{}
	mockUser := &internal_types.User{
		ID:                "123",
		Name:              "Updated User",
		Email:             "updated@example.com",
		CategoryPreferences: []string{"category1", "category2"},
		Role:              "standard_user",
	}
	mockUserService.UpdateUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, update internal_types.UserUpdate) (*internal_types.User, error) {
		return mockUser, nil
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID and valid update payload
	updateUser := internal_types.UserUpdate{
		Name:  "Updated User",
		Email: "updated@example.com",
	}
	body, _ := json.Marshal(updateUser)
	req := httptest.NewRequest("PUT", "/user/123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 200 OK)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body for the updated user data
	expected, _ := json.Marshal(mockUser)
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expected))
	}
}

func TestUpdateUser_MissingUserID(t *testing.T) {
	// Mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with no user ID
	updateUser := internal_types.UserUpdate{
		Name:  "Updated User",
		Email: "updated@example.com",
	}
	body, _ := json.Marshal(updateUser)
	req := httptest.NewRequest("PUT", "/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 400 Bad Request)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Check the response body for the error message
	expected := "Missing user ID"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestUpdateUser_ReadBodyError(t *testing.T) {
	// Mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID but simulate a read error by closing the body
	req := httptest.NewRequest("PUT", "/user/123", nil)
	req.Body = io.NopCloser(&brokenReader{})
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 400 Bad Request)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Check the response body for the error message
	expected := "Failed to read request body"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestUpdateUser_InvalidJSONPayload(t *testing.T) {
	// Mock user service
	mockUserService := &rds_service.MockUserService{}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID but invalid JSON payload
	req := httptest.NewRequest("PUT", "/user/123", strings.NewReader("{invalid_json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 422 Unprocessable Entity)
	if status := rr.Code; status != http.StatusUnprocessableEntity {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
	}

	// Check the response body for the error message
	expected := "Invalid JSON payload"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}


func TestUpdateUser_UserNotFound(t *testing.T) {
	// Mock user service with no user found
	mockUserService := &rds_service.MockUserService{}
	mockUserService.UpdateUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, update internal_types.UserUpdate) (*internal_types.User, error) {
		return nil, nil
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID and valid update payload
	updateUser := internal_types.UserUpdate{
		Name:  "Updated User",
		Email: "updated@example.com",
	}
	body, _ := json.Marshal(updateUser)
	req := httptest.NewRequest("PUT", "/user/123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 404 Not Found)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Check the response body for the error message
	expected := "User not found"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

func TestUpdateUser_DBError(t *testing.T) {
	// Mock user service with a DB error
	mockUserService := &rds_service.MockUserService{}
	mockUserService.UpdateUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, update internal_types.UserUpdate) (*internal_types.User, error) {
		return nil, errors.New("database error")
	}

	// Create the handler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID and valid update payload
	updateUser := internal_types.UserUpdate{
		Name:  "Updated User",
		Email: "updated@example.com",
	}
	body, _ := json.Marshal(updateUser)
	req := httptest.NewRequest("PUT", "/user/123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the UpdateUser handler
	userHandler.UpdateUser(rr, req)

	// Check the status code (should be 500 Internal Server Error)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// Check the response body for the error message
	expected := "Failed to update user"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want contains %v", rr.Body.String(), expected)
	}
}

// Helper to simulate a broken reader
type brokenReader struct{}

func (br *brokenReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (br *brokenReader) Close() error {
	return nil
}

func TestDeleteUser_Success(t *testing.T) {
	// Mock user service
	mockUserService := &rds_service.MockUserService{}
	mockUserService.DeleteUserFunc = func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error {
		return nil // Simulate successful deletion
	}

	// Create a UserHandler with the mock service
	userHandler := NewUserHandler(mockUserService)

	// Create a request with a valid user ID
	req := httptest.NewRequest("DELETE", "/user/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	// Use httptest to create a response recorder
	rr := httptest.NewRecorder()

	// Call the DeleteUser handler
	userHandler.DeleteUser(rr, req)

	// Check the status code (should be 200 OK)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body (should contain success message)
	expected := "User successfully deleted"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

