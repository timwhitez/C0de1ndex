package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	OpenAIApiKey  string
	OpenAIBaseURL string
	DefaultModel  string
	DeepModel     string
}

// LoadConfig loads configuration from a .env file
func LoadConfig() (*Config, error) {
	// Load .env file from the current directory
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, we can rely on environment variables
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return &Config{
		OpenAIApiKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
		DefaultModel:  os.Getenv("DEFAULT_MODEL"),
		DeepModel:     os.Getenv("DEEP_MODEL"),
	}, nil
}
