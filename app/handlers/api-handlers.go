package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meetnearme/api/app/models"
)

func GetEventsList(c *gin.Context) {
	var events = []models.Event{
		{
			Id:          "event1",
			Name:        "Event One",
			Description: "This is the first event",
			Datetime:    "2023-04-01T10:00:00Z",
			Address:     "123 Main St",
			ZipCode:     "12345",
			Country:     "Country A",
		},
		{
			Id:          "event2",
			Name:        "Event Two",
			Description: "This is the second event",
			Datetime:    "2023-04-02T11:00:00Z",
			Address:     "456 Elm Rd",
			ZipCode:     "23456",
			Country:     "Country B",
		},
		{
			Id:          "event3",
			Name:        "Event Three",
			Description: "This is the third event",
			Datetime:    "2023-04-03T12:00:00Z",
			Address:     "789 Oak Ave",
			ZipCode:     "34567",
			Country:     "Country C",
		},
		{
			Id:          "event4",
			Name:        "Event Four",
			Description: "This is the fourth event",
			Datetime:    "2023-04-04T13:00:00Z",
			Address:     "101 Pine St",
			ZipCode:     "45678",
			Country:     "Country D",
		},
		{
			Id:          "event5",
			Name:        "Event Five",
			Description: "This is the fifth event",
			Datetime:    "2023-04-05T14:00:00Z",
			Address:     "202 Birch Rd",
			ZipCode:     "56789",
			Country:     "Country E",
		},
		{
			Id:          "event6",
			Name:        "Event Six",
			Description: "This is the sixth event",
			Datetime:    "2023-04-06T15:00:00Z",
			Address:     "303 Cedar Ave",
			ZipCode:     "67890",
			Country:     "Country F",
		},
	}
	c.JSON(http.StatusOK, events)
}
