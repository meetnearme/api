package shared

type Page struct {
	Name, Desc, Slug string
	// Handlers         func(string, *http.ServeMux)
}

const EventsTablePrefix = "Events"

type Event struct {
	Id string `json:"id" dynamodbav:"id"`
	Name string `json:"name" dynamodbav:"name"`
	Description string  `json:"description" dynamodbav:"description"`
	Datetime string  `json:"datetime" dynamodbav:"datetime"`
	Address string  `json:"address" dynamodbav:"address"`
	ZipCode string  `json:"zip_code" dynamodbav:"zip_code"`
	Country string  `json:"country" dynamodbav:"country"`
}


type CreateEvent struct {
    Name string `json:"name" validate:"required"`
    Description string  `json:"description" validate:"required"`
    Datetime string  `json:"datetime" validate:"required"`
    Address string  `json:"address" validate:"required"`
    ZipCode string  `json:"zip_code" validate:"required"`
    Country string  `json:"country" validate:"required"`
}
