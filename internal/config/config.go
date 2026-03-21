package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment              string
	Port                     int
	MCPPort                  int
	LogLevel                 string
	MongoURI                 string
	RedisURI                 string
	GeminiAPIKey             string
	VeoAPIKey                string
	NanaBananaKey            string
	WorkerPoolSize           int
	RateLimit                int
	IdentityImage            string
	GoogleCloudProject       string
	GoogleCloudProjectID     string
	GoogleCredentials        string
	
	// AWS S3 Configuration
	AWSAccessKey             string
	AWSSecretKey             string
	AWSBucket                string
	AWSRegion                string
	
	// Free Content Sources
	RSSFeeds                 []string
	NitterInstance           string
	NitterAccounts           []string
	RedditSubreddits         []string
	NewsAPIKey               string
}

func Load() *Config {
	return &Config{
		Environment:          getEnv("ENVIRONMENT", "development"),
		Port:                 getEnvAsInt("PORT", 8080),
		MCPPort:              getEnvAsInt("MCP_PORT", 8081),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		MongoURI:             getEnv("MONGO_URI", "mongodb://localhost:27017/newsroom"),
		RedisURI:             getEnv("REDIS_URI", "redis://localhost:6379"),
		GeminiAPIKey:         getEnv("GEMINI_API_KEY", ""),
		VeoAPIKey:            getEnv("VEO_API_KEY", ""),
		NanaBananaKey:        getEnv("NANO_BANANA_KEY", ""),
		WorkerPoolSize:       getEnvAsInt("WORKER_POOL_SIZE", 10),
		RateLimit:            getEnvAsInt("RATE_LIMIT", 100),
		IdentityImage:        getEnv("IDENTITY_IMAGE_URL", ""),
		GoogleCloudProject:   getEnv("GOOGLE_CLOUD_PROJECT_ID", ""),
		GoogleCloudProjectID: getEnv("GOOGLE_CLOUD_PROJECT_ID", ""),
		GoogleCredentials:    getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		
		// AWS S3 Configuration
		AWSAccessKey:         getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:         getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSBucket:            getEnv("AWS_BUCKET", ""),
		AWSRegion:            getEnv("AWS_REGION", "eu-west-2"),
		
		// Free Content Sources
		RSSFeeds:             getEnvAsSlice("RSS_FEEDS", ","),
		NitterInstance:       getEnv("NITTER_INSTANCE", "https://nitter.net"),
		NitterAccounts:       getEnvAsSlice("NITTER_ACCOUNTS", ","),
		RedditSubreddits:     getEnvAsSlice("REDDIT_SUBREDDITS", ","),
		NewsAPIKey:           getEnv("NEWSAPI_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key, separator string) []string {
	value := os.Getenv(key)
	if value == "" {
		return []string{}
	}
	
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}