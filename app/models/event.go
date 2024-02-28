package models

type Event struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Datetime    string `json:"datetime" dynamodbav:"datetime"`
	Address     string `json:"address" dynamodbav:"address"`
	ZipCode     string `json:"zip_code" dynamodbav:"zip_code"`
	Country     string `json:"country" dynamodbav:"country"`
}
