package types

import (
	"context"
	"time"
)

// CompetitionConfigInterface defines the methods for competition-related operations
type CompetitionConfigServiceInterface interface {
	InsertCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, competitionConfig CompetitionConfigInsert) (*CompetitionConfig, error)
	GetCompetitionConfigByPk(ctx context.Context, dynamodbClient DynamoDBAPI, primaryOwner, id string) (*CompetitionConfig, error)
	UpdateCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, primaryOwner, id string, competitionConfig CompetitionConfigUpdate) (*CompetitionConfig, error)
	DeleteCompetitionConfig(ctx context.Context, dynamodbClient DynamoDBAPI, primaryOwner, id string) error
}

// CompetitionRoundServiceInterface defines the methods for competition round operations
type CompetitionRoundServiceInterface interface {
	InsertCompetitionRound(ctx context.Context, dynamodbClient DynamoDBAPI, round CompetitionRoundInsert) (*CompetitionRound, error)
	GetCompetitionRounds(ctx context.Context, dynamodbClient DynamoDBAPI, pk, competitionId string) (*[]CompetitionRound, error)
	GetCompetitionRoundByPk(ctx context.Context, dynamodbClient DynamoDBAPI, pk, competitionId, roundNumber string) (*CompetitionRound, error)
	UpdateCompetitionRound(ctx context.Context, dynamodbClient DynamoDBAPI, pk, sk string, roundUpdate CompetitionRoundUpdate) (*CompetitionRound, error)
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
	Id             string                   `json:"id,omitempty" dynamodbav:"id"`
	PrimaryOwnerId string                   `json:"primaryOwner" dynamodbav:"primaryOwner"`
	AuxilaryOwners []string                 `json:"auxilaryOwners" dynamodbav:"auxilaryOwners"`
	EventIds       []string                 `json:"eventIds" dynamodbav:"eventIds"`
	Name           string                   `json:"name" dynamodbav:"name" validate:"required"`
	ModuleType     string                   `json:"moduleType" dynamodbav:"moduleType" validate:"required,oneof=KARAOKE BOCCE"`
	ScoringMethod  string                   `json:"scoringMethod" dynamodbav:"scoringMethod" validate:"required,oneof=POINTS VOTES"`
	Rounds         []CompetitionRoundInsert `json:"rounds" dynamodbav:"rounds"`
	Competitors    []string                 `json:"competitors" dynamodbav:"competitors"`
	Status         string                   `json:"status" dynamodbav:"status" validate:"required,oneof=DRAFT ACTIVE COMPLETE"`
	CreatedAt      int64                    `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt      int64                    `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Internal type with proper Go types
type CompetitionConfig struct {
	Id             string             `json:"id"`
	PrimaryOwner   string             `json:"primaryOwner" dynamodbav:"primaryOwner"`
	AuxilaryOwners []string           `json:"auxilaryOwners" dynamodbav:"auxilaryOwners"`
	EventIds       []string           `json:"eventIds" dynamodbav:"eventIds"`
	Name           string             `json:"name" dynamodbav:"name"`
	ModuleType     string             `json:"moduleType" dynamodbav:"moduleType"`
	ScoringMethod  string             `json:"scoringMethod" dynamodbav:"scoringMethod"`
	Rounds         []CompetitionRound `json:"rounds" dynamodbav:"rounds"`
	Competitors    []string           `json:"competitors" dynamodbav:"competitors"`
	Status         string             `json:"status" dynamodbav:"status"`
	CreatedAt      int64              `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt      int64              `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Update type (all optional fields)
type CompetitionConfigUpdate struct {
	Name           string `json:"name,omitempty" dynamodbav:"name"`
	ModuleType     string `json:"moduleType,omitempty" dynamodbav:"moduleType" validate:"omitempty,oneof=KARAOKE BOCCE"`
	ScoringMethod  string `json:"scoringMethod,omitempty" dynamodbav:"scoringMethod" validate:"omitempty,oneof=POINTS VOTES"`
	AuxilaryOwners string `json:"auxilaryOwners,omitempty" dynamodbav:"auxilaryOwners"` // JSON array string
	EventIds       string `json:"eventIds,omitempty" dynamodbav:"eventIds"`             // JSON array string
	Rounds         string `json:"rounds,omitempty" dynamodbav:"rounds"`                 // JSON array string
	Competitors    string `json:"competitors,omitempty" dynamodbav:"competitors"`       // JSON array string
	Status         string `json:"status,omitempty" dynamodbav:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETE"`
}

// CompetitionRound types following the same pattern
type CompetitionRoundInsert struct {
	PK               string   `json:"PK" dynamodbav:"PK"` // OWNER_<ownerId>
	SK               string   `json:"SK" dynamodbav:"SK"` // COMPETITION_<competitionId>_ROUND_<roundNumber>
	OwnerId          string   `json:"ownerId" dynamodbav:"ownerId"`
	EventId          string   `json:"eventId" dynamodbav:"eventId"`
	RoundName        string   `json:"roundName" dynamodbav:"roundName"`
	RoundNumber      int64    `json:"roundNumber" dynamodbav:"roundNumber"`
	CompetitorA      string   `json:"competitorA" dynamodbav:"competitorA" validate:"required"`
	CompetitorAScore int64    `json:"competitorAScore" dynamodbav:"competitorAScore"`
	CompetitorB      string   `json:"competitorB" dynamodbav:"competitorB" validate:"required"`
	CompetitorBScore int64    `json:"competitorBScore" dynamodbav:"competitorBScore"`
	Matchup          string   `json:"matchup" dynamdbav:"matchup"`
	Status           string   `json:"status" dynamodbav:"status"`
	Competitors      []string `json:"competitors" dynamodbav:"competitors"` // JSON array string - these are userIds that are not uuids
	IsPending        string   `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen     string   `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	CreatedAt        int64    `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionRound struct {
	PK               string   `json:"PK" dynamodbav:"PK"`
	SK               string   `json:"SK" dynamodbav:"SK"`
	OwnerId          string   `json:"ownerId" dynamodbav:"ownerId"`
	EventId          string   `json:"eventId" dynamodbav:"eventId"`
	RoundName        string   `json:"roundName" dynamodbav:"roundName"`
	RoundNumber      int64    `json:"roundNumber" dynamodbav:"roundNumber"`
	CompetitorA      string   `json:"competitorA" dynamodbav:"competitorA" validate:"required"`
	CompetitorAScore int64    `json:"competitorAScore" dynamodbav:"competitorAScore"`
	CompetitorB      string   `json:"competitorB" dynamodbav:"competitorB" validate:"required"`
	CompetitorBScore int64    `json:"competitorBScore" dynamodbav:"competitorBScore"`
	Matchup          string   `json:"matchup" dynamdbav:"matchup"`
	Status           string   `json:"status" dynamodbav:"status"`
	Competitors      []string `json:"competitors" dynamodbav:"competitors"`
	IsPending        string   `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen     string   `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	CreatedAt        int64    `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CompetitionRoundUpdate struct {
	RoundName        string `json:"roundName,omitempty" dynamodbav:"roundName"`
	Status           string `json:"status,omitempty" dynamodbav:"status" validate:"omitempty,oneof=ACTIVE COMPLETE CANCELLED PENDING"`
	Competitors      string `json:"competitors,omitempty" dynamodbav:"competitors"` // JSON array string
	CompetitorA      string `json:"competitorA" dynamodbav:"competitorA" validate:"required"`
	CompetitorAScore int    `json:"competitorAScore" dynamodbav:"competitorAScore"`
	CompetitorB      string `json:"competitorB" dynamodbav:"competitorB" validate:"required"`
	CompetitorBScore int    `json:"competitorBScore" dynamodbav:"competitorBScore"`
	Matchup          string `json:"matchup" dynamdbav:"matchup"`
	IsPending        string `json:"isPending" dynamodbav:"isPending"`
	IsVotingOpen     string `json:"isVotingOpen" dynamodbav:"isVotingOpen"`
	UpdatedAt        int64  `json:"updatedAt" dynamodbav:"updatedAt"`
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
	TTL          int64     `json:"TTL" dynamodbav:"TTL"`
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
