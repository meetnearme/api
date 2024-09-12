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

func buildSqlPurchasableParams(parameters map[string]interface{}) ([]rds_types.SqlParameter, error) {
	var params []rds_types.SqlParameter

	log.Printf("parameters purchasables: %v", parameters["description"])
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

	// Might need the reference
	// UserID (UUID)
	// Check and add the user_id parameter if present and valid
	if userIdValue, ok := parameters["user_id"].(string); ok && userIdValue != "" {
		userID := rds_types.SqlParameter{
			Name:     aws.String("user_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: userIdValue,
			},
		}
		params = append(params, userID)
	}

	// Name
	if nameValue, ok := parameters["name"].(string); ok && nameValue != "" {
		name := rds_types.SqlParameter{
			Name: aws.String("name"),
			Value: &rds_types.FieldMemberStringValue{
				Value: nameValue,
			},
		}
		params = append(params, name)
	}

	// Cost
	costValue, ok := parameters["cost"].(float64)
	if !ok {
		return nil, fmt.Errorf("cost is not a valid string")
	}

	cost := rds_types.SqlParameter{
		Name: aws.String("cost"),
		Value: &rds_types.FieldMemberDoubleValue{
			Value: costValue,
		},
	}
	params = append(params, cost)

	// Donation Ratio
	donationRatioValue, ok := parameters["donation_ratio"].(float64)
	if !ok {
		return nil, fmt.Errorf("donation ratio is not a valid string")
	}

	donationRatio := rds_types.SqlParameter{
		Name: aws.String("donation_ratio"),
		Value: &rds_types.FieldMemberDoubleValue{
			Value: donationRatioValue,
		},
	}
	params = append(params, donationRatio)

	// Currency
	currencyValue, ok := parameters["currency"].(string)
	if !ok {
		return nil, fmt.Errorf("currency is not a valid string")
	}
	currency := rds_types.SqlParameter{
		Name: aws.String("currency"),
		Value: &rds_types.FieldMemberStringValue{
			Value: currencyValue,
		},
	}
	params = append(params, currency)

	// PurchasableType
	itemTypeValue, ok := parameters["item_type"].(string)
	if !ok {
		return nil, fmt.Errorf("item type is not a valid string")
	}
	itemType := rds_types.SqlParameter{
		Name: aws.String("item_type"),
		Value: &rds_types.FieldMemberStringValue{
			Value: itemTypeValue,
		},
	}
	params = append(params, itemType)

	// Charge Recurrence Interval
	chargeRecurrenceIntervalValue, ok := parameters["charge_recurrence_interval"].(string)
	if !ok {
		return nil, fmt.Errorf("charge recurrence interval is not a valid string")
	}
	chargeRecurrenceInterval := rds_types.SqlParameter{
		Name: aws.String("charge_recurrence_interval"),
		Value: &rds_types.FieldMemberStringValue{
			Value: chargeRecurrenceIntervalValue,
		},
	}
	params = append(params, chargeRecurrenceInterval)

	// Inventory
	inventoryValue, ok := parameters["inventory"].(int64)
	log.Printf("value of inventory: %v", inventoryValue)
	if !ok {
		return nil, fmt.Errorf("inventory is not a valid integer")
	}

	inventory := rds_types.SqlParameter{
		Name: aws.String("inventory"),
		Value: &rds_types.FieldMemberLongValue{
			Value: inventoryValue,
		},
	}
	params = append(params, inventory)


	log.Printf("charge count: %v", parameters["charge_recurrence_interval_count"])


	// Charge Recurrence Interval Count
	chargeRecurrenceIntervalCountValue, ok := parameters["charge_recurrence_interval_count"].(int64)
	if !ok {
		log.Println(" charge recurrence interval count is not an int 64")
	}

	chargeRecurrenceIntervalCount := rds_types.SqlParameter{
		Name: aws.String("charge_recurrence_interval_count"),
		Value: &rds_types.FieldMemberLongValue{
			Value: chargeRecurrenceIntervalCountValue,
		},
	}
	params = append(params, chargeRecurrenceIntervalCount)

	if chargeRecurrenceEndDateValue, ok := parameters["charge_recurrence_end_date"].(string); ok && chargeRecurrenceEndDateValue != "" {
		parsedTime, err := time.Parse(time.RFC3339, chargeRecurrenceEndDateValue)
        if err != nil {
            // Handle parsing error
            log.Printf("Error parsing time for recurrence end date: %v", err)
        }

        // Format the timestamp to required format
        formattedTime := parsedTime.Format("2006-01-02 15:04:05")

        // Create the RDS parameter
        chargeRecurrenceEndDate := rds_types.SqlParameter{
            Name:     aws.String("charge_recurrence_end_date"),
            TypeHint: "TIMESTAMP",
            Value: &rds_types.FieldMemberStringValue{
                Value: formattedTime,
            },
        }
        params = append(params, chargeRecurrenceEndDate)
	}

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

	if descriptionValue, ok := parameters["description"].(string); ok && descriptionValue != "" {
		updatedAt := rds_types.SqlParameter{
			Name:     aws.String("description"),
			Value: &rds_types.FieldMemberStringValue{
				Value: descriptionValue,
			},
		}
		params = append(params, updatedAt)
	}


	return params, nil
}

