package verda

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type InstanceService struct {
	client *Client
}

func (s *InstanceService) Get(ctx context.Context, status string) ([]Instance, error) {
	path := "/instances"
	if status != "" {
		params := url.Values{}
		params.Set("status", status)
		path += "?" + params.Encode()
	}

	instances, _, err := getRequest[[]Instance](ctx, s.client, path)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (s *InstanceService) GetByID(ctx context.Context, id string) (*Instance, error) {
	path := fmt.Sprintf("/instances/%s", id)

	instance, _, err := getRequest[Instance](ctx, s.client, path)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *InstanceService) Create(ctx context.Context, req CreateInstanceRequest) (*Instance, error) {
	if req.LocationCode == "" {
		req.LocationCode = LocationFIN01
	}

	if req.SSHKeyIDs == nil {
		req.SSHKeyIDs = []string{}
	}

	return s.createWithPlainTextResponse(ctx, req)
}

// createWithPlainTextResponse handles API's inconsistent response format (sometimes JSON, sometimes plain text ID)
func (s *InstanceService) createWithPlainTextResponse(ctx context.Context, req CreateInstanceRequest) (*Instance, error) {
	resp, err := s.client.makeRequest(ctx, http.MethodPost, "/instances", req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response: %w", err)
		}
		s.client.Logger.Debug("Instance creation failed with status %d: %s", resp.StatusCode, string(body))

		var apiError APIError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(body),
			}
		}
		apiError.StatusCode = resp.StatusCode
		return nil, &apiError
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var instance Instance
	if err := json.Unmarshal(body, &instance); err != nil {
		// Fall back to plain text ID
		instanceID := strings.TrimSpace(string(body))
		return s.GetByID(ctx, instanceID)
	}

	return &instance, nil
}

// Action performs an action on one or more instances.
func (s *InstanceService) Action(ctx context.Context, req InstanceActionRequest) ([]InstanceActionResult, error) {
	resp, err := s.client.makeRequest(ctx, http.MethodPut, "/instances", req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil

	case http.StatusAccepted, http.StatusMultiStatus:
		var results []InstanceActionResult
		if err := json.Unmarshal(body, &results); err != nil {
			return nil, fmt.Errorf("failed to parse action results: %w", err)
		}
		return results, nil

	default:
		var apiError APIError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body)}
		}
		apiError.StatusCode = resp.StatusCode
		return nil, &apiError
	}
}

func (s *InstanceService) GetLocationAvailabilities(ctx context.Context) ([]LocationAvailability, error) {
	availabilities, _, err := getRequest[[]LocationAvailability](ctx, s.client, "/instance-availability")
	if err != nil {
		return nil, err
	}

	return availabilities, nil
}

func (s *InstanceService) CheckInstanceTypeAvailability(ctx context.Context, instanceType string) (bool, error) {
	path := fmt.Sprintf("/instance-availability/%s", instanceType)

	// API returns "true"/"false" as JSON string, not boolean
	resp, err := s.client.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiError APIError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return false, &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(body),
			}
		}
		apiError.StatusCode = resp.StatusCode
		return false, &apiError
	}

	var boolResult bool
	if err := json.Unmarshal(body, &boolResult); err == nil {
		return boolResult, nil
	}

	var stringResult string
	if err := json.Unmarshal(body, &stringResult); err == nil {
		return stringResult == trueString, nil
	}

	return false, fmt.Errorf("unexpected response format: %s", string(body))
}

func (s *InstanceService) Boot(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionBoot, ID: ids})
	return err
}

func (s *InstanceService) Start(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionStart, ID: ids})
	return err
}

func (s *InstanceService) Shutdown(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionShutdown, ID: ids})
	return err
}

func (s *InstanceService) Delete(ctx context.Context, ids []string, volumeIDs []string, deletePermanently bool) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionDelete, ID: ids, VolumeIDs: volumeIDs, DeletePermanently: deletePermanently})
	return err
}

func (s *InstanceService) Discontinue(ctx context.Context, ids []string, volumeIDs []string, deletePermanently bool) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionDiscontinue, ID: ids, VolumeIDs: volumeIDs, DeletePermanently: deletePermanently})
	return err
}

// Hibernate shuts down and archives an instance - must be shut down first or API will error.
// Volumes are detached and the instance is deleted during hibernation.
func (s *InstanceService) Hibernate(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionHibernate, ID: ids})
	return err
}

func (s *InstanceService) ConfigureSpot(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionConfigureSpot, ID: ids})
	return err
}

func (s *InstanceService) ForceShutdown(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionForceShutdown, ID: ids})
	return err
}

func (s *InstanceService) DeleteStuck(ctx context.Context, volumeIDs []string, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionDeleteStuck, ID: ids, VolumeIDs: volumeIDs})
	return err
}

func (s *InstanceService) Deploy(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionDeploy, ID: ids})
	return err
}

func (s *InstanceService) Transfer(ctx context.Context, ids ...string) error {
	_, err := s.Action(ctx, InstanceActionRequest{Action: ActionTransfer, ID: ids})
	return err
}
