package types

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// CompetitionConfigInterface defines the methods for competition-related operations
type CompetitionConfigServiceInterface interface {
	GetCompetitionConfigById(ctx context.Context, dynamodbClient DynamoDBAPI, id string) (CompetitionConfigResponse, error)
	GetCompetitionConfigsByPrimaryOwner(ctx context.Context, dynamodbClient DynamoDBAPI, primaryOwner string) (*[]CompetitionConfig, error)
	UpdateCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, id string, competitionConfig CompetitionConfigUpdate) (*CompetitionConfig, error)
	DeleteCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, id string) error
}

// CompetitionRoundServiceInterface defines the methods for competition round operations
type CompetitionRoundServiceInterface interface {
	PutCompetitionRounds(ctx context.Context, dynamodbClient DynamoDBAPI, roundUpdate *[]CompetitionRoundUpdate) (dynamodb.BatchWriteItemOutput, error)
	GetCompetitionRounds(ctx context.Context, dynamodbClient DynamoDBAPI, competitionId string) (*[]CompetitionRound, error)
	GetCompetitionRoundsByEventId(ctx context.Context, dynamodbClient DynamoDBAPI, competitionId string) (*[]CompetitionRound, error)
	GetCompetitionRoundByPrimaryKey(ctx context.Context, dynamodbClient DynamoDBAPI, competitionId, roundNumber string) (*CompetitionRound, error)
	DeleteCompetitionRound(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string) error
}

type CompetitionWaitingRoomParticipantServiceInterface interface {
	PutCompetitionWaitingRoomParticipant(ctx context.Context, dynamodbClient DynamoDBAPI, waitingRoomParticipant CompetitionWaitingRoomParticipantUpdate) (dynamodb.PutItemOutput, error)
	GetCompetitionWaitingRoomParticipants(ctx context.Context, dynamodbClient DynamoDBAPI, competitionId string) ([]CompetitionWaitingRoomParticipant, error)
	DeleteCompetitionWaitingRoomParticipant(ctx context.Context, dynamodbClient DynamoDBAPI, competitionId, userId string) error
}

// CompetitionVoteServiceInterface defines the methods for competition vote operations
type CompetitionVoteServiceInterface interface {
	PutCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, voteUpdate CompetitionVoteUpdate) (dynamodb.PutItemOutput, error)
	GetCompetitionVotesByCompetitionRound(ctx context.Context, dynamodbClient DynamoDBAPI, compositePartitionKey string) ([]CompetitionVote, error)
	DeleteCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, compositePartitionKey, userId string) error
}

