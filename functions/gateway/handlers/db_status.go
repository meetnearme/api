package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/meetnearme/api/functions/gateway/transport" // Adjust import path if necessary
)

// GetDatabaseTablesHandler handles requests to retrieve database tables
func GetDatabaseTablesHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Get the RDS client instance
	log.Printf("In the db table handler")
	dbClient := transport.GetRdsDB()

	log.Printf("DBclient: %v", dbClient)

	// Define a context
	ctx := context.Background()

	// Query to get table names
	query := `SELECT * FROM organizations;`

	// Execute the statement
	resp, err := dbClient.ExecStatement(ctx, query)
	if err != nil {
		return transport.SendHtmlError(w, []byte("Failed to query database: "+err.Error()), http.StatusInternalServerError)
	}

	log.Printf("Resp from dbclient: \n%v", resp)

	log.Printf("Records on resp: \n%v", resp.Records)


	// Prepare a slice to hold the table names
	var tables []string
	for _, row := range resp.Records {
		if len(row) > 0 && row[0] == nil {
			// tables = append(tables, *row[0].StringValue)
		}
	}

	// Convert tables slice to JSON
	responseBody, err := json.Marshal(map[string][]string{"tables": tables})
	if err != nil {
		return transport.SendServerRes(w, []byte(fmt.Sprintf("Failed to marshal response: %v", err)), http.StatusInternalServerError, err)
	}

	// Respond with table names
	w.Header().Set("Content-Type", "application/json")
	return transport.SendServerRes(w, responseBody, http.StatusOK, nil)
}

