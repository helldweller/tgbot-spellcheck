package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	EnvTelegramToken   = "TELEGRAM_BOT_TOKEN"
	EnvOpenAIKey       = "OPENAI_API_KEY"
	EnvChannelID       = "TELEGRAM_CHANNEL_ID"
	EnvMinIntervalSecs = "MIN_INTERVAL_SECONDS" // e.g. 600 for 10 minutes
	EnvOpenAIModel     = "OPENAI_MODEL"         // optional, default set below
)

type Config struct {
	TelegramToken string
	OpenAIKey     string
	ChannelID     int64
	MinInterval   time.Duration
	OpenAIModel   string
}

func Load() Config {
	telegramToken := mustGetEnv(EnvTelegramToken)
	openAIKey := mustGetEnv(EnvOpenAIKey)

	channelIDStr := mustGetEnv(EnvChannelID)
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Fatalf("invalid %s: %v", EnvChannelID, err)
	}

	minIntervalStr := os.Getenv(EnvMinIntervalSecs)
	if minIntervalStr == "" {
		minIntervalStr = "600" // default 10 minutes
	}
	secs, err := strconv.Atoi(minIntervalStr)
	if err != nil || secs <= 0 {
		log.Fatalf("invalid %s: %v", EnvMinIntervalSecs, err)
	}

	model := os.Getenv(EnvOpenAIModel)
	if model == "" {
		model = "gpt-4.1-mini"
	}

	return Config{
		TelegramToken: telegramToken,
		OpenAIKey:     openAIKey,
		ChannelID:     channelID,
		MinInterval:   time.Duration(secs) * time.Second,
		OpenAIModel:   model,
	}
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
