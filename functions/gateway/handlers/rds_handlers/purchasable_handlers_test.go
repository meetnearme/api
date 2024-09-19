package rds_handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

// func TestGetPurchasables(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		mockPurchasables []internal_types.Purchasable
// 		mockError      error
// 		expectedStatus int
// 	}{
// 		{
// 			name: "Success",
// 			mockPurchasables: []internal_types.Purchasable{
// 				{ID: "1", Name: "Item 1", Cost: 10.99, ItemType: "Type1", Inventory: 100},
// 				{ID: "2", Name: "Item 2", Cost: 20.99, ItemType: "Type2", Inventory: 50},
// 			},
// 			mockError:      nil,
// 			expectedStatus: http.StatusOK,
// 		},
// 		{
// 			name:           "No Purchasables Found",
// 			mockPurchasables: []internal_types.Purchasable{},
// 			mockError:      nil,
// 			expectedStatus: http.StatusOK,
// 		},
// 		{
// 			name:           "Service Error",
// 			mockPurchasables: nil,
// 			mockError:      errors.New("database error"),
// 			expectedStatus: http.StatusInternalServerError,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mockService := &rds_service.MockPurchasableService{
// 				GetPurchasablesByUserIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.Purchasable, error) {
// 					return tt.mockPurchasables, tt.mockError
// 				},
// 			}
// 			handler := NewPurchasableHandler(mockService)

// 			req := httptest.NewRequest(http.MethodGet, "/purchasables", nil)
// 			w := httptest.NewRecorder()

// 			handler.GetPurchasable(w, req)

// 			res := w.Result()
// 			defer res.Body.Close()

// 			if res.StatusCode != tt.expectedStatus {
// 				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, res.StatusCode)
// 			}

// 			if tt.expectedStatus == http.StatusOK {
// 				var purchasables []internal_types.Purchasable
// 				err := json.NewDecoder(res.Body).Decode(&purchasables)
// 				if err != nil {
// 					t.Fatalf("Failed to decode response body: %v", err)
// 				}
// 				if len(purchasables) != len(tt.mockPurchasables) {
// 					t.Errorf("Expected %d purchasables, got %d", len(tt.mockPurchasables), len(purchasables))
// 				}
// 			}
// 		})
// 	}
// }

func TestGetPurchasable(t *testing.T) {
	tests := []struct {
		name           string
		purchasableID  string
		mockPurchasable *internal_types.Purchasable
		mockError      error
		expectedStatus int
	}{
		{
			name:          "Success",
			purchasableID: "1",
			mockPurchasable: &internal_types.Purchasable{
				ID: "1", Name: "Item 1", Cost: 10.99, ItemType: "Type1", Inventory: 100,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Purchasable Not Found",
			purchasableID:  "nonexistent",
			mockPurchasable: nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Service Error",
			purchasableID:  "error123",
			mockPurchasable: nil,
			mockError:      errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &rds_service.MockPurchasableService{
				GetPurchasableByIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error) {
					return tt.mockPurchasable, tt.mockError
				},
			}
			handler := NewPurchasableHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/purchasables/"+tt.purchasableID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.purchasableID})
			w := httptest.NewRecorder()

			handler.GetPurchasable(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedStatus == http.StatusOK {
				var purchasable internal_types.Purchasable
				err := json.NewDecoder(res.Body).Decode(&purchasable)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}
				if purchasable.ID != tt.mockPurchasable.ID {
					t.Errorf("Expected Purchasable ID %s, got %s", tt.mockPurchasable.ID, purchasable.ID)
				}
			}
		})
	}
}

func TestCreatePurchasable(t *testing.T) {
	tests := []struct {
		name           string
		inputPurchasable internal_types.PurchasableInsert
		mockPurchasable *internal_types.Purchasable
		mockError      error
		expectedStatus int
	}{
		{
			name: "Invalid Input",
			inputPurchasable: internal_types.PurchasableInsert{
				Name: "", Cost: -1, ItemType: "Invalid",
			},
			mockPurchasable: nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &rds_service.MockPurchasableService{
				InsertPurchasableFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.PurchasableInsert) (*internal_types.Purchasable, error) {
					return tt.mockPurchasable, tt.mockError
				},
			}
			handler := NewPurchasableHandler(mockService)

			inputJSON, _ := json.Marshal(tt.inputPurchasable)
			req := httptest.NewRequest(http.MethodPost, "/purchasables", strings.NewReader(string(inputJSON)))
			w := httptest.NewRecorder()

			handler.CreatePurchasable(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedStatus == http.StatusCreated {
				var purchasable internal_types.Purchasable
				err := json.NewDecoder(res.Body).Decode(&purchasable)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}
				if purchasable.ID != tt.mockPurchasable.ID {
					t.Errorf("Expected Purchasable ID %s, got %s", tt.mockPurchasable.ID, purchasable.ID)
				}
			}
		})
	}
}

func TestUpdatePurchasable(t *testing.T) {
	tests := []struct {
		name           string
		purchasableID  string
		inputUpdate    internal_types.PurchasableUpdate
		mockPurchasable *internal_types.Purchasable
		mockError      error
		expectedStatus int
	}{
		{
			name:          "Success",
			purchasableID: "1",
			inputUpdate: internal_types.PurchasableUpdate{
				Name: "Updated Item", Cost: 25.99, ItemType: "UpdatedType", Inventory: 75,
			},
			mockPurchasable: &internal_types.Purchasable{
				ID: "1", Name: "Updated Item", Cost: 25.99, ItemType: "UpdatedType", Inventory: 75,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Purchasable Not Found",
			purchasableID: "nonexistent",
			inputUpdate: internal_types.PurchasableUpdate{
				Name: "Nonexistent Item", Cost: 99.99, ItemType: "NonexistentType", Inventory: 0,
			},
			mockPurchasable: nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:          "Service Error",
			purchasableID: "error123",
			inputUpdate: internal_types.PurchasableUpdate{
				Name: "Error Item", Cost: 29.99, ItemType: "ErrorType", Inventory: 20,
			},
			mockPurchasable: nil,
			mockError:      errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &rds_service.MockPurchasableService{
				UpdatePurchasableFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {
					return tt.mockPurchasable, tt.mockError
				},
			}
			handler := NewPurchasableHandler(mockService)

			inputJSON, _ := json.Marshal(tt.inputUpdate)
			req := httptest.NewRequest(http.MethodPut, "/purchasables/"+tt.purchasableID, strings.NewReader(string(inputJSON)))
			req = mux.SetURLVars(req, map[string]string{"id": tt.purchasableID})
			w := httptest.NewRecorder()

			handler.UpdatePurchasable(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedStatus == http.StatusOK {
				var purchasable internal_types.Purchasable
				err := json.NewDecoder(res.Body).Decode(&purchasable)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}
				if purchasable.ID != tt.mockPurchasable.ID {
					t.Errorf("Expected Purchasable ID %s, got %s", tt.mockPurchasable.ID, purchasable.ID)
				}
			}
		})
	}
}

func TestDeletePurchasable(t *testing.T) {
	tests := []struct {
		name           string
		purchasableID  string
		mockError      error
		expectedStatus int
	}{
		{
			name:          "Service Error",
			purchasableID: "error123",
			mockError:      errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &rds_service.MockPurchasableService{
				DeletePurchasableFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error {
					return tt.mockError
				},
			}
			handler := NewPurchasableHandler(mockService)

			req := httptest.NewRequest(http.MethodDelete, "/purchasables/"+tt.purchasableID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.purchasableID})
			w := httptest.NewRecorder()

			handler.DeletePurchasable(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, res.StatusCode)
			}
		})
	}
}

