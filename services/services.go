package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected user
type Client struct {
	ID       string
	Conn     *websocket.Conn
	StreamID string
	UserType string // "seller" or "viewer"
	Send     chan []byte
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	Clients    map[string]*Client
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

// Message types for WebSocket communication
type WSMessage struct {
	Type      string      `json:"type"`
	StreamID  string      `json:"stream_id"`
	UserID    string      `json:"user_id"`
	UserType  string      `json:"user_type"`
	Content   string      `json:"content,omitempty"`
	Product   interface{} `json:"product,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

var Hub_Instance = &Hub{
	Clients:    make(map[string]*Client),
	Broadcast:  make(chan []byte),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
}

func init() {
	go Hub_Instance.run()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client %s connected to stream %s", client.ID, client.StreamID)

			// Notify others about new viewer
			if client.UserType == "viewer" {
				message := WSMessage{
					Type:      "viewer_joined",
					StreamID:  client.StreamID,
					UserID:    client.ID,
					UserType:  client.UserType,
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(message)
				h.broadcastToStream(client.StreamID, data)
			}

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
				log.Printf("Client %s disconnected from stream %s", client.ID, client.StreamID)
			}
			h.mu.Unlock()

			// Notify others about viewer leaving
			if client.UserType == "viewer" {
				message := WSMessage{
					Type:      "viewer_left",
					StreamID:  client.StreamID,
					UserID:    client.ID,
					UserType:  client.UserType,
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(message)
				h.broadcastToStream(client.StreamID, data)
			}

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToStream(streamID string, message []byte) {
	h.broadcastToStream(streamID, message)
}

func (h *Hub) broadcastToStream(streamID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.Clients {
		if client.StreamID == streamID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client.ID)
			}
		}
	}
}

func (h *Hub) GetStreamViewers(streamID string) (int, []string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	viewerCount := 0
	viewers := []string{}

	for _, client := range h.Clients {
		if client.StreamID == streamID && client.UserType == "viewer" {
			viewerCount++
			viewers = append(viewers, client.ID)
		}
	}

	return viewerCount, viewers
}

func (c *Client) readPump() {
	defer func() {
		Hub_Instance.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var message WSMessage
		if err := json.Unmarshal(messageData, &message); err != nil {
			continue
		}

		message.UserID = c.ID
		message.UserType = c.UserType
		message.StreamID = c.StreamID
		message.Timestamp = time.Now()

		// Process different message types
		switch message.Type {
		case "chat":
			data, _ := json.Marshal(message)
			Hub_Instance.broadcastToStream(c.StreamID, data)

		case "product_showcase":
			if c.UserType == "seller" {
				data, _ := json.Marshal(message)
				Hub_Instance.broadcastToStream(c.StreamID, data)
			}

		case "purchase_intent":
			if c.UserType == "viewer" {
				data, _ := json.Marshal(message)
				Hub_Instance.broadcastToStream(c.StreamID, data)
			}

		case "special_offer":
			if c.UserType == "seller" {
				data, _ := json.Marshal(message)
				Hub_Instance.broadcastToStream(c.StreamID, data)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func NewClient(conn *websocket.Conn, userID, streamID, userType string) *Client {
	client := &Client{
		ID:       userID,
		Conn:     conn,
		StreamID: streamID,
		UserType: userType,
		Send:     make(chan []byte, 256),
	}

	Hub_Instance.Register <- client

	go client.writePump()
	go client.readPump()

	return client
}
