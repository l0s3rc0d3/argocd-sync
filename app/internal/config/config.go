package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ArgocdURL         string
	ArgocdToken       string
	ArgocdAppName     string
	ArgocdInsecure    bool
	ArgocdTimeout     time.Duration
	ArgocdPollInterval time.Duration

	LoggingLevel string

	RepositoryURL               string
	RepositoryPAT               string
	RepositoryYAMLImageTagRoute string
	RepositoryYAMLFilePath      string
	RepositoryNewImageTag       string
	RepositoryCommitMsg         string
	RepositoryCommitAuthorName  string
	RepositoryCommitAuthorEmail string
}

func Load() (*Config, error) {
	cfg := &Config{}
	var missing []string

	cfg.ArgocdURL = os.Getenv("ARGOCD_URL")
	if cfg.ArgocdURL == "" {
		missing = append(missing, "ARGOCD_URL")
	}

	cfg.ArgocdToken = os.Getenv("ARGOCD_TOKEN")
	if cfg.ArgocdToken == "" {
		missing = append(missing, "ARGOCD_TOKEN")
	}

	cfg.ArgocdAppName = os.Getenv("ARGOCD_APP_NAME")
	if cfg.ArgocdAppName == "" {
		missing = append(missing, "ARGOCD_APP_NAME")
	}

	cfg.ArgocdInsecure = os.Getenv("ARGOCD_INSECURE") == "true"

	cfg.ArgocdTimeout = parseDuration(os.Getenv("ARGOCD_TIMEOUT"), 5*time.Minute)
	cfg.ArgocdPollInterval = parseDuration(os.Getenv("ARGOCD_POLL_INTERVAL"), 15*time.Second)

	cfg.LoggingLevel = os.Getenv("LOGGING_LEVEL")
	if cfg.LoggingLevel == "" {
		cfg.LoggingLevel = "INFO"
	}

	cfg.RepositoryURL = os.Getenv("REPOSITORY_URL")
	if cfg.RepositoryURL == "" {
		missing = append(missing, "REPOSITORY_URL")
	}

	cfg.RepositoryPAT = os.Getenv("REPOSITORY_PAT")
	if cfg.RepositoryPAT == "" {
		missing = append(missing, "REPOSITORY_PAT")
	}

	cfg.RepositoryYAMLFilePath = os.Getenv("REPOSITORY_YAML_FILE_PATH")
	if cfg.RepositoryYAMLFilePath == "" {
		missing = append(missing, "REPOSITORY_YAML_FILE_PATH")
	}

	cfg.RepositoryNewImageTag = os.Getenv("REPOSITORY_NEW_IMAGE_TAG")
	if cfg.RepositoryNewImageTag == "" {
		missing = append(missing, "REPOSITORY_NEW_IMAGE_TAG")
	}

	cfg.RepositoryYAMLImageTagRoute = os.Getenv("REPOSITORY_YAML_IMAGE_TAG_ROUTE")
	if cfg.RepositoryYAMLImageTagRoute == "" {
		cfg.RepositoryYAMLImageTagRoute = "image.tag"
	}

	cfg.RepositoryCommitMsg = os.Getenv("REPOSITORY_COMMIT_MSG")
	if cfg.RepositoryCommitMsg == "" && cfg.RepositoryNewImageTag != "" {
		cfg.RepositoryCommitMsg = fmt.Sprintf(
			"feat(argocd-sync): image tag update to: %s",
			cfg.RepositoryNewImageTag,
		)
	}

	cfg.RepositoryCommitAuthorName = os.Getenv("REPOSITORY_COMMIT_AUTHOR_NAME")
	if cfg.RepositoryCommitAuthorName == "" {
		cfg.RepositoryCommitAuthorName = "argocd-sync"
	}

	cfg.RepositoryCommitAuthorEmail = os.Getenv("REPOSITORY_COMMIT_AUTHOR_EMAIL")
	if cfg.RepositoryCommitAuthorEmail == "" {
		cfg.RepositoryCommitAuthorEmail = "argocd-sync@local"
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment variables: %v", missing)
	}

	return cfg, nil
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}