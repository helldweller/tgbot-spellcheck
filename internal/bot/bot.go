package bot

import (
	"context"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/helldweller/tgbot-spellcheck/internal/config"
	"github.com/helldweller/tgbot-spellcheck/internal/openai"
	"github.com/helldweller/tgbot-spellcheck/internal/ratelimit"
	"github.com/helldweller/tgbot-spellcheck/internal/storage"
)

const processedMarker = "[spellchecked]" // текстовая метка в конце сообщения

// Bot инкапсулирует всю бизнес-логику модерации.
type Bot struct {
	cfg       config.Config
	telegram  *tgbotapi.BotAPI
	openai    openai.Client
	limiter   ratelimit.Limiter
	store     storage.Store
}

func New(cfg config.Config, oaClient openai.Client, store storage.Store) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, err
	}

	botAPI.Debug = false

	limiter := ratelimit.NewIntervalLimiter(cfg.MinInterval)

	return &Bot{
		cfg:      cfg,
		telegram: botAPI,
		openai:   oaClient,
		limiter:  limiter,
		store:    store,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.telegram.GetUpdatesChan(u)

	log.Printf("bot started, listening for updates")

	for {
		select {
		case <-ctx.Done():
			log.Printf("bot: context cancelled, stopping")
			return nil
		case update := <-updates:
			log.Printf("incoming update: update_id=%d", update.UpdateID)
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	msg := update.ChannelPost
	if msg == nil {
		// для простоты сейчас обрабатываем только посты в каналах
		log.Printf("skip update %d: no channel post", update.UpdateID)
		return
	}

	if msg.Chat == nil || msg.Chat.ID != b.cfg.ChannelID {
		if msg.Chat != nil {
			log.Printf("skip message %d: unexpected chat_id=%d", msg.MessageID, msg.Chat.ID)
		} else {
			log.Printf("skip message %d: nil chat", msg.MessageID)
		}
		return
	}

	if msg.From != nil && msg.From.IsBot {
		// не обрабатываем сообщения ботов (включая себя)
		log.Printf("skip message %d: from bot (user_id=%d)", msg.MessageID, msg.From.ID)
		return
	}

	if msg.Text == "" && msg.Caption == "" {
		log.Printf("skip message %d: empty text/caption", msg.MessageID)
		return
	}

	if b.store.WasProcessed(msg.Chat.ID, msg.MessageID) {
		log.Printf("skip message %d: already marked processed in store", msg.MessageID)
		return
	}

	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	if text == "" {
		log.Printf("skip message %d: resolved text is empty", msg.MessageID)
		return
	}

	if containsMarker(text) {
		// уже помечено
		log.Printf("message %d already contains processed marker, marking in store", msg.MessageID)
		b.store.MarkProcessed(msg.Chat.ID, msg.MessageID)
		return
	}

	now := time.Now()
	if !b.limiter.Allow(now) {
		log.Printf("rate limit hit for chat_id=%d at %s", msg.Chat.ID, now.Format(time.RFC3339))
		b.notifyRateLimited(msg.Chat.ID)
		b.store.MarkProcessed(msg.Chat.ID, msg.MessageID)
		return
	}

	log.Printf("processing message %d in chat_id=%d", msg.MessageID, msg.Chat.ID)
	go b.processMessage(ctx, msg.Chat.ID, msg.MessageID, text)
}

func (b *Bot) processMessage(parentCtx context.Context, chatID int64, messageID int, text string) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	log.Printf("processMessage start: chat_id=%d message_id=%d", chatID, messageID)

	corrected, err := b.openai.CorrectText(ctx, text)
	if err != nil {
		log.Printf("openai error for message %d: %v", messageID, err)
		return
	}

	correctedWithMarker := corrected + "\n" + processedMarker

	// удаляем исходный пост
	deleteMsg := tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}
	if _, err := b.telegram.Request(deleteMsg); err != nil {
		log.Printf("failed to delete original message: %v", err)
	}

	// публикуем исправленный текст
	msg := tgbotapi.NewMessage(chatID, correctedWithMarker)
	// без HTML-комментариев, чтобы не ломать парсер Telegram

	if _, err := b.telegram.Send(msg); err != nil {
		log.Printf("failed to send corrected message for message %d: %v", messageID, err)
		return
	}

	log.Printf("processMessage done: chat_id=%d message_id=%d", chatID, messageID)
	b.store.MarkProcessed(chatID, messageID)
}

func (b *Bot) notifyRateLimited(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Лимит проверки орфографии исчерпан. Попробуйте опубликовать новый пост чуть позже.")
	if _, err := b.telegram.Send(msg); err != nil {
		log.Printf("failed to send rate limit notification: %v", err)
	}
}

func containsMarker(text string) bool {
	return len(text) >= len(processedMarker) && (text == processedMarker || contains(text, processedMarker))
}

func contains(s, substr string) bool {
	// простая обертка, чтобы не тянуть strings для одной операции
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
