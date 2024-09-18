package rds_service

import (
	"context"
	"fmt"
	"log"

	"github.com/go-playground/validator"
	"github.com/google/uuid"

	// "github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

// Validator instance for struct validation
var validate *validator.Validate = validator.New()

// UserServiceInterface defines the methods required for the user service.
type UserServiceInterface interface {
	InsertUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error)
	GetUserByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error)
	UpdateUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.UserUpdate) (*internal_types.User, error)
	DeleteUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error
	GetUsers(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.User, error) // New method
}

// UserService is the concrete implementation of the UserServiceInterface.
type UserService struct{}

func NewUserService() UserServiceInterface {
	return &UserService{}
}

func (s *UserService) InsertUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error) {
    // Generate a new UUID if not provided
    if user.ID == "" {
        user.ID = uuid.New().String()
    }

    // Validate the user object
    if err := validate.Struct(user); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Construct the SQL query
    query := `
        INSERT INTO users (
            id, name, email, address, phone, profile_picture_url, role,
            created_at, updated_at
        )
        VALUES (
            :id, :name, :email, :address, :phone, :profile_picture_url, :role,
            NOW(), NOW()
        )
        RETURNING id, name, email, address, phone, profile_picture_url, role,
                  created_at, updated_at
    `

    // Prepare the parameters for the query
    params := map[string]interface{}{
        "id":                  user.ID,
        "name":                user.Name,
        "email":               user.Email,
        "address":      user.Address,
        "phone":               user.Phone,
        "profile_picture_url": user.ProfilePictureURL,
        "role":                user.Role,
        "created_at":          user.CreatedAt,
        "updated_at":          user.UpdatedAt,
    }


	paramsRdsFormat, err := buildSqlUserParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the statement
    result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to insert user: %w", err)
    }

    // Extract the inserted user data
    insertedUser, err := extractAndMapSingleUserFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map user after insert: %w", err)
    }

    // return user, nil
	return insertedUser, nil
}


func (s *UserService) GetUserByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error) {
	query := "SELECT * FROM users WHERE id = :id"
	params := map[string]interface{}{
		"id": id,
	}

	var paramsRdsFormat []rds_types.SqlParameter

	// ID (UUID)
	idValue, ok := params["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is not a valid string")
	}
	idRds := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	paramsRdsFormat = append(paramsRdsFormat, idRds)

	result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	log.Printf("Result in getby id: %v", result)
	log.Printf("Result formatted result: %v", result.FormattedRecords)

    // Extract the inserted user data
    user, err := extractAndMapSingleUserFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map user after insert: %w", err)
    }

    // return user, nil
	return user, nil
}

func (s *UserService) GetUsers(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.User, error) {
    // Updated query to limit to 10 records
    query := "SELECT id, name, email, address, phone, profile_picture_url, role, created_at, updated_at FROM users LIMIT 10"
    var params []rds_types.SqlParameter

    // Execute the SQL query
    result, err := rdsClient.ExecStatement(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("failed to get users: %w", err)
    }

    var users []internal_types.User

    // Check if formattedRecords is available
    if result.FormattedRecords != nil {
        users, err = extractUsersFromJson(*result.FormattedRecords)
        if err != nil {
            return nil, fmt.Errorf("error extracting users from JSON: %w", err)
        }
    } else {
        return nil, fmt.Errorf("no formatted records found")
    }

    return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.UserUpdate) (*internal_types.User, error) {
    // Build the SQL query to update user information
    query := `
        UPDATE users
        SET
            name = :name,
            email = :email,
            address = :address,
            phone = :phone,
            profile_picture_url = :profile_picture_url,
            role = :role,
            organization_user_id = :organization_user_id,
            updated_at = now()
        WHERE id = :id
        RETURNING id, name, email, address, phone, profile_picture_url, role, organization_user_id, created_at, updated_at`

    // Build SQL parameters from UserUpdate struct
    params := map[string]interface{}{
        "id":                    id,
        "name":                  user.Name,
        "email":                 user.Email,
        "address":        user.Address,
        "phone":                 user.Phone,
        "profile_picture_url":   user.ProfilePictureURL,
        "role":                  user.Role,
    }

    // Convert parameters to RDS types
	rdsParams, err := buildSqlUserParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the SQL statement
    result, err := rdsClient.ExecStatement(ctx, query, rdsParams)
    if err != nil {
        return nil, fmt.Errorf("failed to update user: %w", err)
    }

    // Extract the updated user from the formattedRecords JSON
    updatedUser, err := extractAndMapSingleUserFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to extract updated user: %w", err)
    }

    // Since we are expecting one user in the result
    return updatedUser, nil
}

func (s *UserService) DeleteUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
 	query := "DELETE FROM users WHERE id = :id"
	params := map[string]interface{}{
		"id": id,
	}

	var paramsRdsFormat []rds_types.SqlParameter

	// ID (UUID)
	idValue, ok := params["id"].(string)
	if !ok {
		return fmt.Errorf("id is not a valid string")
	}
	idRds := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	paramsRdsFormat = append(paramsRdsFormat, idRds)

	result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
	log.Printf("Err from exec delete: %v", err)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	// Check if any rows were affected by the delete operation
	if result.NumberOfRecordsUpdated == 0 {
		// No rows were affected, meaning the user was not found
		return fmt.Errorf("user not found")
	}

	// No need to return user info; just return nil to indicate success.
	return nil
}

type MockUserService struct {
	InsertUserFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error)
	GetUserByIDFunc func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error)
	UpdateUserFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.UserUpdate) (*internal_types.User, error)
	DeleteUserFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error
	GetUsersFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.User, error) // New function
}

func (m *MockUserService) InsertUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.UserInsert) (*internal_types.User, error) {
	return m.InsertUserFunc(ctx, rdsClient, user)
}

func (m *MockUserService) GetUserByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.User, error) {
	return m.GetUserByIDFunc(ctx, rdsClient, id)
}

func (m *MockUserService) UpdateUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.UserUpdate) (*internal_types.User, error) {
	return m.UpdateUserFunc(ctx, rdsClient, id, user)
}

func (m *MockUserService) DeleteUser(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
	return m.DeleteUserFunc(ctx, rdsClient, id)
}

func (m *MockUserService) GetUsers(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.User, error) {
	return m.GetUsersFunc(ctx, rdsClient)
}

