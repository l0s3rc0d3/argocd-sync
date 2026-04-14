package main

import (
	"log/slog"
	"os"

	"github.com/l0s3rc0d3/argocd-sync/app/internal/argocd"
	"github.com/l0s3rc0d3/argocd-sync/app/internal/config"
)

func parseLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LoggingLevel),
	}))

	logger.Info("configuration loaded",
		"ARGOCD_URL", cfg.ArgocdURL,
		"ARGOCD_APP_NAME", cfg.ArgocdAppName,
		"LOGGING_LEVEL", cfg.LoggingLevel,
	)

	if cfg.ArgocdInsecure {
		logger.Warn("TLS certificate verification disabled", "ARGOCD_INSECURE", true)
	}

	client := argocd.NewClient(cfg.ArgocdURL, cfg.ArgocdToken, cfg.ArgocdInsecure)

	app, err := client.GetApplication(cfg.ArgocdAppName)
	if err != nil {
		logger.Error("failed to get application", "error", err)
		os.Exit(1)
	}

	logger.Info("application retrieved", "app", app)
}