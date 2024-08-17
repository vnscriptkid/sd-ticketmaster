package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vnscriptkid/sd-ticketmaster/search/handler"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/search", handler.SearchEvents).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
