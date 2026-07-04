package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken                string
	AiAPIKey                string
	ModelName               string
	SystemPrompt            string
	ResponseCriteria        string
	IsResponseRequiredModel string
}

func Load() *Config {
	godotenv.Load()
	return &Config{
		BotToken:                mustGet("DISCORD_BOT_TOKEN"),
		AiAPIKey:                mustGet("AI_API"),
		ModelName:               mustGet("AI_MODEL"),
		SystemPrompt:            mustGet("SYSTEM_PROMPT_V1"),
		ResponseCriteria:        mustGet("RESPONSE_CRITERIA_LATEST"),
		IsResponseRequiredModel: mustGet("IS_RESPONSE_REQUIRED_MODEL"),
	}
}

func mustGet(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatal("Could not load env of key=", key)
	}
	return val
}
