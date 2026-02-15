package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bonus2k/contain-sentry/cmd/containsentry/config"
	"github.com/bonus2k/contain-sentry/internal/dockerfile"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadApplicationSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load application settings: %v\n", err)
		os.Exit(1)
	}

	log := config.NewLogger(config.WithConfig(cfg.Logger))
	log.Info(
		"Application config loaded",
		zap.Any("config", cfg),
	)

	rules, err := loadRules(cfg.RulesPath)
	if err != nil {
		log.Fatal("Failed to load rules", zap.Error(err), zap.String("rules", cfg.RulesPath))
	}

	ctx := config.WithLogger(context.Background(), log)

	df, err := dockerfile.NewDockerfile(ctx, cfg.DockerfilePath)
	if err != nil {
		log.Fatal("Failed to create Dockerfile", zap.Error(err), zap.String("dockerfile", cfg.DockerfilePath))
	}

	validate, err := df.Validate(ctx, rules)
	if err != nil {
		log.Fatal("Failed to validate Dockerfile", zap.Error(err))
	}

	for _, finding := range validate {
		log.Info(fmt.Sprintf("[%s][%s] %s | code=%q | location=%v", finding.Severity, finding.ID, finding.Name, finding.CodeSample, finding.Location))
	}
	log.Info(fmt.Sprintf("Total findings: %d", len(validate)))
}
