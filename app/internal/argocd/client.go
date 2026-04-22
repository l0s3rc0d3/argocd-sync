package argocd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"io"
)

var ErrSyncConflict = errors.New("sync already in progress")

type Application struct {
	Status ApplicationStatus `json:"status"`
}

type ApplicationStatus struct {
	Summary        Summary         `json:"summary"`
	Resources      []Resource      `json:"resources"`
	OperationState *OperationState `json:"operationState"`
}

type Summary struct {
	Images []string `json:"images"`
}

type Resource struct {
	Group     string  `json:"group"`
	Version   string  `json:"version"`
	Kind      string  `json:"kind"`
	Namespace string  `json:"namespace"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Health    *Health `json:"health"`
}

type Health struct {
	Status string `json:"status"`
}

type OperationState struct {
	Phase string `json:"phase"`
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, insecure bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (c *Client) GetApplication(appName string) (map[string]interface{}, error) {
	resp, err := c.doGet(fmt.Sprintf("/api/v1/applications/%s", appName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}

func (c *Client) HardRefresh(appName string) error {
	path := fmt.Sprintf("/api/v1/applications/%s?refresh=hard", appName)
	resp, err := c.doGet(path)
	if err != nil {
		return fmt.Errorf("hard refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hard refresh returned unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) SyncApplication(appName string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, appName)

	req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create sync request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return ErrSyncConflict
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync returned unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}
}

func (c *Client) WatchUntilHealthy(
	ctx context.Context,
	logger *slog.Logger,
	appName, newImageTag string,
	pollInterval time.Duration,
) error {
	logger.Info("starting watch",
		"app", appName,
		"newImageTag", newImageTag,
		"pollInterval", pollInterval.String(),
	)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for application %q to become healthy", appName)

		case <-ticker.C:
			app, err := c.getTypedApplication(appName)
			if err != nil {
				logger.Warn("failed to fetch application during watch, will retry",
					"app", appName,
					"error", err,
				)
				continue
			}

			ready, reason := isApplicationReady(app, newImageTag)
			if ready {
				logger.Info("application is synced and healthy", "app", appName)
				return nil
			}

			logger.Info("application not ready yet, retrying",
				"app", appName,
				"reason", reason,
			)
		}
	}
}

func (c *Client) getTypedApplication(appName string) (*Application, error) {
	resp, err := c.doGet(fmt.Sprintf("/api/v1/applications/%s", appName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode application: %w", err)
	}
	return &app, nil
}

func (c *Client) doGet(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	return resp, nil
}

func isApplicationReady(app *Application, newImageTag string) (bool, string) {
	imageFound := false
	for _, img := range app.Status.Summary.Images {
		if strings.Contains(img, newImageTag) {
			imageFound = true
			break
		}
	}
	if !imageFound {
		return false, fmt.Sprintf("new image tag %q not yet visible in status.summary.images %v",
			newImageTag, app.Status.Summary.Images)
	}

	if app.Status.OperationState == nil {
		return false, "status.operationState is nil — sync has not started yet"
	}
	if app.Status.OperationState.Phase != "Succeeded" {
		return false, fmt.Sprintf("status.operationState.phase is %q, waiting for Succeeded",
			app.Status.OperationState.Phase)
	}

	if len(app.Status.Resources) == 0 {
		return false, "status.resources is empty"
	}
	for _, r := range app.Status.Resources {
		id := resourceID(r)
		if r.Status != "Synced" {
			return false, fmt.Sprintf("resource %s has status %q, waiting for Synced", id, r.Status)
		}
		if r.Health != nil && r.Health.Status != "Healthy" {
			return false, fmt.Sprintf("resource %s has health %q, waiting for Healthy", id, r.Health.Status)
		}
	}

	return true, ""
}

func resourceID(r Resource) string {
	if r.Group != "" {
		return fmt.Sprintf("%s/%s/%s", r.Group, r.Kind, r.Name)
	}
	return fmt.Sprintf("%s/%s", r.Kind, r.Name)
}