func extractAndMapSinglePurchasableFromJSON(formattedRecords string) (*internal_types.Purchasable, error) {
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


	purchasable := internal_types.Purchasable{
		ID:                           getString(record, "id"),
		UserID:                       getString(record, "user_id"),
		Name:                         getString(record, "name"),
		ItemType:                     getString(record, "item_type"),
		Cost:                         getFloat64(record, "cost"),
		Currency:                     getString(record, "currency"),
		ChargeRecurrenceInterval:     getString(record, "charge_recurrence_interval"),
		ChargeRecurrenceIntervalCount: int64(record["charge_recurrence_interval_count"].(float64)),
		ChargeRecurrenceEndDate:      getTime(record, "charge_recurrence_end_date"),
		DonationRatio:                getFloat64(record, "donation_ratio"),
		CreatedAt:                    getTime(record, "created_at"),
		UpdatedAt:                    getTime(record, "updated_at"),
	}

	// Handle the optional inventory field
	if inventory, ok := record["inventory"]; ok && inventory != nil {
		purchasable.Inventory = int64(inventory.(float64))
	}

	log.Printf("Purchasable item from extractions: %v", purchasable)

	return &purchasable, nil
}

func extractPurchasablesFromJson(formattedRecords string) ([]internal_types.Purchasable, error) {
	var purchasables []internal_types.Purchasable

	// Parse formattedRecords as a JSON array
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling formatted records: %w", err)
	}
	for _, record := range records {
		var purchasable internal_types.Purchasable

		// Map fields from record to purchasable struct
		purchasable.ID = getString(record, "id")
		purchasable.UserID = getString(record, "user_id")
		purchasable.Name = getString(record, "name")
		purchasable.Cost = getFloat64(record, "cost")
		purchasable.Currency = getString(record, "currency")
		purchasable.ItemType = getString(record, "item_type")
		purchasable.ChargeRecurrenceInterval = getString(record, "charge_recurrence_interval")
		purchasable.ChargeRecurrenceIntervalCount = int64(record["charge_recurrence_interval_count"].(float64))
		purchasable.ChargeRecurrenceEndDate = getTime(record, "charge_recurrence_end_date")
		purchasable.DonationRatio = getFloat64(record, "donation_ratio")
		purchasable.CreatedAt = getTime(record, "createdAt")
		purchasable.UpdatedAt = getTime(record, "updatedAt")

		// Handle the optional inventory field
		if inventory, ok := record["inventory"]; ok && inventory != nil {
			purchasable.Inventory = int64(inventory.(float64))
		}

		purchasables = append(purchasables, purchasable)
	}

	return purchasables, nil
}


func buildUpdatePurchasablesQuery(params map[string]interface{}) (string, map[string]interface{}) {
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
        UPDATE purchasables
        SET %s,
            updated_at = now()
        WHERE id = :id
        RETURNING id, user_id, name, item_type, inventory, cost, currency, charge_recurrence_interval, charge_recurrence_interval_count, charge_recurrence_end_date, created_at, updated_at`,
        strings.Join(setClauses, ", "))

    // Ensure 'id' is always included in the parameters
    if _, ok := sqlParams["id"]; !ok {
        return "", nil // or return an error if `id` is a required field
    }

    return query, sqlParams
}

