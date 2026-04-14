package config

import (
	"fmt"
	"os"
)

type Config struct {
	ArgocdURL      string
	ArgocdToken    string
	ArgocdAppName  string
	ArgocdInsecure bool
	LoggingLevel   string
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

	cfg.LoggingLevel = os.Getenv("LOGGING_LEVEL")
	if cfg.LoggingLevel == "" {
		cfg.LoggingLevel = "INFO"
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment variables: %v", missing)
	}

	return cfg, nil
}