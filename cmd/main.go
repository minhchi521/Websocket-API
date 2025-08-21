package main

import (
	"LiveStreamService/controllers"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	r := mux.NewRouter()

	// WebSocket routes
	r.HandleFunc("/ws/livestream/{stream_id}", controllers.HandleLiveStreamWebSocket)
	r.HandleFunc("/api/livestream/{stream_id}/viewers", controllers.GetStreamViewers).Methods("GET")
	r.HandleFunc("/api/livestream/{stream_id}/announcement", controllers.SendStreamAnnouncement).Methods("POST")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("WebSocket service is running"))
	}).Methods("GET")

	log.Printf("WebSocket server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
