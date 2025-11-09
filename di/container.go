package di

import (
	"DobrikaDev/max-bot/internal/service/bot"
	"DobrikaDev/max-bot/utils/config"
	"context"
	"net/http"

	"go.uber.org/zap"
)

type Container struct {
	ctx        context.Context
	cfg        *config.Config
	logger     *zap.Logger
	bot        *bot.Bot
	httpClient *http.Client
}

func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) *Container {
	return &Container{ctx: ctx, cfg: cfg, logger: logger}
}

func (c *Container) GetBot() *bot.Bot {
	return get(&c.bot, func() *bot.Bot {
		return bot.NewBot(c.ctx, c.cfg, c.logger)
	})
}

func (c *Container) GetHTTPClient() *http.Client {
	return get(&c.httpClient, func() *http.Client {
		return http.DefaultClient
	})
}

func get[T comparable](obj *T, builder func() T) T {
	if *obj != *new(T) {
		return *obj
	}

	*obj = builder()
	return *obj
}
