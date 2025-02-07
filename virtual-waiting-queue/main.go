package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// Redis client and context.
	redisClient     *redis.Client
	ctx             = context.Background()
	waitingQueueKey = "waiting_queue"
)

// HTML templates for our pages.
const indexHTML = `
<!DOCTYPE html>
<html>
<head>
	<title>Virtual Waiting Queue</title>
</head>
<body>
	<h1>Welcome to the Virtual Waiting Queue Demo</h1>
	<p><a href="/join">Join the Queue</a></p>
</body>
</html>
`

const statusHTML = `
<!DOCTYPE html>
<html>
<head>
	<title>Queue Status</title>
	{{if not .IsFirst}}
	<meta http-equiv="refresh" content="5">
	{{end}}
</head>
<body>
	<h1>Queue Status</h1>
	<p>Your position in the queue: {{.Position}} out of {{.Total}}</p>
	{{if .IsFirst}}
		<p>You are next in line!</p>
		<form action="/serve" method="post">
			<button type="submit">Proceed</button>
		</form>
	{{else}}
		<p>Please wait until it's your turn...</p>
	{{end}}
</body>
</html>
`

const serveHTML = `
<!DOCTYPE html>
<html>
<head>
	<title>Service Page</title>
</head>
<body>
	<h1>Welcome to the Ticket Purchase Page</h1>
	<p>You are now being served. Enjoy the event!</p>
</body>
</html>
`

func main() {
	// Initialize Redis client.
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test the Redis connection.
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Cannot connect to Redis:", err)
	}

	// Set up HTTP handlers.
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/join", joinHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/serve", serveHandler)

	fmt.Println("Server starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// generateUserID creates a random 16-byte hex string.
func generateUserID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to using the current time.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// indexHandler shows a welcome page and sets a unique user ID cookie if not present.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("user_id")
	if err != nil {
		uid := generateUserID()
		http.SetCookie(w, &http.Cookie{
			Name:    "user_id",
			Value:   uid,
			Path:    "/",
			Expires: time.Now().Add(24 * time.Hour),
		})
	}
	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// joinHandler adds the user to the waiting queue if they are not already in it.
func joinHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	userID := cookie.Value

	// Check if the user is already in the queue.
	_, err = redisClient.ZScore(ctx, waitingQueueKey, userID).Result()
	if err == redis.Nil {
		// Not in the queue; add with the current timestamp as the score.
		now := float64(time.Now().UnixNano())
		if err := redisClient.ZAdd(ctx, waitingQueueKey, &redis.Z{
			Score:  now,
			Member: userID,
		}).Err(); err != nil {
			http.Error(w, "Error joining queue", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Error checking queue", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s joined the queue", userID)

	// Redirect to the status page.
	http.Redirect(w, r, "/status", http.StatusSeeOther)
}

// statusHandler shows the user's current position in the queue.
// The page auto-refreshes every 5 seconds unless the user is first.
func statusHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	userID := cookie.Value

	rank, err := redisClient.ZRank(ctx, waitingQueueKey, userID).Result()
	if err == redis.Nil {
		// If the user isn’t in the queue, redirect them to join.
		http.Redirect(w, r, "/join", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, "Error checking queue status", http.StatusInternalServerError)
		return
	}

	// Get the total number of users in the queue.
	total, err := redisClient.ZCard(ctx, waitingQueueKey).Result()
	if err != nil {
		http.Error(w, "Error checking queue count", http.StatusInternalServerError)
		return
	}

	data := struct {
		Position int64
		Total    int64
		IsFirst  bool
	}{
		Position: rank + 1, // rank is zero-based.
		Total:    total,
		IsFirst:  rank == 0,
	}

	tmpl, err := template.New("status").Parse(statusHTML)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// serveHandler allows the user to proceed only if they are first in line.
// It uses a Lua script to atomically check if the user is at the front and remove them.
func serveHandler(w http.ResponseWriter, r *http.Request) {
	// For safety, only allow POST requests.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	userID := cookie.Value

	// Lua script:
	// 1. Get the first member of the sorted set.
	// 2. If it matches the current user, remove it.
	luaScript := `
	local first = redis.call("ZRANGE", KEYS[1], 0, 0)[1]
	if first == ARGV[1] then
		return redis.call("ZREM", KEYS[1], ARGV[1])
	else
		return 0
	end
	`
	res, err := redisClient.Eval(ctx, luaScript, []string{waitingQueueKey}, userID).Result()
	if err != nil {
		http.Error(w, "Error processing queue", http.StatusInternalServerError)
		return
	}
	// If the result is 0, the user is not first.
	if res.(int64) == 0 {
		http.Redirect(w, r, "/status", http.StatusSeeOther)
		return
	}

	// Render the protected (service) page.
	tmpl, err := template.New("serve").Parse(serveHTML)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}
