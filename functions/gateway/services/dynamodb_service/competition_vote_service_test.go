package dynamodb_service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func init() {
	votesTableName = "test-votes-table"
}

func TestPutCompetitionVote(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(t *testing.T) *test_helpers.MockDynamoDBClient
		vote        internal_types.CompetitionVoteUpdate
		want        *dynamodb.PutItemOutput
		wantErr     bool
		errContains string
	}{
		// {
		// 	name: "successful vote",
		// 	vote: internal_types.CompetitionVoteUpdate{
		// 		CompositePartitionKey: "test-competition-round",
		// 		UserId:                "test-user",
		// 		VoteRecipientId:       "recipient-123",
		// 		VoteValue:             5,
		// 		ExpiresOn:             time.Now().Add(24 * time.Hour).Unix(),
		// 	},
		// 	setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
		// 		mockDB := &test_helpers.MockDynamoDBClient{
		// 			PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		// 				if params == nil {
		// 					return nil, errors.New("nil params")
		// 				}
		// 				if *params.TableName != votesTableName {
		// 					t.Errorf("expected table name %s, got %s", votesTableName, *params.TableName)
		// 				}
		// 				return &dynamodb.PutItemOutput{}, nil
		// 			},
		// 		}
		// 		return mockDB
		// 	},
		// 	want:    &dynamodb.PutItemOutput{},
		// 	wantErr: false,
		// },
		{
			name: "validation error - missing required fields",
			vote: internal_types.CompetitionVoteUpdate{
				CompositePartitionKey: "test-competition-round",
				UserId:                "test-user",
				// Missing VoteRecipientId which is required
				VoteValue: 5,
			},
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				return &test_helpers.MockDynamoDBClient{}
			},
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "validation error - missing vote value",
			vote: internal_types.CompetitionVoteUpdate{
				CompositePartitionKey: "test-competition-round",
				UserId:                "test-user",
				VoteRecipientId:       "recipient-123",
				// Missing VoteValue which is required
			},
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				return &test_helpers.MockDynamoDBClient{}
			},
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "empty table name",
			vote: internal_types.CompetitionVoteUpdate{
				CompositePartitionKey: "test-competition-round",
				UserId:                "test-user",
				VoteRecipientId:       "recipient-123",
				VoteValue:             5,
			},
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				originalTableName := votesTableName
				votesTableName = ""
				t.Cleanup(func() {
					votesTableName = originalTableName
				})
				return &test_helpers.MockDynamoDBClient{}
			},
			wantErr:     true,
			errContains: "votesTableName is empty",
		},
		{
			name: "dynamodb error",
			vote: internal_types.CompetitionVoteUpdate{
				CompositePartitionKey: "test-competition-round",
				UserId:                "test-user",
				VoteRecipientId:       "recipient-123",
				VoteValue:             5,
				ExpiresOn:             time.Now().Add(24 * time.Hour).Unix(),
			},
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				return &test_helpers.MockDynamoDBClient{
					PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
						return nil, errors.New("dynamodb error")
					},
				}
			},
			wantErr:     true,
			errContains: "dynamodb error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.setupMock(t)
			svc := NewCompetitionVoteService()

			got, err := svc.PutCompetitionVote(context.Background(), mockDB, tt.vote)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutCompetitionVote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("PutCompetitionVote() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("PutCompetitionVote() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutCompetitionVote() = %v, want %v", got, tt.want)
			}
		})
	}
}
