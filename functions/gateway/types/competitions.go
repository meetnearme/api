package types

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// CompetitionConfigInterface defines the methods for competition-related operations
type CompetitionConfigServiceInterface interface {
	InsertCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, competitionConfig CompetitionConfigInsert) (*CompetitionConfig, error)
	GetCompetitionConfigById(ctx context.Context, dynamodbClient DynamoDBAPI, id string) (*CompetitionConfig, error)
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

// CompetitionVoteServiceInterface defines the methods for competition vote operations
type CompetitionVoteServiceInterface interface {
	InsertCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, vote CompetitionVoteInsert) (*CompetitionVote, error)
	GetCompetitionVoteByPk(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string) (*CompetitionVote, error)
	GetVotesByRoundID(ctx context.Context, dynamodbClient DynamoDBAPI, roundId string) ([]CompetitionVote, error)
	UpdateCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string, vote CompetitionVoteUpdate) (*CompetitionVote, error)
	DeleteCompetitionVote(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string) error
}

// Insert type (all strings for DynamoDB)
type CompetitionConfigInsert struct {
	Id             string   `json:"id,omitempty" dynamodbav:"id"`
	PrimaryOwner   string   `json:"primaryOwner" dynamodbav:"primaryOwner"`
	AuxilaryOwners []string `json:"auxilaryOwners" dynamodbav:"auxilaryOwners,stringset"`
	EventIds       []string `json:"eventIds" dynamodbav:"eventIds,stringset"`
	Name           string   `json:"name" dynamodbav:"name" validate:"required"`
	ModuleType     string   `json:"moduleType" dynamodbav:"moduleType" validate:"required,oneof=KARAOKE BOCCE"`
	ScoringMethod  string   `json:"scoringMethod" dynamodbav:"scoringMethod" validate:"required,oneof=POINTS VOTES"`
	Rounds         []string `json:"rounds" dynamodbav:"rounds,stringset"`
	Competitors    []string `json:"competitors" dynamodbav:"competitors,stringset"`
	Status         string   `json:"status" dynamodbav:"status" validate:"required,oneof=DRAFT ACTIVE COMPLETE"`
	CreatedAt      int64    `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt      int64    `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Internal type with proper Go types
type CompetitionConfig struct {
	Id             string   `json:"id"`
	PrimaryOwner   string   `json:"primaryOwner" dynamodbav:"primaryOwner"`
	AuxilaryOwners []string `json:"auxilaryOwners" dynamodbav:"auxilaryOwners,stringset"`
	EventIds       []string `json:"eventIds" dynamodbav:"eventIds,stringset"`
	Name           string   `json:"name" dynamodbav:"name"`
	ModuleType     string   `json:"moduleType" dynamodbav:"moduleType"`
	ScoringMethod  string   `json:"scoringMethod" dynamodbav:"scoringMethod"`
	Rounds         []string `json:"rounds" dynamodbav:"rounds,stringset"`
	Competitors    []string `json:"competitors" dynamodbav:"competitors,stringset"`
	Status         string   `json:"status" dynamodbav:"status"`
	CreatedAt      int64    `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt      int64    `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Update type (all optional fields)
type CompetitionConfigUpdate struct {
	Name           string   `json:"name,omitempty" dynamodbav:"name"`
	ModuleType     string   `json:"moduleType,omitempty" dynamodbav:"moduleType" validate:"omitempty,oneof=KARAOKE BOCCE"`
	ScoringMethod  string   `json:"scoringMethod,omitempty" dynamodbav:"scoringMethod" validate:"omitempty,oneof=POINTS VOTES"`
	AuxilaryOwners []string `json:"auxilaryOwners,omitempty" dynamodbav:"auxilaryOwners,stringset"` // JSON array string
	EventIds       []string `json:"eventIds,omitempty" dynamodbav:"eventIds,stringset"`             // JSON array string
	Rounds         []string `json:"rounds,omitempty" dynamodbav:"rounds,stringset"`                 // JSON array string
	Competitors    []string `json:"competitors,omitempty" dynamodbav:"competitors,stringset"`       // JSON array string
	Status         string   `json:"status,omitempty" dynamodbav:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETE"`
	UpdatedAt      int64    `json:"updatedAt" dynamodbav:"updatedAt"`
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

// Competition Vote Types
type CompetitionVoteInsert struct {
	PK           string    `json:"PK" dynamodbav:"PK" validate:"required"`
	SK           string    `json:"SK" dynamodbav:"SK" validate:"required"`
	CompetitorId string    `json:"competitorId" dynamodbav:"competitorId" validate:"required"`
	VoteValue    int       `json:"voteValue" dynamodbav:"voteValue" validate:"required"`
	ModuleType   string    `json:"moduleType" dynamodbav:"moduleType" validate:"required,oneof=KARAOKE BOCCE"`
	CreatedAt    time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	TTL          int64     `json:"ttl" dynamodbav:"ttl"`
}

type CompetitionVote struct {
	PK           string    `json:"PK" dynamodbav:"PK"`
	SK           string    `json:"SK" dynamodbav:"SK"`
	CompetitorId string    `json:"competitorId" dynamodbav:"competitorId"`
	VoteValue    int       `json:"voteValue" dynamodbav:"voteValue"`
	ModuleType   string    `json:"moduleType" dynamodbav:"moduleType"`
	CreatedAt    time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	TTL          int64     `json:"TTL" dynamodbav:"TTL"`
}

type CompetitionVoteUpdate struct {
	VoteValue int       `json:"voteValue,omitempty" dynamodbav:"voteValue"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	TTL       int64     `json:"TTL,omitempty" dynamodbav:"TTL"`
}
