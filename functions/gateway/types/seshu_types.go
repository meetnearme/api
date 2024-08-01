package types

import ( 
    "net/url"
)


type EventInfo struct {
	EventTitle    	 string `json:"event_title"`
	EventLocation 	 string `json:"event_location"`
	EventDate     	 string `json:"event_date"`
	EventURL      	 string `json:"event_url"`
	EventDescription string `json:"event_description"`
}

type SeshuSession struct {
	OwnerId    string `json:"ownerId" validate:"required"`
	Url        string `json:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" validate:"required"`
	UrlPath        string `json:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" validate:"optional"`
	Html      string `json:"html" validate:"required"`
	EventCandidates	 []EventInfo `json:"eventCandidates" validate:"optional"`
	Status 		string `json:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" validate:"required"`
	UpdatedAt int64  `json:"updatedAt" validate:"required"`
	ExpireAt  int64  `json:"expireAt" validate:"required"`
}

type SeshuSessionInput struct {
	SeshuSession
	CreatedAt struct{} `json:"createdAt,omitempty"`
	UpdatedAt struct{} `json:"updatedAt,omitempty"`
	ExpireAt struct{} `json:"expireAt,omitempty"`
}

type SeshuSessionInsert struct {
	OwnerId    string `json:"ownerId" dynamodbav:"ownerId" validate:"required"`
	Url        string `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" dynamodbav:"urlDomain" validate:"required"`
	UrlPath        string `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html      string `json:"html" dynamodbav:"html" validate:"required"`
	EventCandidates	 []EventInfo `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations [][]bool `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status 		string `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" dynamodbav:"createdAt" validate:"required"`
	UpdatedAt int64  `json:"updatedAt" dynamodbav:"updatedAt" validate:"required"`
	ExpireAt  int64  `json:"expireAt" dynamodbav:"expireAt" validate:"required"`
}
type SeshuSessionUpdate struct {
	OwnerId    string `json:"ownerId" dynamodbav:"ownerId" validate:"optional"`
	Url        string `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" dynamodbav:"urlDomain" validate:"optional"`
	UrlPath        string `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html      string `json:"html" dynamodbav:"html" validate:"optional"`
	EventCandidates   []EventInfo `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations [][]bool `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status 		string `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" dynamodbav:"createdAt" validate:"optional"`
	UpdatedAt int64  `json:"updatedAt" dynamodbav:"updatedAt" validate:"optional"`
	ExpireAt  int64  `json:"expireAt" dynamodbav:"expireAt" validate:"optional"`
}