// Internal type with proper Go types
type CompetitionConfig struct {
	Id             string   `json:"id"`
	PrimaryOwner   string   `json:"primaryOwner" dynamodbav:"primaryOwner" validate:"required"`
	AuxilaryOwners []string `json:"auxilaryOwners" dynamodbav:"auxilaryOwners" validate:"required"`
	// TODO: `dynamodbav:"eventIds,stringset"` shows to the user string set must only contain non-nil strings which
	// is not sufficiently human readable. We need controller level validation to
	// ensure the user understands the error
	EventIds      []string           `json:"eventIds" dynamodbav:"eventIds" validate:"required"`
	Name          string             `json:"name" dynamodbav:"name" validate:"required"`
	ModuleType    string             `json:"moduleType" dynamodbav:"moduleType" validate:"required,oneof=KARAOKE BOCCE"`
	ScoringMethod string             `json:"scoringMethod" dynamodbav:"scoringMethod" validate:"required,oneof=VOTE_MATCHUPS POINT_MATCHUPS VOTE_TOTAL POINT_TOTAL"`
	Rounds        []CompetitionRound `json:"rounds" dynamodbav:"rounds" validate:"required"`
	Competitors   []string           `json:"competitors" dynamodbav:"competitors"`
	Status        string             `json:"status" dynamodbav:"status" validate:"required,oneof=DRAFT ACTIVE COMPLETE"`
	CreatedAt     int64              `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt     int64              `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionConfigResponse struct {
	CompetitionConfig
	Owners []UserSearchResultDangerous `json:"owners"`
}

type CompetitionConfigUpdatePayload struct {
	CompetitionConfigUpdate
	Rounds []CompetitionRoundUpdate `json:"rounds,omitempty"` // JSON array string
	Teams  []CompetitionTeamUpdate  `json:"teams,omitempty"`  // JSON array string
}

type CompetitionTeamUpdate struct {
	Id          string                  `json:"id,omitempty" dynamodbav:"id"`
	DisplayName string                  `json:"displayName" dynamodbav:"displayName"`
	Competitors []CompetitionCompetitor `json:"competitors,omitempty" dynamodbav:"competitors"`
}

type CompetitionCompetitor struct {
	UserId      string `json:"userId" dynamodbav:"userId"`
	DisplayName string `json:"displayName" dynamodbav:"displayName"`
}

// Update type (all optional fields)
type CompetitionConfigUpdate struct {
	Id           string `json:"id,omitempty" dynamodbav:"id"`
	PrimaryOwner string `json:"primaryOwner" dynamodbav:"primaryOwner"`
	Name         string `json:"name,omitempty" dynamodbav:"name"`
	// TODO: these should be enums for re-use on the client
	ModuleType string `json:"moduleType,omitempty" dynamodbav:"moduleType" validate:"omitempty,oneof=KARAOKE BOCCE"`
	// TODO: these should be enums for re-use on the client
	ScoringMethod  string   `json:"scoringMethod,omitempty" dynamodbav:"scoringMethod" validate:"omitempty,oneof=VOTE_MATCHUPS POINT_MATCHUPS VOTE_TOTAL POINT_TOTAL"`
	AuxilaryOwners []string `json:"auxilaryOwners,omitempty" dynamodbav:"auxilaryOwners"` // JSON array string
	EventIds       []string `json:"eventIds,omitempty" dynamodbav:"eventIds"`             // JSON array string
	Competitors    []string `json:"competitors,omitempty" dynamodbav:"competitors"`       // JSON array string
	// TODO: these should be enums for re-use on the client
	Status    string `json:"status,omitempty" dynamodbav:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETE"`
	UpdatedAt int64  `json:"updatedAt" dynamodbav:"updatedAt"`
}

// CompetitionRound types following the same pattern

type CompetitionRound struct {
	CompetitionId    string  `json:"competitionId" dynamodbav:"competitionId" `
	RoundNumber      int64   `json:"roundNumber" dynamodbav:"roundNumber" `
	EventId          string  `json:"eventId" dynamodbav:"eventId" `
	RoundName        string  `json:"roundName" dynamodbav:"roundName"`
	CompetitorA      string  `json:"competitorA" dynamodbav:"competitorA" `
	CompetitorAScore float64 `json:"competitorAScore" dynamodbav:"competitorAScore"`
	CompetitorB      string  `json:"competitorB" dynamodbav:"competitorB" `
	CompetitorBScore float64 `json:"competitorBScore" dynamodbav:"competitorBScore"`
	Matchup          string  `json:"matchup" dynamdbav:"matchup"`
	Status           string  `json:"status" dynamodbav:"status"`
	IsPending        bool    `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen     bool    `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	Description      string  `json:"description" dynamodbav:"description"`
	CreatedAt        int64   `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionRoundUpdate struct {
	// TODO: clean these up for iterative saving in incomplete state
	CompetitionId    string  `json:"competitionId" dynamodbav:"competitionId" validate:"required"`
	RoundNumber      int64   `json:"roundNumber" dynamodbav:"roundNumber" validate:"required"`
	EventId          string  `json:"eventId,omitempty" dynamodbav:"eventId"`
	RoundName        string  `json:"roundName" dynamodbav:"roundName" validate:"required"`
	CompetitorA      string  `json:"competitorA" dynamodbav:"competitorA" validate:"required"`
	CompetitorAScore float64 `json:"competitorAScore" dynamodbav:"competitorAScore"` // these are required but validation fails with 0 score
	CompetitorB      string  `json:"competitorB" dynamodbav:"competitorB" validate:"required"`
	CompetitorBScore float64 `json:"competitorBScore" dynamodbav:"competitorBScore"` // these are required but validation fails with 0 score
	Matchup          string  `json:"matchup" dynamdbav:"matchup"`
	Status           string  `json:"status" dynamodbav:"status"`
	IsPending        bool    `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen     bool    `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	Description      string  `json:"description" dynamodbav:"description"`
	CreatedAt        int64   `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionWaitingRoomParticipant struct {
	CompetitionId string `json:"competitionId" dynamodbav:"competitionId"`
	UserId        string `json:"userId" dynamodbav:"userId"`
	ExpiresOn     int64  `json:"expiresOn" dynamodbav:"expiresOn"`
}

type CompetitionWaitingRoomParticipantUpdate struct {
	CompetitionId string `json:"competitionId" dynamodbav:"competitionId" validate:"required"`
	UserId        string `json:"userId" dynamodbav:"userId" validate:"required"`
	ExpiresOn     int64  `json:"expiresOn" dynamodbav:"expiresOn" validate:"required"`
}

// Competition Vote Types
type CompetitionVoteUpdate struct {
	CompositePartitionKey string `json:"compositePartitionKey" dynamodbav:"compositePartitionKey"`
	UserId                string `json:"userId" dynamodbav:"userId"`
	VoteRecipientId       string `json:"voteRecipientId" dynamodbav:"voteRecipientId" validate:"required"`
	VoteValue             int64  `json:"voteValue" dynamodbav:"voteValue" validate:"required"`
	ExpiresOn             int64  `json:"expiresOn" dynamodbav:"expiresOn"`
}

type CompetitionVote struct {
	CompositePartitionKey string `json:"compositePartitionKey" dynamodbav:"compositePartitionKey" `
	UserId                string `json:"userId" dynamodbav:"userId" `
	VoteRecipientId       string `json:"voteRecipientId" dynamodbav:"voteRecipientId" `
	VoteValue             int64  `json:"voteValue" dynamodbav:"voteValue" `
	ExpiresOn             int64  `json:"expiresOn" dynamodbav:"expiresOn" `
}
