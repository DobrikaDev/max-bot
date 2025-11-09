package main

import (
	"DobrikaDev/max-bot/di"
	"DobrikaDev/max-bot/utils/config"
	"DobrikaDev/max-bot/utils/logger"
	"context"
)

func main() {
	ctx := context.Background()
	cfg := config.MustLoadConfigFromFile("deployments/config.yaml")
	logger, _ := logger.NewLogger()
	defer logger.Sync()

	container := di.NewContainer(ctx, cfg, logger)

	go func() {
		logger.Info("Starting MAX bot")

		container.GetBot().Start()
	}()

	<-ctx.Done()
}
