package controllers

import (
	"LiveStreamService/services"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket handler for live shopping stream
func HandleLiveStreamWebSocket(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	streamID := params["stream_id"]
	userID := r.URL.Query().Get("user_id")
	userType := r.URL.Query().Get("user_type")

	if streamID == "" || userID == "" || userType == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	services.NewClient(conn, userID, streamID, userType)
}

// Get active viewers count for a stream
func GetStreamViewers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	streamID := params["stream_id"]

	viewerCount, viewers := services.Hub_Instance.GetStreamViewers(streamID)

	response := map[string]interface{}{
		"stream_id":    streamID,
		"viewer_count": viewerCount,
		"viewers":      viewers,
	}

	json.NewEncoder(w).Encode(response)
}

// Send announcement to stream
func SendStreamAnnouncement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	streamID := params["stream_id"]

	var announcement struct {
		Message  string `json:"message"`
		UserID   string `json:"user_id"`
		UserType string `json:"user_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&announcement); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if announcement.UserType != "seller" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Only sellers can send announcements"})
		return
	}

	message := services.WSMessage{
		Type:     "announcement",
		StreamID: streamID,
		UserID:   announcement.UserID,
		UserType: announcement.UserType,
		Content:  announcement.Message,
	}

	data, _ := json.Marshal(message)
	services.Hub_Instance.BroadcastToStream(streamID, data)

	json.NewEncoder(w).Encode(map[string]string{"message": "Announcement sent successfully"})
}
