package rds_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

// Validator instance for struct validation
var validate *validator.Validate = validator.New()

// UserHandler handles user-related requests
type UserHandler struct {
	UserService internal_types.UserServiceInterface
}

// NewUserHandler creates a new UserHandler with the given UserService
func NewUserHandler(userService internal_types.UserServiceInterface) *UserHandler {
	return &UserHandler{UserService: userService}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var createUser internal_types.UserInsert
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &createUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    log.Printf("Create user: %v", createUser)

    err = validate.Struct(&createUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    // Default optional fields to empty strings if not provided
    if createUser.Address == "" {
        createUser.Address = ""
    }
    if createUser.Phone == "" {
        createUser.Phone = ""
    }
    if createUser.ProfilePictureURL == "" {
        createUser.ProfilePictureURL = ""
    }
    if createUser.OrganizationUserID == "" {
        createUser.OrganizationUserID = ""
    }

    now := time.Now().UTC().Format(time.RFC3339)
    createUser.CreatedAt = now
    createUser.UpdatedAt = now

	// Parse timestamps
	createdAtTime, err := time.Parse(time.RFC3339, createUser.CreatedAt)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid created_at timestamp: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedAtTime := createdAtTime // Default to the same value if not provided
	if createUser.UpdatedAt != "" {
		updatedAtTime, err = time.Parse(time.RFC3339, createUser.UpdatedAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid updated_at timestamp: "+err.Error()), http.StatusBadRequest, err)
			return
		}
	}

	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Format timestamps for RDS
	createUser.CreatedAt = createdAtTime.Format(rdsTimeFormat)
	createUser.UpdatedAt = updatedAtTime.Format(rdsTimeFormat)

    db := transport.GetRdsDB()
    res, err := h.UserService.InsertUser(r.Context(), db, createUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new user: %+v", res)
    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.UserService.GetUserByID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("User not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

// This needs to change for use cases of fetching multiple users based on org ID or other
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
    db := transport.GetRdsDB()
    users, err := h.UserService.GetUsers(r.Context(), db)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get users: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(users)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    var updateUser internal_types.UserUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &updateUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    err = validate.Struct(&updateUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.UserService.UpdateUser(r.Context(), db, id, updateUser)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("User not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Vars in delete: %v", vars)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    err := h.UserService.DeleteUser(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("User successfully deleted"), http.StatusOK, nil)
}


func CreateUserHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	userService := rds_service.NewUserService()
	handler := NewUserHandler(userService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateUser(w, r)
	}
}


// GetUserHandler is a wrapper that creates the UserHandler and returns the handler function for getting a user by ID
func GetUserHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	userService := rds_service.NewUserService()
	handler := NewUserHandler(userService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetUser(w, r)
	}
}

// GetUsersHandler is a wrapper that creates the UserHandler and returns the handler function for getting all users
func GetUsersHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	userService := rds_service.NewUserService()
	handler := NewUserHandler(userService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetUsers(w, r)
	}
}

// UpdateUserHandler is a wrapper that creates the UserHandler and returns the handler function for updating a user
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	userService := rds_service.NewUserService()
	handler := NewUserHandler(userService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateUser(w, r)
	}
}

// DeleteUserHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a user
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	userService := rds_service.NewUserService()
	handler := NewUserHandler(userService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteUser(w, r)
	}
}
