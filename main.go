package main

import (
	"fmt"
	"log"
	"net/http"
)

func handleHelloTest(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello from the Go server!!"))
} 



func main() {
    server := http.NewServeMux()
    server.HandleFunc("/", handleHelloTest)

    fmt.Printf("Serving on PORT 3001")
    if err := http.ListenAndServe(":3001", server); err != nil {
        log.Fatal(err)
    } 
} 
