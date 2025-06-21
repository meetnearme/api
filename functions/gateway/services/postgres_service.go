package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connection pooling for Postgres
var pool *pgxpool.Pool
var ctx = context.Background()

func GetPostgresClient() (*pgxpool.Pool, error) {

	postGresHost := os.Getenv("POSTGRES_HOST")
	postGresPort := os.Getenv("POSTGRES_PORT")
	postGresUser := os.Getenv("POSTGRES_USER")
	postGresPassword := os.Getenv("POSTGRES_PASSWORD")
	postGresDB := os.Getenv("POSTGRES_DB")

	if postGresHost == "" {
		log.Printf("Please add POSTGRES_HOST")
	}

	if postGresPort == "" {
		log.Printf("Please add POSTGRES_PORT")
	}

	if postGresUser == "" {
		log.Printf("Please add POSTGRES_USER")
	}

	if postGresPassword == "" {
		log.Printf("Please add POSTGRES_PASSWORD")
	}

	if postGresDB == "" {
		log.Printf("Please add POSTGRES_DB")
	}

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		postGresUser,
		postGresPassword,
		postGresHost,
		postGresPort,
		postGresDB,
	)

	dbpool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection pool: %v\n", err)
	}

	fmt.Println("Connected to PostgreSQL database!")

	return dbpool, nil
}
