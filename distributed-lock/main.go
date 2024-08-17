package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func initDB() {
	var err error
	db, err = sql.Open("postgres", "user=postgres password=123456 dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
}

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis address
	})
}

func reserveTicket(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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

	lockKey := fmt.Sprintf("ticket_lock:%s", ticketID)
	ttl := 10 * time.Minute

	// Try to acquire the lock with TTL
	success, err := rdb.SetNX(ctx, lockKey, userID, ttl).Result()
	if err != nil {
		http.Error(w, "Failed to acquire lock", http.StatusInternalServerError)
		return
	}
	if !success {
		http.Error(w, "Ticket is already reserved", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Ticket reserved",
		"expires_at": time.Now().Add(ttl),
	})
}

func confirmReservation(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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

	// Check if the reservation exists in Redis and get the user who reserved it
	lockKey := fmt.Sprintf("ticket_lock:%s", ticketID)
	storedUserID, err := rdb.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		http.Error(w, "Reservation expired or not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to verify reservation", http.StatusInternalServerError)
		return
	}

	// Check if the user IDs match
	if storedUserID != userID {
		http.Error(w, "User ID does not match the reservation", http.StatusForbidden)
		return
	}

	// Start a transaction to confirm the reservation in the tickets table
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`UPDATE tickets SET status = 'BOOKED', user_id = $1 WHERE id = $2`, userID, ticketID)
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

	// Remove the lock from Redis since the reservation is confirmed
	rdb.Del(ctx, lockKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Reservation confirmed",
	})
}

func main() {
	initDB()
	initRedis()

	http.HandleFunc("/reserve", reserveTicket)
	http.HandleFunc("/confirm", confirmReservation)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
