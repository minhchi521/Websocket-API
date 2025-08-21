package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddress string
	MongoDBURI    string
	DatabaseName  string
	CORSOrigins   string
}

func LoadConfig() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Không bắt buộc phải có .env file
	}

	return &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8081"),
		MongoDBURI:    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:  getEnv("DATABASE_NAME", "websocket_db"),
		CORSOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
