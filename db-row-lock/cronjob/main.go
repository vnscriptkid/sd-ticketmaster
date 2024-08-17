package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("postgres", "user=postgres dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
}

func expirePendingReservations() {
	for {
		// Wait for 1 minute before running the job again
		time.Sleep(1 * time.Minute)

		// Start a new transaction
		tx, err := db.Begin()
		if err != nil {
			log.Println("Error starting transaction:", err)
			continue
		}

		// Find expired reservations
		rows, err := tx.Query(`SELECT id, ticket_id FROM reservations WHERE expires_at < $1 AND status = 'PENDING'`, time.Now())
		if err != nil {
			log.Println("Error querying expired reservations:", err)
			tx.Rollback()
			continue
		}

		var reservationID, ticketID string
		for rows.Next() {
			err := rows.Scan(&reservationID, &ticketID)
			if err != nil {
				log.Println("Error scanning row:", err)
				continue
			}

			// Update reservation status to CANCELLED
			_, err = tx.Exec(`UPDATE reservations SET status = 'CANCELLED' WHERE id = $1`, reservationID)
			if err != nil {
				log.Println("Error updating reservation status:", err)
				tx.Rollback()
				continue
			}

			// Update ticket status to AVAILABLE
			_, err = tx.Exec(`UPDATE tickets SET status = 'AVAILABLE' WHERE id = $1`, ticketID)
			if err != nil {
				log.Println("Error updating ticket status:", err)
				tx.Rollback()
				continue
			}
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			log.Println("Error committing transaction:", err)
		}
	}
}

func main() {
	initDB()

	go expirePendingReservations() // Run the cron job as a background goroutine

	// Other application logic here...

	// Prevent the main function from exiting
	select {}
}
