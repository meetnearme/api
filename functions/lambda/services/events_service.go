package services

import "time"

type Event struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Datetime    string `json:"datetime" dynamodbav:"datetime"`
	Address     string `json:"address" dynamodbav:"address"`
	ZipCode     string `json:"zip_code" dynamodbav:"zip_code"`
	Country     string `json:"country" dynamodbav:"country"`
}

func GetEvents() []Event {
	return []Event{
		{
			Id:          "event1",
			Name:        "Concert Night",
			Description: "Enjoy live music under the stars",
			Datetime:    time.Now().Format(time.RFC3339),
			Address:     "123 Main St",
			ZipCode:     "12345",
			Country:     "US",
		},
		{
			Id:          "event2",
			Name:        "Food Festival",
			Description: "Sample delicious cuisines from around the world",
			Datetime:    time.Now().AddDate(0, 0, 7).Format(time.RFC3339), // One week from now
			Address:     "Central Park",
			ZipCode:     "98765",
			Country:     "US",
		},
		{
			Id:          "event3",
			Name:        "Movie Marathon",
			Description: "Catch all your favorite classics on the big screen",
			Datetime:    time.Now().AddDate(0, 1, 0).Format(time.RFC3339), // One month from now
			Address:     "Galaxy Theater",
			ZipCode:     "54321",
			Country:     "US",
		},
		{
			Id:          "event4",
			Name:        "Art Exhibition",
			Description: "Explore stunning works from emerging and established artists",
			Datetime:    time.Now().AddDate(0, 2, 0).Format(time.RFC3339), // Two months from now
			Address:     "Modern Art Museum",
			ZipCode:     "09876",
			Country:     "US",
		},
		{
			Id:          "event5",
			Name:        "Hiking Adventure",
			Description: "Challenge yourself with a scenic hike in nature",
			Datetime:    time.Now().AddDate(0, 3, 0).Format(time.RFC3339), // Three months from now
			Address:     "Mountain Trails",
			ZipCode:     "78901",
			Country:     "US",
		},
	}
}
