package types

type Event struct {
	Id          	string `json:"id,omitempty"`
	EventOwners 	[]string `json:"eventOwners" validate:"required,min=1"`
	Name        	string `json:"name" validate:"required"`
	Description 	string `json:"description" validate:"required"`
	StartTime   	int64 `json:"startTime" validate:"required"`
	EndTime     	int64 `json:"endTime,omitempty"`
	Address     	string `json:"address" validate:"required"`
	Lat    				float64 `json:"lat" validate:"required"`
	Long    			float64 `json:"long" validate:"required"`
	StartingPrice int32 `json:"startingPrice,omitempty"`
	Currency 			string `json:"currency,omitempty"`
	PayeeId  			string `json:"payeeId,omitempty"`
	HasRegistrationFields bool `json:"hasRegistrationFields,omitempty"`
	HasPurchasable bool  `json:"hasPurchasable,omitempty"`
	ImageUrl      string `json:"imageUrl,omitempty"`
	Timezone      string `json:"timezone" validate:"required"`
	Categories    []string `json:"categories,omitempty"`
	Tags    			[]string `json:"tags,omitempty"`
	CreatedAt     int64 `json:"createdAt,omitempty"`
	UpdatedAt     int64 `json:"updatedAt,omitempty"`
	UpdatedBy     string `json:"updatedBy,omitempty"`
	RefUrl 				string `json:"refUrl,omitempty"`
}

type EventSearchResponse struct {
	Events			[]Event `json:"events"`
	Filter 			string 	`json:"filter,omitempty"`
	Query				string	`json:"query,omitempty"`
}
