package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ----------------------------------------------------------------------
// 1. DATA MODELS
// ----------------------------------------------------------------------

// SeatStatus indicates whether a seat is available, reserved, or booked.
type SeatStatus string

const (
	StatusAvailable SeatStatus = "available"
	StatusReserved  SeatStatus = "reserved"
	StatusBooked    SeatStatus = "booked"
)

// Seat represents a seat within an event.
type Seat struct {
	ID        int64      `json:"id"`
	Row       int        `json:"row"`
	Number    int        `json:"number"`
	Status    SeatStatus `json:"status"`
	EventID   int64      `json:"event_id"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Event represents a show or performance for which seats can be booked.
type Event struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Venue     string    `json:"venue"`
	StartTime time.Time `json:"start_time"`
}

// Reservation tracks a user's hold on a specific seat.
type Reservation struct {
	ID        int64     `json:"id"`
	SeatID    int64     `json:"seat_id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // e.g. "active", "completed", "cancelled", "expired"
}

// ----------------------------------------------------------------------
// 2. GLOBAL MOCK STORAGE
// ----------------------------------------------------------------------

var (
	mu sync.Mutex

	mockEvents       = make(map[int64]*Event)
	mockSeats        = make(map[int64]*Seat)
	mockReservations = make(map[int64]*Reservation)

	eventIDCounter       int64 = 1
	seatIDCounter        int64 = 1
	reservationIDCounter int64 = 1
)

// ----------------------------------------------------------------------
// 3. ERROR TYPES
// ----------------------------------------------------------------------

var (
	ErrSeatNotFound        = &SeatMapError{"seat not found"}
	ErrSeatNotAvailable    = &SeatMapError{"seat not available"}
	ErrSeatNotReserved     = &SeatMapError{"seat not reserved"}
	ErrReservationNotFound = &SeatMapError{"reservation not found"}
	ErrReservationExpired  = &SeatMapError{"reservation expired"}
)

type SeatMapError struct {
	Message string
}

func (e *SeatMapError) Error() string {
	return e.Message
}

// ----------------------------------------------------------------------
// 4. INIT() -> SEED DUMMY DATA
// ----------------------------------------------------------------------

func init() {
	// Create a sample event.
	e := &Event{
		ID:        eventIDCounter,
		Name:      "Rock Concert 2025",
		Venue:     "Mega Stadium",
		StartTime: time.Now().Add(24 * time.Hour), // tomorrow
	}
	mockEvents[e.ID] = e
	eventIDCounter++

	// Create 5 seats for this event.
	for i := 1; i <= 5; i++ {
		s := &Seat{
			ID:      seatIDCounter,
			Row:     1,
			Number:  i,
			Status:  StatusAvailable,
			EventID: e.ID,
		}
		mockSeats[s.ID] = s
		seatIDCounter++
	}
}

// ----------------------------------------------------------------------
// 5. IN-MEMORY DB FUNCTIONS
// ----------------------------------------------------------------------

func GetAllSeatsForEvent(eventID int64) []*Seat {
	mu.Lock()
	defer mu.Unlock()

	var result []*Seat
	for _, seat := range mockSeats {
		if seat.EventID == eventID {
			copySeat := *seat
			result = append(result, &copySeat)
		}
	}
	return result
}

// ReserveSeat attempts to reserve a seat if it is available.
func ReserveSeat(seatID, userID int64, duration time.Duration) error {
	mu.Lock()
	defer mu.Unlock()

	seat, found := mockSeats[seatID]
	if !found {
		return ErrSeatNotFound
	}
	if seat.Status != StatusAvailable {
		return ErrSeatNotAvailable
	}

	// Create a new reservation
	r := &Reservation{
		ID:        reservationIDCounter,
		SeatID:    seatID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
		Status:    "active",
	}
	mockReservations[r.ID] = r
	reservationIDCounter++

	// Update the seat status to reserved
	seat.Status = StatusReserved
	seat.UpdatedAt = time.Now()

	return nil
}

