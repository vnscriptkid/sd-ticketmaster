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
	Status    string    `json:"status"` // "active", "completed", "cancelled", "expired", etc.
}

// ----------------------------------------------------------------------
// 2. GLOBAL MOCK STORAGE & ERRORS
// ----------------------------------------------------------------------

var (
	mu                 sync.Mutex
	mockEvents               = make(map[int64]*Event)
	mockSeats                = make(map[int64]*Seat)
	mockReservations         = make(map[int64]*Reservation)
	eventIDCounter     int64 = 1
	seatIDCounter      int64 = 1
	reservationCounter int64 = 1
)

// Errors
var (
	ErrSeatNotFound        = &SeatMapError{"seat not found"}
	ErrSeatNotAvailable    = &SeatMapError{"seat not available"}
	ErrSeatNotReserved     = &SeatMapError{"seat not reserved"}
	ErrReservationNotFound = &SeatMapError{"reservation not found"}
	ErrReservationExpired  = &SeatMapError{"reservation expired"}
)

// SeatMapError is a simple custom error type.
type SeatMapError struct {
	Message string
}

func (e *SeatMapError) Error() string {
	return e.Message
}

// ----------------------------------------------------------------------
// 3. INIT() -> SEED DUMMY DATA
// ----------------------------------------------------------------------

func init() {
	// Create a sample event
	e := &Event{
		ID:        eventIDCounter,
		Name:      "Rock Concert 2025",
		Venue:     "Mega Stadium",
		StartTime: time.Now().Add(24 * time.Hour), // tomorrow
	}
	mockEvents[e.ID] = e
	eventIDCounter++

	// Create 5 seats for the above event
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
// 4. SSE MANAGER
// ----------------------------------------------------------------------

// SSEManager manages SSE subscribers for each event.
type SSEManager struct {
	mu          sync.Mutex
	subscribers map[int64][]chan string // eventID -> list of subscriber channels
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		subscribers: make(map[int64][]chan string),
	}
}

// Subscribe returns a channel on which the client will receive seatmap updates for a specific event.
func (m *SSEManager) Subscribe(eventID int64) <-chan string {
	ch := make(chan string, 1) // buffered channel to avoid blocking
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribers[eventID] = append(m.subscribers[eventID], ch)
	return ch
}

// Unsubscribe removes a channel from the SSEManager's subscriber list for the given event.
func (m *SSEManager) Unsubscribe(eventID int64, ch <-chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[eventID]
	for i, subscriber := range subs {
		if subscriber == ch {
			// Remove it from the slice
			subs = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	m.subscribers[eventID] = subs
}

// Broadcast sends a message to all subscribers of the given event.
func (m *SSEManager) Broadcast(eventID int64, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subs, ok := m.subscribers[eventID]; ok {
		for _, ch := range subs {
			select {
			case ch <- message:
			default:
				// If a subscriber's channel is blocked, skip it
			}
		}
	}
}

// ----------------------------------------------------------------------
// 5. IN-MEMORY SEAT OPERATIONS
// ----------------------------------------------------------------------

// GetAllSeatsForEvent returns all seats for a given event
func GetAllSeatsForEvent(eventID int64) []*Seat {
	mu.Lock()
	defer mu.Unlock()

	var seats []*Seat
	for _, seat := range mockSeats {
		if seat.EventID == eventID {
			copySeat := *seat
			seats = append(seats, &copySeat)
		}
	}
	return seats
}

// ReserveSeat attempts to reserve a seat if it is available
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
		ID:        reservationCounter,
		SeatID:    seatID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
		Status:    "active",
	}
	mockReservations[r.ID] = r
	reservationCounter++

	// Update seat status to reserved
	seat.Status = StatusReserved
	seat.UpdatedAt = time.Now()
	return nil
}

// BookSeat finalizes the purchase if the seat is still reserved by that user
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
// 6. HTTP HANDLERS
// ----------------------------------------------------------------------

// We'll use a global SSE manager
var sseManager = NewSSEManager()

