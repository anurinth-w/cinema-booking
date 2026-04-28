package services

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/cinema-booking/backend/models"
	"github.com/gorilla/websocket"
)

// Client represents a single WebSocket connection
type Client struct {
	Conn       *websocket.Conn
	ShowtimeID string
	Send       chan []byte
}

// WSHub manages all connected WebSocket clients
type WSHub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*Client]struct{}),
	}
}

func (h *WSHub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	log.Printf("[WS] Client registered for showtime %s (total: %d)", c.ShowtimeID, len(h.clients))
}

func (h *WSHub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.Send)
	log.Printf("[WS] Client unregistered (total: %d)", len(h.clients))
}

// Broadcast sends a WSMessage to all clients watching the same showtime
func (h *WSHub) Broadcast(msg models.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ShowtimeID != msg.ShowtimeID {
			continue
		}
		select {
		case client.Send <- data:
		default:
			// Client buffer full — skip
		}
	}
}

// WritePump pumps messages from the Send channel to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[WS] Write error: %v", err)
			return
		}
	}
}

// ReadPump keeps the connection alive and handles client disconnection
func (c *Client) ReadPump(hub *WSHub) {
	defer hub.Unregister(c)
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
