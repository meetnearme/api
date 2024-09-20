package rds_service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

	categoryPreferencesValue, ok := parameters["category_preferences"].(string)
	if !ok {
		return nil, fmt.Errorf("category_preferences is not a valid string")
	}
	categoryPreferences := rds_types.SqlParameter{
		Name: aws.String("category_preferences"),
		Value: &rds_types.FieldMemberStringValue{
			Value: categoryPreferencesValue,
		},
	}
	params = append(params, categoryPreferences)

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
    var records []map[string]interface{}
    if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
        return nil, fmt.Errorf("error unmarshaling JSON records: %v", err)
    }

    // Example assuming only one record for simplicity
    if len(records) == 0 {
        return nil, fmt.Errorf("no records found")
    }

    record := records[0]


	// Extract category_preferences and convert to []string if present
	var categoryPreferences []string
    if categoryPrefStr, ok := record["category_preferences"].(string); ok {
		// cleanedString := strings.ReplaceAll(categoryPrefStr, "\\", "")
        // First, check if it's a JSON array string and try to unmarshal it directly
        err := json.Unmarshal([]byte(categoryPrefStr), &categoryPreferences)
        if err != nil {
            return nil, fmt.Errorf("error unmarshaling category_preferences field: %v", err)
        }
    } else {
        return nil, fmt.Errorf("category_preferences field is not a string")
    }

    user := internal_types.User{
        ID:                  getString(record, "id"),
        Name:                getString(record, "name"),
        Email:               getString(record, "email"),
        Address:       getString(record, "address"),
        Phone:               getString(record, "phone"),
        ProfilePictureURL:   getString(record, "profile_picture_url"),
		CategoryPreferences: categoryPreferences,
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

		if categoryPreferences, ok := record["category_preferences"].(string); ok {
			var categoryPreferencesSlice []string
			err := json.Unmarshal([]byte(record["category_preferences"].(string)), &categoryPreferences)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling category_preferences field")
			}

			user.CategoryPreferences = categoryPreferencesSlice
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

func buildUpdateUserQuery(params map[string]interface{}) (string, map[string]interface{}) {
    // Initialize the SQL query parts
    setClauses := []string{}
    sqlParams := map[string]interface{}{}

    // Iterate through the params map
    for key, value := range params {
        if value != nil && value != "" {
            // Special case for category_preferences: treat as a string (just like in the insertion)
            if key == "category_preferences" {
                categoryPreferencesValue, ok := value.(string)
                if !ok {
                    continue // Skip this field if it's not a valid string
                }
                // Treat as a string, as it should be stored as a JSON-encoded string in the database
                setClauses = append(setClauses, fmt.Sprintf("%s = :%s", key, key))
                sqlParams[key] = categoryPreferencesValue
            } else {
                // Handle all other fields normally
                setClauses = append(setClauses, fmt.Sprintf("%s = :%s", key, key))
                sqlParams[key] = value
            }
        }
    }

    // If no fields are provided, return an error or an empty query
    if len(setClauses) == 0 {
        return "", nil
    }

    // Ensure 'id' is always included in the parameters
    if _, ok := sqlParams["id"]; !ok {
        return "", nil // 'id' is required for the update operation
    }

    // Construct the full SQL query
    query := fmt.Sprintf(`
        UPDATE users
        SET %s,
            updated_at = now()
        WHERE id = :id
        RETURNING id, name, email, address, phone, profile_picture_url, category_preferences, role, created_at, updated_at`,
        strings.Join(setClauses, ", "))

    return query, sqlParams
}

