package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/helldweller/tgbot-spellcheck/internal/bot"
	"github.com/helldweller/tgbot-spellcheck/internal/config"
	"github.com/helldweller/tgbot-spellcheck/internal/openai"
	"github.com/helldweller/tgbot-spellcheck/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		log.Printf("fatal error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config) error {
	store := storage.NewInMemoryStore()
	openAIClient := openai.NewClient(cfg.OpenAIKey, cfg.OpenAIModel)

	b, err := bot.New(cfg, openAIClient, store)
	if err != nil {
		return err
	}

	return b.Run(ctx)
}
