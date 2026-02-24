package telegram

import (
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	maxRetries  = 3
	retryBaseMs = 2000
	retryGrowth = 2
)

// NewBot creates a Telegram bot with retry logic for transient network failures.
// The underlying tgbotapi.NewBotAPI call contacts api.telegram.org which can
// occasionally fail with TCP resets; retrying avoids a fatal startup failure.
func NewBot(token string, chatID int64, logger *slog.Logger) (*Bot, error) {
	var api *tgbotapi.BotAPI
	var err error

	delay := time.Duration(retryBaseMs) * time.Millisecond
	for attempt := 1; attempt <= maxRetries; attempt++ {
		api, err = tgbotapi.NewBotAPI(token)
		if err == nil {
			break
		}
		if attempt < maxRetries {
			logger.Warn("Telegram API connection failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("maxRetries", maxRetries),
				slog.Duration("retryIn", delay),
				slog.Any("error", err),
			)
			time.Sleep(delay)
			delay *= retryGrowth
		}
	}
	if err != nil {
		return nil, fmt.Errorf("after %d attempts: %w", maxRetries, err)
	}
	return &Bot{bot: api, chatID: chatID, logger: logger}, nil
}
