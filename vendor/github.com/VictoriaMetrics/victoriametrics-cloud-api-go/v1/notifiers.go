package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NotifierAuthType is the authentication method used by a custom Alertmanager notifier.
type NotifierAuthType string

const (
	NotifierAuthTypeBasic  NotifierAuthType = "basic"
	NotifierAuthTypeBearer NotifierAuthType = "bearer"
)

// NotifierStatus is the last known health status of a notifier.
type NotifierStatus string

const (
	NotifierStatusUnknown NotifierStatus = "unknown"
	NotifierStatusError   NotifierStatus = "error"
	NotifierStatusActive  NotifierStatus = "active"
)

// NotifierType identifies which kind of notifier is attached to a deployment.
type NotifierType string

const (
	NotifierTypeCustomAlertmanager NotifierType = "custom-alertmanager"
	NotifierTypeCloudAlertmanager  NotifierType = "cloud-alertmanager"
)

// CustomAlertmanager is a notifier pointing at an externally hosted Alertmanager.
type CustomAlertmanager struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	URL         string           `json:"url"`
	AuthType    NotifierAuthType `json:"auth_type"`
	Username    string           `json:"username,omitempty"`
	Password    string           `json:"password,omitempty"`
	BearerToken string           `json:"bearer_token,omitempty"`
	Status      NotifierStatus   `json:"status"`
	Message     string           `json:"message,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// CustomAlertmanagersList is a list of CustomAlertmanager.
type CustomAlertmanagersList []CustomAlertmanager

// CustomAlertmanagerRequest creates or updates a CustomAlertmanager.
type CustomAlertmanagerRequest struct {
	Name        string           `json:"name"`
	URL         string           `json:"url"`
	AuthType    NotifierAuthType `json:"auth_type"`
	Username    string           `json:"username,omitempty"`
	Password    string           `json:"password,omitempty"`
	BearerToken string           `json:"bearer_token,omitempty"`
}

// CloudAlertmanager is a notifier backed by a VictoriaMetrics-hosted Alertmanager.
type CloudAlertmanager struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Config    string         `json:"config"`
	Status    NotifierStatus `json:"status"`
	Message   string         `json:"message,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CloudAlertmanagersList is a list of CloudAlertmanager.
type CloudAlertmanagersList []CloudAlertmanager

// CloudAlertmanagerRequest creates or updates a CloudAlertmanager.
type CloudAlertmanagerRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// DeploymentNotifier is the notifier currently attached to a deployment.
type DeploymentNotifier struct {
	NotifierID   string       `json:"notifier_id,omitempty"`
	NotifierType NotifierType `json:"notifier_type,omitempty"`
	NotifierName string       `json:"notifier_name,omitempty"`
}

// DeploymentNotifierRequest attaches a notifier to a deployment.
type DeploymentNotifierRequest struct {
	NotifierID   string       `json:"notifier_id"`
	NotifierType NotifierType `json:"notifier_type"`
}

func checkNotifierID(notifierID string) error {
	if notifierID == "" {
		return fmt.Errorf("notifier ID cannot be empty")
	}
	return nil
}

// ListCustomAlertmanagers retrieves the custom (externally hosted) Alertmanager notifiers for the account.
func (a *VMCloudAPIClient) ListCustomAlertmanagers(ctx context.Context) (CustomAlertmanagersList, error) {
	return requestAPI[CustomAlertmanagersList](ctx, a, http.MethodGet, nil, "/api/v1/integrations/custom-alertmanagers")
}

// GetCustomAlertmanager retrieves a custom Alertmanager notifier by ID.
func (a *VMCloudAPIClient) GetCustomAlertmanager(ctx context.Context, notifierID string) (CustomAlertmanager, error) {
	if err := checkNotifierID(notifierID); err != nil {
		return CustomAlertmanager{}, err
	}
	return requestAPI[CustomAlertmanager](ctx, a, http.MethodGet, nil, "/api/v1/integrations/custom-alertmanagers", notifierID)
}

// CreateCustomAlertmanager creates a new custom Alertmanager notifier.
func (a *VMCloudAPIClient) CreateCustomAlertmanager(ctx context.Context, req CustomAlertmanagerRequest) (CustomAlertmanager, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CustomAlertmanager{}, fmt.Errorf("failed to marshal custom alertmanager request: %w", err)
	}
	return requestAPI[CustomAlertmanager](ctx, a, http.MethodPost, bytes.NewReader(body), "/api/v1/integrations/custom-alertmanagers")
}

