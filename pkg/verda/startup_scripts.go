package verda

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type StartupScriptService struct {
	client *Client
}

type CreateStartupScriptRequest struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type DeleteMultipleStartupScriptsRequest struct {
	Scripts []string `json:"scripts"`
}

func (s *StartupScriptService) GetAllStartupScripts(ctx context.Context) ([]StartupScript, error) {
	scripts, _, err := getRequest[[]StartupScript](ctx, s.client, "/scripts")
	if err != nil {
		return nil, err
	}
	return scripts, nil
}

func (s *StartupScriptService) GetStartupScriptByID(ctx context.Context, scriptID string) (*StartupScript, error) {
	path := fmt.Sprintf("/scripts/%s", scriptID)

	resp, err := s.client.makeRequest(ctx, http.MethodGet, path, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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

	script, err := parseStartupScriptResponse(body)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// AddStartupScript creates a script and refetches it since the API returns only the ID as plain text
func (s *StartupScriptService) AddStartupScript(ctx context.Context, req *CreateStartupScriptRequest) (*StartupScript, error) {
	return s.createWithPlainTextResponse(ctx, req)
}

// createWithPlainTextResponse handles API's inconsistent response format (sometimes JSON, sometimes plain text ID)
func (s *StartupScriptService) createWithPlainTextResponse(ctx context.Context, req *CreateStartupScriptRequest) (*StartupScript, error) {
	resp, err := s.client.makeRequest(ctx, http.MethodPost, "/scripts", req)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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

	// Try JSON first, fall back to plain text ID
	script, err := parseStartupScriptResponse(body)
	if err == nil {
		return script, nil
	}

	scriptID := strings.TrimSpace(string(body))
	scriptID = strings.Trim(scriptID, "\"")

	return s.GetStartupScriptByID(ctx, scriptID)
}

func parseStartupScriptResponse(body []byte) (*StartupScript, error) {
	var script StartupScript
	if err := json.Unmarshal(body, &script); err == nil && script.ID != "" {
		return &script, nil
	}

	var scripts []StartupScript
	if err := json.Unmarshal(body, &scripts); err == nil {
		if len(scripts) == 0 {
			return nil, fmt.Errorf("script not found")
		}
		return &scripts[0], nil
	}

	return nil, fmt.Errorf("unexpected startup script response format")
}

func (s *StartupScriptService) DeleteStartupScript(ctx context.Context, scriptID string) error {
	path := fmt.Sprintf("/scripts/%s", scriptID)
	_, err := deleteRequestAllowEmptyResponse(ctx, s.client, path)
	return err
}

func (s *StartupScriptService) DeleteMultipleStartupScripts(ctx context.Context, scriptIDs []string) error {
	req := &DeleteMultipleStartupScriptsRequest{
		Scripts: scriptIDs,
	}
	_, err := deleteRequestWithBody(ctx, s.client, "/scripts", req)
	return err
}
