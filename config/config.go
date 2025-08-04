package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config struct holds all configuration values needed by the application.
// The struct tags (mapstructure) tell Viper how to map environment variables to struct fields.
type Config struct {
	DBSource            string        	`mapstructure:"DB_SOURCE"`             	// Database connection string
	ServerAddress       string        	`mapstructure:"SERVER_ADDRESS"`        	// Address where the server will run (e.g., "localhost:8080")
	TokenSymmetricKey   string        	`mapstructure:"TOKEN_SYMMETRIC_KEY"`   	// Secret key for signing tokens
	AccessTokenDuration time.Duration 	`mapstructure:"ACCESS_TOKEN_DURATION"` 	// Duration tokens will remain valid (e.g., "15m", "1h")
	GeminiAPIURL		string			`mapstructure:"GEMINI_API_URL"`
	GeminiAPIKey        string        	`mapstructure:"GEMINI_API_KEY"`        	// API key for accessing Gemini (or any external service)
	RecommenderAPIURL	string			`mapstructure:"RECOMMENDER_API_KEY"`
	RecommenderAPIKey	string			`mapstructure:"RECOMMENDER_API_KEY"`	// API key for accessing Recommendations
	FrontendURL			string			`mapstructure:"FRONTEND_URL"`
}

// LoadConfig loads environment variables from a file and environment into the Config struct
func LoadConfig(path string) (config Config, err error) {
	// Add the directory where the config file is located
	viper.AddConfigPath(path)

	// Specify the name of the config file (without extension)
	viper.SetConfigName("app")

	// Specify the file type. In this case, we're using a .env-style file
	viper.SetConfigType("env")

	// Automatically read in any environment variables that match the keys
	viper.AutomaticEnv()

	// Read the config file
	err = viper.ReadInConfig()
	if err != nil {
		// If there's an error reading the file, return immediately with the error
		return
	}

	// Unmarshal the config values into the Config struct
	err = viper.Unmarshal(&config)

	// Return the filled config struct and any error encountered during unmarshaling
	return
}