// getSeatsByEventHandler -> GET /events/{eventID}/seats
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

// reserveSeatHandler -> POST /seats/{seatID}/reserve
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

	// For demo, userID=1
	userID := int64(1)

	// Duration from query param or 300s default
	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "300"
	}
	durationSec, _ := strconv.Atoi(durationStr)
	duration := time.Duration(durationSec) * time.Second

	if err := ReserveSeat(seatID, userID, duration); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// After reserving seat, broadcast updated seatmap for that event
	seat := getSeatByID(seatID)
	if seat != nil {
		broadcastSeatMap(seat.EventID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Seat reserved successfully."))
}

// bookSeatHandler -> POST /seats/{seatID}/book
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

	// For demo, userID=1
	userID := int64(1)

	if err := BookSeat(seatID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// After booking, broadcast updated seatmap
	seat := getSeatByID(seatID)
	if seat != nil {
		broadcastSeatMap(seat.EventID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Seat booked successfully."))
}

// sseEventStreamHandler -> GET /events/{eventID}/seats/stream
// This endpoint keeps the connection open and pushes event updates.
func sseEventStreamHandler(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	// Expect: ["events", "{eventID}", "seats", "stream"]
	if len(parts) < 4 {
		http.Error(w, "invalid path; expected /events/{id}/seats/stream", http.StatusBadRequest)
		return
	}
	eventIDStr := parts[1]
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid event ID", http.StatusBadRequest)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Keep the connection open
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe this client to SSE for the given event
	ch := sseManager.Subscribe(eventID)
	defer func() {
		// On exit, unsubscribe
		sseManager.Unsubscribe(eventID, ch)
	}()

	// Optionally, send an initial seatmap
	initialMap := getSeatMapJSON(eventID)
	if len(initialMap) > 0 {
		fmt.Fprintf(w, "data: %s\n\n", initialMap)
		flusher.Flush()
	}

	// Listen for seatmap updates in a loop
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected or request canceled
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// ----------------------------------------------------------------------
// 7. UTILITIES
// ----------------------------------------------------------------------

func getSeatMapJSON(eventID int64) string {
	seats := GetAllSeatsForEvent(eventID)
	data, _ := json.Marshal(seats)
	return string(data)
}

func broadcastSeatMap(eventID int64) {
	// Build JSON of seats for that event
	seatJSON := getSeatMapJSON(eventID)
	// Broadcast via SSE
	sseManager.Broadcast(eventID, seatJSON)
}

// getSeatByID is a simple helper to fetch a seat from the map by ID (locked).
func getSeatByID(seatID int64) *Seat {
	mu.Lock()
	defer mu.Unlock()
	seat, found := mockSeats[seatID]
	if !found {
		return nil
	}
	return seat
}

// Helper: split path into segments, ignoring leading "/"
func splitPath(p string) []string {
	if len(p) > 0 && p[0] == '/' {
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
// 8. CORS MIDDLEWARE
// ----------------------------------------------------------------------

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ----------------------------------------------------------------------
// 9. MAIN
// ----------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()

	// GET /events/{id}/seats -> List seats
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		// handle seats or SSE stream
		parts := splitPath(r.URL.Path)
		if r.Method == http.MethodGet {
			if len(parts) == 3 && parts[2] == "seats" {
				// GET /events/{id}/seats
				getSeatsByEventHandler(w, r)
				return
			}
			if len(parts) == 4 && parts[2] == "seats" && parts[3] == "stream" {
				// GET /events/{id}/seats/stream
				sseEventStreamHandler(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	// POST /seats/{id}/reserve or /seats/{id}/book
	mux.HandleFunc("/seats/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			parts := splitPath(r.URL.Path)
			if len(parts) >= 3 {
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

	wrappedMux := corsMiddleware(mux)

	addr := ":8080"
	log.Printf("Server listening at http://localhost:8080")
	log.Fatal(http.ListenAndServe(addr, wrappedMux))
}
