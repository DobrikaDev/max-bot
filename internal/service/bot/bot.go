package bot

import (
	"DobrikaDev/max-bot/internal/service/bot/handlers"
	"DobrikaDev/max-bot/utils/config"
	"context"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
	"go.uber.org/zap"
)

type Bot struct {
	ctx            context.Context
	cfg            *config.Config
	logger         *zap.Logger
	api            *maxbot.Api
	messageHandler *handlers.MessageHandler
}

func NewBot(ctx context.Context, cfg *config.Config, logger *zap.Logger) *Bot {
	logger.Info("Creating bot API")
	api, err := maxbot.New(cfg.MaxToken)
	if err != nil {
		logger.Panic("failed to create bot API", zap.Error(err))
	}

	return &Bot{ctx: ctx, cfg: cfg, api: api, logger: logger, messageHandler: handlers.NewMessageHandler(api, cfg, logger)}
}

func (b *Bot) Start() {
	u := b.api.GetUpdates(b.ctx)

	for update := range u {
		switch update := update.(type) {
		case *schemes.MessageCreatedUpdate:
			b.messageHandler.HandleMessage(b.ctx, update)
		case *schemes.MessageCallbackUpdate:
			b.messageHandler.HandleCallbackQuery(b.ctx, update)
		}
	}
}