// BookSeat finalizes the purchase if the seat is still reserved by that user.
func BookSeat(seatID, userID int64) error {
	mu.Lock()
	defer mu.Unlock()

	seat, found := mockSeats[seatID]
	if !found {
		return ErrSeatNotFound
	}
	if seat.Status != StatusReserved {
		return ErrSeatNotReserved
	}

	// Find the active reservation for this seat/user
	var res *Reservation
	for _, r := range mockReservations {
		if r.SeatID == seatID && r.UserID == userID && r.Status == "active" {
			res = r
			break
		}
	}
	if res == nil {
		return ErrReservationNotFound
	}

	if time.Now().After(res.ExpiresAt) {
		// Mark seat back to available if expired
		seat.Status = StatusAvailable
		seat.UpdatedAt = time.Now()
		res.Status = "expired"
		return ErrReservationExpired
	}

	// Mark seat as booked
	seat.Status = StatusBooked
	seat.UpdatedAt = time.Now()

	// Mark reservation as completed
	res.Status = "completed"
	return nil
}

// ----------------------------------------------------------------------
// 6. HANDLERS
// ----------------------------------------------------------------------

// getSeatsByEventHandler: GET /events/{eventID}/seats
func getSeatsByEventHandler(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	// Expect: ["events", "{eventID}", "seats"]
	if len(parts) < 3 {
		http.Error(w, "invalid path; expected /events/{id}/seats", http.StatusBadRequest)
		return
	}

	eventIDStr := parts[1]
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid event ID", http.StatusBadRequest)
		return
	}

	seats := GetAllSeatsForEvent(eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(seats)
}

// reserveSeatHandler: POST /seats/{seatID}/reserve
func reserveSeatHandler(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	// Expect: ["seats", "{seatID}", "reserve"]
	if len(parts) < 3 {
		http.Error(w, "invalid path; expected /seats/{id}/reserve", http.StatusBadRequest)
		return
	}

	seatIDStr := parts[1]
	seatID, err := strconv.ParseInt(seatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid seat ID", http.StatusBadRequest)
		return
	}

	// In a real app, you'd parse JWT/cookie for user info. We'll mock userID=1
	userID := int64(1)

	// Optional: read 'duration' from query params
	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "300" // default 300 seconds (5 mins)
	}
	durationSec, _ := strconv.Atoi(durationStr)
	duration := time.Duration(durationSec) * time.Second

	if err := ReserveSeat(seatID, userID, duration); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Seat reserved successfully."))
}

// bookSeatHandler: POST /seats/{seatID}/book
func bookSeatHandler(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	// Expect: ["seats", "{seatID}", "book"]
	if len(parts) < 3 {
		http.Error(w, "invalid path; expected /seats/{id}/book", http.StatusBadRequest)
		return
	}

	seatIDStr := parts[1]
	seatID, err := strconv.ParseInt(seatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid seat ID", http.StatusBadRequest)
		return
	}

	// Mock user ID = 1
	userID := int64(1)

	if err := BookSeat(seatID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Seat booked successfully."))
}

// splitPath is a helper that trims the leading slash and splits by "/".
func splitPath(p string) []string {
	if len(p) == 0 {
		return nil
	}
	if p[0] == '/' {
		p = p[1:]
	}
	return splitOnSlash(p)
}

func splitOnSlash(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// ----------------------------------------------------------------------
// 7. CORS MIDDLEWARE
// ----------------------------------------------------------------------

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow any domain for demo; restrict in production
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			// Preflight request
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ----------------------------------------------------------------------
// 8. MAIN
// ----------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()

	// GET /events/{id}/seats
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests for seats listing
		if r.Method == http.MethodGet {
			getSeatsByEventHandler(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// POST /seats/{id}/reserve, POST /seats/{id}/book
	mux.HandleFunc("/seats/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			parts := splitPath(r.URL.Path)
			if len(parts) == 3 {
				action := parts[2]
				switch action {
				case "reserve":
					reserveSeatHandler(w, r)
					return
				case "book":
					bookSeatHandler(w, r)
					return
				}
			}
		}
		http.NotFound(w, r)
	})

	// Wrap with CORS
	corsWrappedMux := corsMiddleware(mux)

	addr := ":8080"
	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(addr, corsWrappedMux))
}