// UpdateCustomAlertmanager updates an existing custom Alertmanager notifier.
func (a *VMCloudAPIClient) UpdateCustomAlertmanager(ctx context.Context, notifierID string, req CustomAlertmanagerRequest) (CustomAlertmanager, error) {
	if err := checkNotifierID(notifierID); err != nil {
		return CustomAlertmanager{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return CustomAlertmanager{}, fmt.Errorf("failed to marshal custom alertmanager request: %w", err)
	}
	return requestAPI[CustomAlertmanager](ctx, a, http.MethodPost, bytes.NewReader(body), "/api/v1/integrations/custom-alertmanagers", notifierID)
}

// DeleteCustomAlertmanager deletes a custom Alertmanager notifier by ID.
func (a *VMCloudAPIClient) DeleteCustomAlertmanager(ctx context.Context, notifierID string) error {
	if err := checkNotifierID(notifierID); err != nil {
		return err
	}
	_, err := requestAPI[any](ctx, a, http.MethodDelete, nil, "/api/v1/integrations/custom-alertmanagers", notifierID)
	return err
}

// ListCloudAlertmanagers retrieves the VictoriaMetrics-hosted Alertmanager notifiers for the account.
func (a *VMCloudAPIClient) ListCloudAlertmanagers(ctx context.Context) (CloudAlertmanagersList, error) {
	return requestAPI[CloudAlertmanagersList](ctx, a, http.MethodGet, nil, "/api/v1/integrations/cloud-alertmanagers")
}

// GetCloudAlertmanager retrieves a cloud Alertmanager notifier by ID.
func (a *VMCloudAPIClient) GetCloudAlertmanager(ctx context.Context, notifierID string) (CloudAlertmanager, error) {
	if err := checkNotifierID(notifierID); err != nil {
		return CloudAlertmanager{}, err
	}
	return requestAPI[CloudAlertmanager](ctx, a, http.MethodGet, nil, "/api/v1/integrations/cloud-alertmanagers", notifierID)
}

// CreateCloudAlertmanager creates a new cloud Alertmanager notifier.
func (a *VMCloudAPIClient) CreateCloudAlertmanager(ctx context.Context, req CloudAlertmanagerRequest) (CloudAlertmanager, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CloudAlertmanager{}, fmt.Errorf("failed to marshal cloud alertmanager request: %w", err)
	}
	return requestAPI[CloudAlertmanager](ctx, a, http.MethodPost, bytes.NewReader(body), "/api/v1/integrations/cloud-alertmanagers")
}

// UpdateCloudAlertmanager updates an existing cloud Alertmanager notifier.
func (a *VMCloudAPIClient) UpdateCloudAlertmanager(ctx context.Context, notifierID string, req CloudAlertmanagerRequest) (CloudAlertmanager, error) {
	if err := checkNotifierID(notifierID); err != nil {
		return CloudAlertmanager{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return CloudAlertmanager{}, fmt.Errorf("failed to marshal cloud alertmanager request: %w", err)
	}
	return requestAPI[CloudAlertmanager](ctx, a, http.MethodPost, bytes.NewReader(body), "/api/v1/integrations/cloud-alertmanagers", notifierID)
}

// DeleteCloudAlertmanager deletes a cloud Alertmanager notifier by ID.
func (a *VMCloudAPIClient) DeleteCloudAlertmanager(ctx context.Context, notifierID string) error {
	if err := checkNotifierID(notifierID); err != nil {
		return err
	}
	_, err := requestAPI[any](ctx, a, http.MethodDelete, nil, "/api/v1/integrations/cloud-alertmanagers", notifierID)
	return err
}

// GetDeploymentNotifier retrieves the notifier currently attached to a deployment, if any.
func (a *VMCloudAPIClient) GetDeploymentNotifier(ctx context.Context, deploymentID string) (DeploymentNotifier, error) {
	if err := checkDeploymentID(deploymentID); err != nil {
		return DeploymentNotifier{}, err
	}
	return requestAPI[DeploymentNotifier](ctx, a, http.MethodGet, nil, "/api/v1/deployments", deploymentID, "alerting", "notifier")
}

// SetDeploymentNotifier attaches a notifier to a deployment.
func (a *VMCloudAPIClient) SetDeploymentNotifier(ctx context.Context, deploymentID string, req DeploymentNotifierRequest) (DeploymentNotifier, error) {
	if err := checkDeploymentID(deploymentID); err != nil {
		return DeploymentNotifier{}, err
	}
	if req.NotifierID == "" {
		return DeploymentNotifier{}, fmt.Errorf("notifier ID cannot be empty")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return DeploymentNotifier{}, fmt.Errorf("failed to marshal deployment notifier request: %w", err)
	}
	return requestAPI[DeploymentNotifier](ctx, a, http.MethodPost, bytes.NewReader(body), "/api/v1/deployments", deploymentID, "alerting", "notifier")
}

// DeleteDeploymentNotifier detaches the notifier from a deployment.
func (a *VMCloudAPIClient) DeleteDeploymentNotifier(ctx context.Context, deploymentID string) error {
	if err := checkDeploymentID(deploymentID); err != nil {
		return err
	}
	_, err := requestAPI[any](ctx, a, http.MethodDelete, nil, "/api/v1/deployments", deploymentID, "alerting", "notifier")
	return err
}
