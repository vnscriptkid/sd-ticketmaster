package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSE Manager to handle multiple clients
type SSEManager struct {
	clients map[chan string]bool
	mu      sync.Mutex
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[chan string]bool),
	}
}

// AddClient adds a new client to the SSE manager
func (m *SSEManager) AddClient(client chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client] = true
}

// RemoveClient removes a client from the SSE manager
func (m *SSEManager) RemoveClient(client chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, client)
	close(client)
}

// Broadcast sends messages to all connected clients
func (m *SSEManager) Broadcast(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for client := range m.clients {
		client <- message
	}
}

var sseManager = NewSSEManager()

func main() {
	r := gin.Default()

	// Serve the HTML file
	r.StaticFile("/", "./index.html")

	// SSE endpoint
	r.GET("/events", func(c *gin.Context) {
		clientChan := make(chan string)
		sseManager.AddClient(clientChan)

		c.Stream(func(w io.Writer) bool {
			if msg, ok := <-clientChan; ok {
				c.SSEvent("ticket_update", msg)
				return true
			}
			return false
		})

		defer sseManager.RemoveClient(clientChan)
	})

	// Mock ticket update in the background
	go func() {
		for {
			// Simulate ticket availability changes
			availableTickets := fmt.Sprintf("%d tickets available", time.Now().Unix()%100)
			sseManager.Broadcast(availableTickets)
			time.Sleep(5 * time.Second)
		}
	}()

	log.Fatal(r.Run(":8080"))
}
