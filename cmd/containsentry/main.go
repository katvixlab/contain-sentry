package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/katvixlab/contain-sentry/cmd/containsentry/config"
	"github.com/katvixlab/contain-sentry/internal/compose"
	"github.com/katvixlab/contain-sentry/internal/dockerfile"
	"github.com/katvixlab/contain-sentry/internal/entities"
	"github.com/katvixlab/contain-sentry/internal/report"
	"go.uber.org/zap"
)

func main() {
	cfg, help, err := config.LoadApplicationSettings(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load application settings: %v\n", err)
		os.Exit(1)
	}
	if help {
		return
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

	var validate []entities.Finding
	switch strings.ToLower(strings.TrimSpace(cfg.Target)) {
	case "compose":
		project, err := compose.NewProject(ctx, cfg.ComposeFiles)
		if err != nil {
			log.Fatal("Failed to create Compose project", zap.Error(err), zap.Strings("compose_files", cfg.ComposeFiles))
		}

		findings, err := project.Validate(ctx, rules)
		if err != nil {
			log.Fatal("Failed to validate Compose project", zap.Error(err))
		}
		validate = findings
	default:
		df, err := dockerfile.NewDockerfile(ctx, cfg.DockerfilePath)
		if err != nil {
			log.Fatal("Failed to create Dockerfile", zap.Error(err), zap.String("dockerfile", cfg.DockerfilePath))
		}

		findings, err := df.Validate(ctx, rules)
		if err != nil {
			log.Fatal("Failed to validate Dockerfile", zap.Error(err))
		}
		validate = findings
	}

	for _, finding := range validate {
		log.Info(fmt.Sprintf("[%s][%s] %s | code=%q | location=%v", finding.Severity, finding.ID, finding.Name, finding.CodeSample, finding.Location))
	}
	log.Info(fmt.Sprintf("Total findings: %d", len(validate)))

	if strings.TrimSpace(cfg.ReportJSONPath) != "" {
		if err := report.WriteJSON(cfg.ReportJSONPath, report.Build(validate)); err != nil {
			log.Fatal("Failed to write JSON report", zap.Error(err), zap.String("report_json", cfg.ReportJSONPath))
		}
		log.Info("JSON report written", zap.String("report_json", cfg.ReportJSONPath))
	}
}
