package rds_service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func buildSqlRegistrationFieldsParams(parameters map[string]interface{}) ([]rds_types.SqlParameter, error) {
	var params []rds_types.SqlParameter

	log.Printf("parameters event rsvps: %v", parameters["description"])
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

	// TODO: do we need check in enum before DB?
	// Status
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

	typeValue, ok := parameters["type"].(string)
	if !ok {
		return nil, fmt.Errorf("type is not a valid string")
	}
	typeField := rds_types.SqlParameter{
		Name: aws.String("type"),
		Value: &rds_types.FieldMemberStringValue{
			Value: typeValue,
		},
	}
	params = append(params, typeField)

	optionsValue, ok := parameters["options"].(string)
	if !ok {
		return nil, fmt.Errorf("options is not a valid string")
	}
	options := rds_types.SqlParameter{
		Name: aws.String("options"),
		Value: &rds_types.FieldMemberStringValue{
			Value: optionsValue,
		},
	}
	params = append(params, options)

	defaultValue, ok := parameters["default_val"].(string)
	if !ok {
		return nil, fmt.Errorf("default is not a valid string")
	}
	defaultField := rds_types.SqlParameter{
		Name: aws.String("default_val"),
		Value: &rds_types.FieldMemberStringValue{
			Value: defaultValue,
		},
	}
	params = append(params, defaultField)

	placeholderValue, ok := parameters["placeholder"].(string)
	if !ok {
		return nil, fmt.Errorf("placeholder is not a valid string")
	}
	placeholder := rds_types.SqlParameter{
		Name: aws.String("placeholder"),
		Value: &rds_types.FieldMemberStringValue{
			Value: placeholderValue,
		},
	}
	params = append(params, placeholder)

	descriptionValue, ok := parameters["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description is not a valid string")
	}
	description := rds_types.SqlParameter{
		Name: aws.String("description"),
		Value: &rds_types.FieldMemberStringValue{
			Value: descriptionValue,
		},
	}
	params = append(params, description)

	requiredValue, ok := parameters["required"].(bool)
	if !ok {
		return nil, fmt.Errorf("required is not a valid boolean")
	}
	required := rds_types.SqlParameter{
		Name: aws.String("required"),
		Value: &rds_types.FieldMemberBooleanValue{
			Value: requiredValue,
		},
	}
	params = append(params, required)

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

	return params, nil
}

func extractAndMapSingleRegistrationFieldsFromJSON(formattedRecords string) (*internal_types.RegistrationFields, error) {
	log.Printf("formatted records from JSON: %v", formattedRecords)
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON records: %v", err)
	}

	// Example assuming only one record for simplicity
	if len(records) == 0 {
		return nil, fmt.Errorf("no records found")
	}

	log.Printf("recordSSSSSSSS: %v", records)

	record := records[0]


	registrationFields := internal_types.RegistrationFields{
		ID:                           getString(record, "id"),
		Name:                           getString(record, "name"),
		Type:                           getString(record, "type"),
		Options:                           getString(record, "options"),
		Default:                           getString(record, "default_val"),
		Placeholder:                           getString(record, "placeholder"),
		Description:                           getString(record, "description"),
		Required:                           record["required"].(bool),
		CreatedAt:                    getTime(record, "created_at"),
		UpdatedAt:                    getTime(record, "updated_at"),
	}


	log.Printf("RegistrationFields item from extractions: %v", registrationFields)

	return &registrationFields, nil
}

func extractRegistrationFieldssFromJson(formattedRecords string) ([]internal_types.RegistrationFields, error) {
	var registrationFieldsSlice []internal_types.RegistrationFields

	// Parse formattedRecords as a JSON array
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling formatted records: %w", err)
	}
	for _, record := range records {
		var registrationFields internal_types.RegistrationFields

		// Map fields from record to purchasable struct
		registrationFields.ID = getString(record, "id")
		registrationFields.Name = getString(record, "name")
		registrationFields.Type = getString(record, "type")
		registrationFields.Options = getString(record, "options")
		registrationFields.Default = getString(record, "default")
		registrationFields.Placeholder = getString(record, "placeholder")
		registrationFields.Description = getString(record, "description")
		registrationFields.Required =	record["required"].(bool)
		registrationFields.CreatedAt = getTime(record, "created_at")
		registrationFields.UpdatedAt = getTime(record, "updated_at")

		registrationFieldsSlice = append(registrationFieldsSlice, registrationFields)
	}

	return registrationFieldsSlice, nil
}


func buildUpdateRegistrationFieldsQuery(params map[string]interface{}) (string, map[string]interface{}) {
    // Initialize the SQL query parts
    setClauses := []string{}
    sqlParams := map[string]interface{}{}

    // Iterate through the params map
    for key, value := range params {
        if value != nil {
            // Build the SET clause dynamically
            setClauses = append(setClauses, fmt.Sprintf("%s = :%s", key, key))
            sqlParams[key] = value
        }
    }

    // If no fields are provided, return an error or an empty query
    if len(setClauses) == 0 {
        return "", nil
    }

    // Construct the full SQL query
    query := fmt.Sprintf(`
        UPDATE registration_fields
        SET %s,
            updated_at = now()
        WHERE id = :id
        RETURNING id, name, type, options, default_val, placeholder, description, required,
			created_at, updated_at`,
        strings.Join(setClauses, ", "))

    // Ensure 'id' is always included in the parameters
    if _, ok := sqlParams["id"]; !ok {
        return "", nil // or return an error if `id` is a required field
    }

    return query, sqlParams
}



