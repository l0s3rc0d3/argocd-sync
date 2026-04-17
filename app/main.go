package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/l0s3rc0d3/argocd-sync/app/internal/argocd"
	"github.com/l0s3rc0d3/argocd-sync/app/internal/config"
	"github.com/l0s3rc0d3/argocd-sync/app/internal/repository"
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
		"REPOSITORY_URL", cfg.RepositoryURL,
		"REPOSITORY_YAML_FILE_PATH", cfg.RepositoryYAMLFilePath,
		"REPOSITORY_YAML_IMAGE_TAG_ROUTE", cfg.RepositoryYAMLImageTagRoute,
		"REPOSITORY_NEW_IMAGE_TAG", cfg.RepositoryNewImageTag,
		"LOGGING_LEVEL", cfg.LoggingLevel,
	)

	if cfg.ArgocdInsecure {
		logger.Warn("TLS certificate verification disabled", "ARGOCD_INSECURE", true)
	}

	argoClient := argocd.NewClient(cfg.ArgocdURL, cfg.ArgocdToken, cfg.ArgocdInsecure)

	app, err := argoClient.GetApplication(cfg.ArgocdAppName)
	if err != nil {
		logger.Error("failed to get application", "error", err)
		os.Exit(1)
	}
	logger.Debug("application retrieved", "app", app)

	repoClient, err := repository.NewClient(repository.Options{
		URL:         cfg.RepositoryURL,
		PAT:         cfg.RepositoryPAT,
		AuthorName:  cfg.RepositoryCommitAuthorName,
		AuthorEmail: cfg.RepositoryCommitAuthorEmail,
	})
	if err != nil {
		logger.Error("failed to init repository client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cleanupErr := repoClient.Cleanup(); cleanupErr != nil {
			logger.Warn("failed to cleanup workdir", "error", cleanupErr)
		}
	}()

	logger.Info("cloning repository", "workdir", repoClient.WorkDir())
	if err := repoClient.Clone(); err != nil {
		logger.Error("failed to clone repository", "error", err)
		os.Exit(1)
	}

	absYAMLPath := filepath.Join(repoClient.WorkDir(), cfg.RepositoryYAMLFilePath)
	logger.Info("updating yaml value",
		"file", absYAMLPath,
		"route", cfg.RepositoryYAMLImageTagRoute,
		"newValue", cfg.RepositoryNewImageTag,
	)
	if err := repository.UpdateYAMLValue(
		absYAMLPath,
		cfg.RepositoryYAMLImageTagRoute,
		cfg.RepositoryNewImageTag,
	); err != nil {
		logger.Error("failed to update yaml", "error", err)
		os.Exit(1)
	}

	logger.Info("committing and pushing", "message", cfg.RepositoryCommitMsg)
	if err := repoClient.CommitAndPush(
		cfg.RepositoryYAMLFilePath,
		cfg.RepositoryCommitMsg,
	); err != nil {
		logger.Error("failed to commit and push", "error", err)
		os.Exit(1)
	}

	logger.Info("repository synced successfully")
}