package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var (
	dbHost     = "localhost"
	dbPort     = os.Getenv("SESHUJOBS_POSTGRES_PORT")
	dbUser     = os.Getenv("SESHUJOBS_POSTGRES_USER")
	dbPassword = os.Getenv("SESHUJOBS_POSTGRES_PASSWORD")
	dbName     = os.Getenv("SESHUJOBS_POSTGRES_DB")
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/pingdb", pingDBHandler).Methods("GET")

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func pingDBHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(dbPort)
	dbport, err := strconv.Atoi(dbPort)
	if err != nil {
		http.Error(w, "Invalid DB port", http.StatusInternalServerError)
		log.Println("Invalid DB port:", dbPort)
		return
	}

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbport, dbUser, dbPassword, dbName,
	)

	fmt.Print(psqlInfo)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		http.Error(w, "Failed to connect to DB", http.StatusInternalServerError)
		log.Println("Connection error:", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		http.Error(w, "DB not reachable", http.StatusInternalServerError)
		log.Println("Ping error:", err)
		return
	}

	fmt.Fprintln(w, "Successfully connected to the DB!")
}
