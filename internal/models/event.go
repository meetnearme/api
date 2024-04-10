package models

import "github.com/go-playground/validator/v10"

type Event struct {
    ID string `json:"id" dynamodbav:"id"`
    StartTime string `json:"startTime" dynamodbav:"start_time" validator:"required"`
    Latitude float64 `json:"latitude" dynamodbav:"latitude" validator:"required"`
    Longitude float64 `json:"longitude" dynamodbav:"longitude" validator:"required"`
    ZOrderIndex []byte `json:"zOrderIndex" dynamodbav:"z_order_index" validator:"required"`
    Title string `json:"title" dynamodbav:"title" validator:"required"`
    Description string `json:"description" dynamodbav:"description" validator:"required"`
    Location string `json:"location" dynamodbav:"location" validator:"required"`
    URL string `json:"url" dynamodbav:"url" validator:"url"`
} 
