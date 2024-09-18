package rds_handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetPurchasable(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{
		GetPurchasableByIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error) {
			createdAt, _ := time.Parse(time.RFC3339, "2023-09-01T00:00:00Z")
			updatedAt, _ := time.Parse(time.RFC3339, "2023-09-01T00:00:00Z")
			return &internal_types.Purchasable{
				ID:        id,
				Name:      "Test Purchasable",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}, nil
		},
	}
	handler := NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/purchasables/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handler.GetPurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, res.StatusCode)
	}

	var response internal_types.Purchasable
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		t.Errorf("failed to decode response body: %v", err)
	}

	if response.ID != "1" {
		t.Errorf("expected ID to be '1', got %v", response.ID)
	}
	if response.Name != "Test Purchasable" {
		t.Errorf("expected Name to be 'Test Purchasable', got %v", response.Name)
	}
}

