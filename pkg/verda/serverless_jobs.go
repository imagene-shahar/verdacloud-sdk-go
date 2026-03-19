package verda

import (
	"context"
	"fmt"
	"net/url"
)

type ServerlessJobsService struct {
	client *Client
}

func validateUpdateJobDeploymentRequest(req *UpdateJobDeploymentRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	for i, container := range req.Containers {
		if container.Name == "" {
			return fmt.Errorf("containers[%d].name is required", i)
		}
		if container.Image != "" && isLatestTag(container.Image) {
			return fmt.Errorf("containers[%d].image: 'latest' tag is not allowed, please specify a specific version tag (e.g., alpine:3.19)", i)
		}
	}

	return nil
}

func (s *ServerlessJobsService) GetJobDeployments(ctx context.Context) ([]JobDeploymentShortInfo, error) {
	jobs, _, err := getRequest[[]JobDeploymentShortInfo](ctx, s.client, "/job-deployments")
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *ServerlessJobsService) CreateJobDeployment(ctx context.Context, req *CreateJobDeploymentRequest) (*JobDeployment, error) {
	if err := validateCreateJobDeploymentRequest(req); err != nil {
		return nil, err
	}

	job, _, err := postRequest[JobDeployment](ctx, s.client, "/job-deployments", req)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// validateCreateJobDeploymentRequest validates all required fields for job deployment creation
func validateCreateJobDeploymentRequest(req *CreateJobDeploymentRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Basic required fields
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Compute == nil {
		return fmt.Errorf("compute is required")
	}
	if req.Compute.Name == "" {
		return fmt.Errorf("compute.name is required")
	}

	// Container validation
	if len(req.Containers) == 0 {
		return fmt.Errorf("at least one container is required")
	}
	for i, c := range req.Containers {
		if c.Image == "" {
			return fmt.Errorf("containers[%d].image is required", i)
		}
		if c.ExposedPort == 0 {
			return fmt.Errorf("containers[%d].exposed_port is required", i)
		}
		// Check for "latest" tag - API does not allow it
		if isLatestTag(c.Image) {
			return fmt.Errorf("containers[%d].image: 'latest' tag is not allowed, please specify a specific version tag (e.g., alpine:3.19)", i)
		}
	}

	// Scaling validation
	if req.Scaling == nil {
		return fmt.Errorf("scaling is required")
	}
	if req.Scaling.MaxReplicaCount == 0 {
		return fmt.Errorf("scaling.max_replica_count is required")
	}
	if req.Scaling.DeadlineSeconds == 0 {
		return fmt.Errorf("scaling.deadline_seconds is required (job timeout in seconds)")
	}
	if req.Scaling.QueueMessageTTLSeconds == 0 {
		return fmt.Errorf("scaling.queue_message_ttl_seconds is required")
	}

	return nil
}

func (s *ServerlessJobsService) GetJobDeploymentByName(ctx context.Context, jobName string) (*JobDeployment, error) {
	path := fmt.Sprintf("/job-deployments/%s", jobName)
	job, _, err := getRequest[JobDeployment](ctx, s.client, path)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *ServerlessJobsService) UpdateJobDeployment(ctx context.Context, jobName string, req *UpdateJobDeploymentRequest) (*JobDeployment, error) {
	if jobName == "" {
		return nil, fmt.Errorf("jobName is required")
	}
	if err := validateUpdateJobDeploymentRequest(req); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/job-deployments/%s", jobName)
	job, _, err := patchRequest[JobDeployment](ctx, s.client, path, req)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// DeleteJobDeployment removes a job with timeout in milliseconds (0-300000ms)
// timeoutMs behavior:
//   - 0: Skip waiting (returns immediately)
//   - Negative (e.g., -1): Use API default of 60000ms (omit query parameter)
//   - 1-300000: Wait specified milliseconds
//   - >300000: Capped at 300000ms
func (s *ServerlessJobsService) DeleteJobDeployment(ctx context.Context, jobName string, timeoutMs int) error {
	path := fmt.Sprintf("/job-deployments/%s", jobName)

	// Handle timeout parameter based on API specification
	if timeoutMs >= 0 {
		timeout := timeoutMs
		if timeout > 300000 {
			timeout = 300000 // cap at max 300 seconds
		}
		params := url.Values{}
		params.Set("timeout", fmt.Sprintf("%d", timeout))
		path += "?" + params.Encode()
	}
	// If timeoutMs < 0, don't add timeout parameter (use API default)

	_, err := deleteRequestAllowEmptyResponse(ctx, s.client, path)
	return err
}

func (s *ServerlessJobsService) GetJobDeploymentScaling(ctx context.Context, jobName string) (*JobScalingOptions, error) {
	if jobName == "" {
		return nil, fmt.Errorf("jobName is required")
	}
	path := fmt.Sprintf("/job-deployments/%s/scaling", jobName)
	scaling, _, err := getRequest[JobScalingOptions](ctx, s.client, path)
	if err != nil {
		return nil, err
	}
	return &scaling, nil
}

func (s *ServerlessJobsService) PurgeJobDeploymentQueue(ctx context.Context, jobName string) error {
	path := fmt.Sprintf("/job-deployments/%s/purge-queue", jobName)
	_, _, err := postRequest[interface{}](ctx, s.client, path, nil)
	return err
}

func (s *ServerlessJobsService) PauseJobDeployment(ctx context.Context, jobName string) error {
	path := fmt.Sprintf("/job-deployments/%s/pause", jobName)
	_, _, err := postRequest[interface{}](ctx, s.client, path, nil)
	return err
}

func (s *ServerlessJobsService) ResumeJobDeployment(ctx context.Context, jobName string) error {
	path := fmt.Sprintf("/job-deployments/%s/resume", jobName)
	_, _, err := postRequest[interface{}](ctx, s.client, path, nil)
	return err
}

func (s *ServerlessJobsService) GetJobDeploymentStatus(ctx context.Context, jobName string) (*JobDeploymentStatus, error) {
	path := fmt.Sprintf("/job-deployments/%s/status", jobName)
	status, _, err := getRequest[JobDeploymentStatus](ctx, s.client, path)
	if err != nil {
		return nil, err
	}
	return &status, nil
}
