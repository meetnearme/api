package services

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
)

func TestExtractClaimsMeta(t *testing.T) {
	// Save original projectID and restore after test
	originalProjectID := projectID
	projectID = &[]string{"test-project"}[0]
	defer func() { projectID = originalProjectID }()

	tests := []struct {
		name          string
		claims        map[string]interface{}
		expectedRoles []constants.RoleClaim
		expectedMeta  map[string]interface{}
	}{
		{
			name: "successful extraction of roles and metadata",
			claims: map[string]interface{}{
				constants.AUTH_METADATA_KEY: map[string]interface{}{
					"name": "John Doe",
					"age":  30,
				},
				"urn:zitadel:iam:org:project:test-project:roles": map[string]interface{}{
					"admin": map[string]interface{}{
						"proj-123": "Project Alpha",
					},
					"user": map[string]interface{}{
						"proj-456": "Project Beta",
					},
				},
			},
			expectedRoles: []constants.RoleClaim{
				{Role: "admin", ProjectID: "proj-123", ProjectName: "Project Alpha"},
				{Role: "user", ProjectID: "proj-456", ProjectName: "Project Beta"},
			},
			expectedMeta: map[string]interface{}{
				"name": "John Doe",
				"age":  30,
			},
		},
		{
			name:          "empty claims",
			claims:        map[string]interface{}{},
			expectedRoles: []constants.RoleClaim{},
			expectedMeta:  nil,
		},
		{
			name: "malformed metadata",
			claims: map[string]interface{}{
				constants.AUTH_METADATA_KEY: "invalid",
			},
			expectedRoles: []constants.RoleClaim{},
			expectedMeta:  nil,
		},
		{
			name: "malformed roles",
			claims: map[string]interface{}{
				"urn:zitadel:iam:org:project:test-project:roles": "invalid",
			},
			expectedRoles: []constants.RoleClaim{},
			expectedMeta:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, meta := ExtractClaimsMeta(tt.claims)

			// Compare lengths first
			if len(roles) != len(tt.expectedRoles) {
				t.Errorf("got %d roles, want %d roles", len(roles), len(tt.expectedRoles))
				return
			}

			// Compare roles (order may vary due to map iteration)
			roleMap := make(map[string]bool)
			for _, role := range roles {
				key := fmt.Sprintf("%s-%s-%s", role.Role, role.ProjectID, role.ProjectName)
				roleMap[key] = true
			}

			for _, expectedRole := range tt.expectedRoles {
				key := fmt.Sprintf("%s-%s-%s", expectedRole.Role, expectedRole.ProjectID, expectedRole.ProjectName)
				if !roleMap[key] {
					t.Errorf("missing expected role: %v", expectedRole)
				}
			}

			// Compare metadata
			if !reflect.DeepEqual(meta, tt.expectedMeta) {
				t.Errorf("metadata = %v, want %v", meta, tt.expectedMeta)
			}
		})
	}
}
