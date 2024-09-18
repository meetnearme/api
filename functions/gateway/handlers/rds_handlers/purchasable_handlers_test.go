package rds_handlers

import (
	"context"
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

func TestDeletePurchasable(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{
		DeletePurchasableFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error {
			if id == "1" {
				return nil
			}
			return fmt.Errorf("purchasable not found")
		},
	}
	handler := NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/purchasables/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handler.DeletePurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "Purchasable successfully deleted" {
		t.Errorf("expected body to be 'Purchasable successfully deleted', got %v", string(body))
	}
}

func TestCreatePurchasable_InvalidJSON(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{}
	handler := NewPurchasableHandler(mockService)

	payload := `{"name": "New Purchasable"`
	req := httptest.NewRequest(http.MethodPost, "/purchasables", strings.NewReader(payload))
	w := httptest.NewRecorder()

	handler.CreatePurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected status %v, got %v", http.StatusUnprocessableEntity, res.StatusCode)
	}
}

func TestUpdatePurchasable_InvalidID(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{}
	handler := NewPurchasableHandler(mockService)

	payload := `{"name": "Updated Purchasable"}`
	req := httptest.NewRequest(http.MethodPut, "/purchasables/invalid", strings.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handler.UpdatePurchasable(w, req)

	res := w.Result()
	// Expected status should be 404 instead of 400

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, res.StatusCode)
	}
}

func TestGetPurchasable_NotFound(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{
		GetPurchasableByIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error) {
			return nil, fmt.Errorf("purchasable not found")
		},
	}
	handler := NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/purchasables/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handler.GetPurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %v, got %v", http.StatusInternalServerError, res.StatusCode)
	}
}

func TestGetPurchasablesByUserID_InvalidID(t *testing.T) {
	mockService := &rds_service.MockPurchasableService{}
	handler := NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/purchasables/user/", nil)
	req = mux.SetURLVars(req, map[string]string{"user_id": ""})
	w := httptest.NewRecorder()

	handler.GetPurchasablesByUserID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, res.StatusCode)
	}
}

