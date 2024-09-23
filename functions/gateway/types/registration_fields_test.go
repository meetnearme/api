package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestRegistrationFieldsInsert tests the RegistrationFieldsInsert struct and JSON marshaling/unmarshaling
func TestRegistrationFieldsInsert(t *testing.T) {
	now := time.Now()

	// Create an instance of RegistrationFieldsInsert
	registrationFields := RegistrationFieldsInsert{
		EventId: "test-event-id",
		Fields: []RegistrationFieldItemInsert{
			{
				Name:        "attendeeEmail",
				Type:        "text",
				Required:    true,
				Default:     "",
				Placeholder: "email@example.com",
				Description: "We need your updated email in case of any changes",
			},
			{
				Name:        "tshirtSize",
				Type:        "select",
				Required:    true,
				Default:     "large",
				Options:     []string{"small", "medium", "large", "XL"},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
		UpdatedBy: "test-user",
	}

	// Check field values
	if registrationFields.EventId != "test-event-id" {
		t.Errorf("expected EventId to be 'test-event-id', got %s", registrationFields.EventId)
	}

	if len(registrationFields.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(registrationFields.Fields))
	}

	if registrationFields.Fields[0].Name != "attendeeEmail" {
		t.Errorf("expected first field name to be 'attendeeEmail', got %s", registrationFields.Fields[0].Name)
	}

	if !registrationFields.Fields[0].Required {
		t.Error("expected first field to be required")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(registrationFields)
	if err != nil {
		t.Errorf("error marshaling JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled RegistrationFieldsInsert
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("error unmarshaling JSON: %v", err)
	}

	// Check unmarshaled data
	if unmarshaled.EventId != registrationFields.EventId {
		t.Errorf("expected EventId to be '%s', got '%s'", registrationFields.EventId, unmarshaled.EventId)
	}

	if len(unmarshaled.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(unmarshaled.Fields))
	}

	if unmarshaled.Fields[1].Type != registrationFields.Fields[1].Type {
		t.Errorf("expected second field type to be '%s', got '%s'", registrationFields.Fields[1].Type, unmarshaled.Fields[1].Type)
	}

	if len(unmarshaled.Fields[1].Options) != len(registrationFields.Fields[1].Options) {
		t.Errorf("expected second field options length to be %d, got %d", len(registrationFields.Fields[1].Options), len(unmarshaled.Fields[1].Options))
	}
}

// TestRegistrationFieldsServiceInterface tests the RegistrationFieldsServiceInterface methods
func TestRegistrationFieldsServiceInterface(t *testing.T) {
	// Mock implementation could be done here with a mocking library
	// Example usage with a mocking library like testify/mock can be added later
}

