package types

import (
	"context"
	"time"

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
	GetCompetitionVotesByPk(ctx context.Context, dynamodbClient DynamoDBAPI, pk string) (*CompetitionVote, error)
	DeleteCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string) error
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
	Rounds []CompetitionRound `json:"rounds,omitempty" dynamodbav:"rounds"` // JSON array string
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
	// Competitors      []string `json:"competitors" dynamodbav:"competitors"`
	IsPending    string `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen string `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	CreatedAt    int64  `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionRoundUpdate struct {
	CompetitionId    string  `json:"competitionId" dynamodbav:"competitionId"`
	RoundNumber      int64   `json:"roundNumber" dynamodbav:"roundNumber" validate:"required"`
	EventId          string  `json:"eventId,omitempty" dynamodbav:"eventId"`
	RoundName        string  `json:"roundName" dynamodbav:"roundName" validate:"required"`
	CompetitorA      string  `json:"competitorA" dynamodbav:"competitorA" validate:"required"`
	CompetitorAScore float64 `json:"competitorAScore" dynamodbav:"competitorAScore"` // these are required but validation fails with 0 score
	CompetitorB      string  `json:"competitorB" dynamodbav:"competitorB" validate:"required"`
	CompetitorBScore float64 `json:"competitorBScore" dynamodbav:"competitorBScore"` // these are required but validation fails with 0 score
	Matchup          string  `json:"matchup" dynamdbav:"matchup" validate:"required"`
	Status           string  `json:"status" dynamodbav:"status"`
	// is this needed? ?
	// Competitors      []string `json:"competitors" dynamodbav:"competitors" validate:"required"` // JSON array string - these are userIds that are not uuids
	IsPending    string `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen string `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	CreatedAt    int64  `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionWaitingRoomParticipant struct {
	CompetitionId string `json:"competitionId" dynamodbav:"competitionId"`
	UserId        string `json:"userId" dynamodbav:"userId"`
	PurchaseId    string `json:"purchaseId" dynamodbav:"purchaseId"`
	ExpiresOn     int64  `json:"expiresOn" dynamodbav:"expiresOn"`
}

type CompetitionWaitingRoomParticipantUpdate struct {
	CompetitionId string `json:"competitionId" dynamodbav:"competitionId" validate:"required"`
	UserId        string `json:"userId" dynamodbav:"userId" validate:"required"`
	PurchaseId    string `json:"purchaseId" dynamodbav:"purchaseId" validate:"required"`
	ExpiresOn     int64  `json:"expiresOn" dynamodbav:"expiresOn" validate:"required"`
}

// Competition Vote Types
type CompetitionVoteUpdate struct {
	PK           string    `json:"PK" dynamodbav:"PK" validate:"required"`
	SK           string    `json:"SK" dynamodbav:"SK" validate:"required"`
	CompetitorId string    `json:"competitorId" dynamodbav:"competitorId" validate:"required"`
	VoteValue    int       `json:"voteValue" dynamodbav:"voteValue" validate:"required"`
	ModuleType   string    `json:"moduleType" dynamodbav:"moduleType" validate:"required,oneof=KARAOKE BOCCE"`
	CreatedAt    time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	ExpiresOn    int64     `json:"expiresOn" dynamodbav:"expiresOn" validate:"required"`
}

type CompetitionVote struct {
	PK           string    `json:"PK" dynamodbav:"PK"`
	SK           string    `json:"SK" dynamodbav:"SK"`
	CompetitorId string    `json:"competitorId" dynamodbav:"competitorId"`
	VoteValue    int       `json:"voteValue" dynamodbav:"voteValue"`
	ModuleType   string    `json:"moduleType" dynamodbav:"moduleType"`
	CreatedAt    time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	ExpiresOn    int64     `json:"expiresOn" dynamodbav:"expiresOn" validate:"required"`
}
