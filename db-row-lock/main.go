package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("postgres", "user=postgres password=123456 dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
}

func reserveTicket(w http.ResponseWriter, r *http.Request) {
	// Decode JSON body into a map
	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract values from the map
	ticketID, ok := req["ticket_id"].(string)
	if !ok || ticketID == "" {
		http.Error(w, "Invalid or missing ticket_id", http.StatusBadRequest)
		return
	}
	userID, ok := req["user_id"].(string)
	if !ok || userID == "" {
		http.Error(w, "Invalid or missing user_id", http.StatusBadRequest)
		return
	}

	reservationID := uuid.New()
	expiresAt := time.Now().Add(10 * time.Minute)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Lock the ticket row
	var ticketStatus string
	err = tx.QueryRow(`SELECT status FROM tickets WHERE id = $1 FOR UPDATE`, ticketID).Scan(&ticketStatus)
	if err != nil {
		fmt.Printf("Error querying ticket: %s\n", err)
		tx.Rollback()
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if ticketStatus != "AVAILABLE" {
		tx.Rollback()
		http.Error(w, "Ticket is not available", http.StatusConflict)
		return
	}

	// Update ticket status
	_, err = tx.Exec(`UPDATE tickets SET status = 'RESERVED' WHERE id = $1`, ticketID)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert reservation
	_, err = tx.Exec(`INSERT INTO reservations (id, ticket_id, user_id, expires_at, status) VALUES ($1, $2, $3, $4, 'PENDING')`, reservationID, ticketID, userID, expiresAt)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Reservation ID: %s, Expires At: %s", reservationID, expiresAt)
}

func confirmReservation(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reservationID, ok := req["reservation_id"].(string)
	if !ok || reservationID == "" {
		http.Error(w, "Invalid or missing reservation_id", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if reservation is still valid
	var status string
	err = tx.QueryRow(`SELECT status FROM reservations WHERE id = $1 AND expires_at > $2`, reservationID, time.Now()).Scan(&status)
	if err != nil || status != "PENDING" {
		tx.Rollback()
		http.Error(w, "Invalid or expired reservation", http.StatusBadRequest)
		return
	}

	// Update reservation status
	_, err = tx.Exec(`UPDATE reservations SET status = 'CONFIRMED' WHERE id = $1`, reservationID)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update ticket status
	_, err = tx.Exec(`UPDATE tickets SET status = 'BOOKED' WHERE id = (SELECT ticket_id FROM reservations WHERE id = $1)`, reservationID)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Reservation confirmed")
}

func main() {
	initDB()

	http.HandleFunc("/reserve", reserveTicket)
	http.HandleFunc("/confirm", confirmReservation)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
