package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)


const TableName = "Users"

var db *dynamodb.Client

type User struct {
    Id string `json:"id" dynamodbav:"id"`
    Name string `json:"name" dynamodbav:"name"`
    Kind string `json:"kind" dynamodbav:"kind"`
    Region string `json:"region" dynamodbav:"region"`
} 

func init() {
    db = CreateLocalClient()
}

func CreateLocalClient() *dynamodb.Client {
    sdkConfig, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithEndpointResolver(aws.EndpointResolverFunc(
            func(service, region string) (aws.Endpoint, error) {
                return aws.Endpoint{URL: "http://dynamodb-local:8000"}, nil 
            })),
        config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
            Value: aws.Credentials{
                AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
                Source: "Hard-coded credentials; values are irrelevant for local dynamo",
            },
        }),
    )
    if err != nil {
        panic(err)
    } 
    return dynamodb.NewFromConfig(sdkConfig)
} 

func listItems(ctx context.Context) ([]User, error) {
    users := make([]User, 0)
    var token map[string]types.AttributeValue

    for {
        input := &dynamodb.ScanInput{
            TableName: aws.String(TableName),
            ExclusiveStartKey: token,
        }

        result, err := db.Scan(ctx, input)
        if err != nil {
            return nil, err
        } 

        var fetchedUsers []User
        err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedUsers)
        if err != nil {
            return nil, err
        } 

        users = append(users, fetchedUsers...)
        token = result.LastEvaluatedKey
        if token == nil {
            break
        }
    }
    return users, nil 
} 

func insertItem( ctx context.Context, createUser CreateUser) (*User, error) {
    user := User{
        Name: createUser.Name,
        Kind: createUser.Kind,
        Region: createUser.Region,
        Id: uuid.NewString(),
    }

    item, err := attributevalue.MarshalMap(user)
    if err != nil {
        fmt.Println("Hitting after marshal map")
        return nil, err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String(TableName),
        Item: item,
    }

    res, err := db.PutItem(ctx, input)
    if err != nil {
        fmt.Println("Hitting after put item")
        return nil, err
    }


    err = attributevalue.UnmarshalMap(res.Attributes, &user)
    if err != nil {
        return nil, err
    }
    return &user, nil 
} 
