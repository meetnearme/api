package rds_service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

func buildSqlUserParams(parameters map[string]interface{}) ([]rds_types.SqlParameter, error) {
	var params []rds_types.SqlParameter

	// ID (UUID)
	idValue, ok := parameters["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is not a valid string")
	}
	id := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	params = append(params, id)

	// Name
	nameValue, ok := parameters["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is not a valid string")
	}
	name := rds_types.SqlParameter{
		Name: aws.String("name"),
		Value: &rds_types.FieldMemberStringValue{
			Value: nameValue,
		},
	}
	params = append(params, name)

	// Email
	emailValue, ok := parameters["email"].(string)
	if !ok {
		return nil, fmt.Errorf("email is not a valid string")
	}
	email := rds_types.SqlParameter{
		Name: aws.String("email"),
		Value: &rds_types.FieldMemberStringValue{
			Value: emailValue,
		},
	}
	params = append(params, email)

	// Role
	roleValue, ok := parameters["role"].(string)
	if !ok {
		return nil, fmt.Errorf("role is not a valid string")
	}
	role := rds_types.SqlParameter{
		Name: aws.String("role"),
		Value: &rds_types.FieldMemberStringValue{
			Value: roleValue,
		},
	}
	params = append(params, role)

	// Convert string timestamps to SQL parameters if they are provided
	if createdAtValue, ok := parameters["created_at"].(string); ok && createdAtValue != "" {
		createdAt := rds_types.SqlParameter{
			Name:     aws.String("created_at"),
			TypeHint: "TIMESTAMP",
			Value: &rds_types.FieldMemberStringValue{
				Value: createdAtValue,
			},
		}
		params = append(params, createdAt)
	}

	if updatedAtValue, ok := parameters["updated_at"].(string); ok && updatedAtValue != "" {
		updatedAt := rds_types.SqlParameter{
			Name:     aws.String("updated_at"),
			TypeHint: "TIMESTAMP",
			Value: &rds_types.FieldMemberStringValue{
				Value: updatedAtValue,
			},
		}
		params = append(params, updatedAt)
	}

	// Optional Fields (address_street, address_city, address_zip_code, address_country, phone, profile_picture_url)
	addressFields := []string{"address", "phone", "profile_picture_url"}

	for _, field := range addressFields {
		value, ok := parameters[field].(string)
		if !ok {
			value = "" // Default to an empty string if not provided
		}
		param := rds_types.SqlParameter{
			Name: aws.String(field),
			Value: &rds_types.FieldMemberStringValue{
				Value: value,
			},
		}
		params = append(params, param)
	}


	return params, nil
}

func extractAndMapSingleUserFromJSON(formattedRecords string) (*internal_types.User, error) {
	log.Printf("formatted records from JSON: %v", formattedRecords)
    var records []map[string]interface{}
    if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
        return nil, fmt.Errorf("error unmarshaling JSON records: %v", err)
    }

    // Example assuming only one record for simplicity
    if len(records) == 0 {
        return nil, fmt.Errorf("no records found")
    }

    record := records[0]

    user := internal_types.User{
        ID:                  getString(record, "id"),
        Name:                getString(record, "name"),
        Email:               getString(record, "email"),
        Address:       getString(record, "address"),
        Phone:               getString(record, "phone"),
        ProfilePictureURL:   getString(record, "profile_picture_url"),
        Role:                getString(record, "role"),
        CreatedAt:           getTime(record, "created_at"),
        UpdatedAt:           getTime(record, "updated_at"),
    }

    return &user, nil
}


// Extract users from formatted JSON records without column metadata
func extractUsersFromJson(formattedRecords string) ([]internal_types.User, error) {
    var users []internal_types.User

    // Parse formattedRecords as a JSON array
    var records []map[string]interface{}
    if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
        return nil, fmt.Errorf("error unmarshaling formatted records: %w", err)
    }

    // Iterate over each record and extract user information
    for _, record := range records {
        var user internal_types.User

        // Map fields from record to user struct
        if id, ok := record["id"].(string); ok {
            user.ID = id
        }

        if name, ok := record["name"].(string); ok {
            user.Name = name
        }

        if email, ok := record["email"].(string); ok {
            user.Email = email
        }

        if addressStreet, ok := record["address"].(string); ok {
            user.Address = addressStreet
        }

        if phone, ok := record["phone"].(string); ok {
            user.Phone = phone
        }

        if profilePictureURL, ok := record["profile_picture_url"].(string); ok {
            user.ProfilePictureURL = profilePictureURL
        }

        if role, ok := record["role"].(string); ok {
            user.Role = role
        }


        if createdAtStr, ok := record["created_at"].(string); ok {
            createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
            if err == nil {
                user.CreatedAt = createdAt
            }
        }

        if updatedAtStr, ok := record["updated_at"].(string); ok {
            updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
            if err == nil {
                user.UpdatedAt = updatedAt
            }
        }

        users = append(users, user)
    }

    return users, nil
}
