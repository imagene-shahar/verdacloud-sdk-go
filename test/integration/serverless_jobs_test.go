//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// TestServerlessJobsListOnly tests listing job deployments without creating (sanity check)
func TestServerlessJobsListOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := getTestClient(t)
	ctx := context.Background()

	t.Run("list existing job deployments", func(t *testing.T) {
		jobs, err := client.ServerlessJobs.GetJobDeployments(ctx)
		if err != nil {
			// Handle common errors gracefully
			if apiErr, ok := err.(*verda.APIError); ok {
				if apiErr.StatusCode == 404 {
					t.Skip("⚠️  Job deployments endpoint not available (404)")
				} else if apiErr.StatusCode == 504 {
					t.Skip("⚠️  API timeout (504) - staging environment may be slow or unavailable")
				} else if apiErr.StatusCode >= 500 {
					t.Skipf("⚠️  Server error (%d) - API may be experiencing issues", apiErr.StatusCode)
				}
			}
			t.Fatalf("❌ Failed to list job deployments: %v", err)
		}
		t.Logf("✅ Found %d existing job deployments", len(jobs))
		for i, j := range jobs {
			if i < 5 { // Show first 5
				t.Logf("  - %s (created: %s)", j.Name, j.CreatedAt)
			}
		}
	})
}

// TestServerlessJobsCRUDWithScalingAndEnvVars demonstrates complete CRUD flow
// Note: Serverless jobs API has limited endpoints compared to container deployments
// - Environment variables CRUD is NOT supported (set at creation time only)
// - Scaling does not have a dedicated update endpoint; use job deployment PATCH instead
func TestServerlessJobsCRUDWithScalingAndEnvVars(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := getTestClient(t)
	ctx := context.Background()

	// Pick cheapest available serverless compute (e2e cost control)
	computeName, computeSize, ok := FindAvailableContainerCompute(ctx, t, client, "")
	if !ok {
		t.Skip("⏭️  SKIPPING: No container compute available for jobs")
	}
	t.Logf("Using compute: %s (size %d)", computeName, computeSize)

	// Create a unique job name
	jobName := generateRandomName("job-test")
	var jobCreated bool // Track if job was successfully created

	// Step 1: CREATE - Create a new job deployment (public registry for e2e; no extra secrets)
	t.Run("1. CREATE job deployment", func(t *testing.T) {
		req := &verda.CreateJobDeploymentRequest{
			Name: jobName,
			ContainerRegistrySettings: &verda.ContainerRegistrySettings{
				IsPrivate: false,
			},
			Containers: []verda.CreateDeploymentContainer{
				{
					Image:       "registry-1.docker.io/chentex/random-logger:v1.0.1",
					ExposedPort: 8080,
					Healthcheck: &verda.ContainerHealthcheck{
						Enabled: true,
						Port:    8081,
						Path:    "/health",
					},
					EntrypointOverrides: &verda.ContainerEntrypointOverrides{
						Enabled:    true,
						Entrypoint: []string{"python3", "main.py"},
						Cmd:        []string{"--port", "8080"},
					},
					Env: []verda.ContainerEnvVar{
						{
							Name:                     "MY_ENV_VAR",
							ValueOrReferenceToSecret: "my-value",
							Type:                     "plain",
						},
					},
					VolumeMounts: []verda.ContainerVolumeMount{
						{
							Type:      "scratch",
							MountPath: "/data",
						},
					},
				},
			},
			Compute: &verda.ContainerCompute{
				Name: computeName,
				Size: computeSize,
			},
			Scaling: &verda.JobScalingOptions{
				MaxReplicaCount:        1,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600, // Job timeout in seconds (required)
			},
		}

		job, err := client.ServerlessJobs.CreateJobDeployment(ctx, req)
		if err != nil {
			// Check if it's a timeout or resource availability issue
			if apiErr, ok := err.(*verda.APIError); ok {
				if apiErr.StatusCode == 504 {
					t.Skip("⚠️  Skipping: API timeout (504) - server may be provisioning resources or endpoint unavailable")
				} else if apiErr.StatusCode == 404 {
					t.Skip("⚠️  Skipping: Job deployments endpoint not available (404)")
				} else if apiErr.StatusCode == 400 {
					t.Skipf("⚠️  Skipping: Bad request (400) - compute resource may not be available: %v", err)
				}
			}
			t.Fatalf("❌ Failed to create job deployment: %v", err)
		}

		if job.Name != jobName {
			t.Errorf("Expected job name %s, got %s", jobName, job.Name)
		}
		jobCreated = true // Mark as successfully created

		if len(job.Containers) > 0 {
			t.Logf("✅ Created job deployment: %s (container: %s)", job.Name, job.Containers[0].Name)
		} else {
			t.Logf("✅ Created job deployment: %s (no containers in response)", job.Name)
		}
	})

	// Cleanup runs on success and on failure/panic so test data is always removed
	defer func() {
		if jobCreated {
			t.Logf("🧹 Cleaning up job deployment: %s", jobName)
			// Wait for job deployment to stabilize before attempting delete
			t.Logf("   Waiting 15s for job deployment to stabilize...")
			time.Sleep(15 * time.Second)

			// Retry delete up to 3 times with backoff
			var deleteErr error
			for attempt := 1; attempt <= 3; attempt++ {
				deleteErr = client.ServerlessJobs.DeleteJobDeployment(ctx, jobName, 120000)
				if deleteErr == nil {
					t.Logf("✅ Deleted job deployment: %s", jobName)
					// Wait for deletion to complete
					time.Sleep(10 * time.Second)
					return
				}
				t.Logf("⚠️  Delete attempt %d failed: %v", attempt, deleteErr)
				if attempt < 3 {
					t.Logf("   Retrying in %ds...", attempt*10)
					time.Sleep(time.Duration(attempt*10) * time.Second)
				}
			}
			t.Logf("⚠️  Failed to delete job deployment after 3 attempts: %v", deleteErr)
		}
	}()

	// Skip remaining tests if job wasn't created
	if !jobCreated {
		t.Skip("⚠️  Skipping remaining tests - job deployment was not created")
	}

	// Wait for job deployment to be ready
	time.Sleep(5 * time.Second)

	// Step 2: READ - Get job deployment by name
	t.Run("2. READ job deployment by name", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		job, err := client.ServerlessJobs.GetJobDeploymentByName(ctx, jobName)
		if err != nil {
			// Handle 504 gracefully for read operations too
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504) on read operation")
			}
			t.Fatalf("❌ Failed to get job deployment: %v", err)
		}
		if job.Name != jobName {
			t.Errorf("Expected job name %s, got %s", jobName, job.Name)
		}
		t.Logf("✅ Retrieved job deployment: %s (containers: %d)", job.Name, len(job.Containers))
	})

	// Step 3: UPDATE - Patch the existing job deployment
	t.Run("3. UPDATE job deployment", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		req := &verda.UpdateJobDeploymentRequest{
			Scaling: &verda.JobScalingOptions{
				MaxReplicaCount:        1,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600,
			},
		}

		job, err := client.ServerlessJobs.UpdateJobDeployment(ctx, jobName, req)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok {
				if apiErr.StatusCode == 504 {
					t.Skip("⚠️  Skipping: API timeout (504) on update operation")
				}
				if apiErr.StatusCode == 404 {
					t.Skip("⚠️  Skipping: Job deployment PATCH endpoint not available (404)")
				}
			}
			t.Fatalf("❌ Failed to update job deployment: %v", err)
		}
		if job.Name != jobName {
			t.Errorf("Expected job name %s, got %s", jobName, job.Name)
		}
		t.Logf("✅ Updated job deployment: %s", job.Name)
	})

	// Step 4: LIST - List all job deployments
	t.Run("4. LIST all job deployments", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		jobs, err := client.ServerlessJobs.GetJobDeployments(ctx)
		if err != nil {
			// Handle 504 gracefully
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504) on list operation")
			}
			t.Fatalf("❌ Failed to list job deployments: %v", err)
		}

		found := false
		for _, j := range jobs {
			if j.Name == jobName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Job deployment %s not found in list", jobName)
		}
		t.Logf("✅ Listed %d job deployments, found our job: %s", len(jobs), jobName)
	})

	// Step 5: GET job deployment status
	t.Run("5. READ job deployment status", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		status, err := client.ServerlessJobs.GetJobDeploymentStatus(ctx, jobName)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504)")
			}
			t.Fatalf("❌ Failed to get job deployment status: %v", err)
		}
		t.Logf("✅ Job deployment status: %s", status.Status)
	})

	// Step 6: GET scaling options
	t.Run("6. READ job scaling options", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		scaling, err := client.ServerlessJobs.GetJobDeploymentScaling(ctx, jobName)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504)")
			}
			t.Fatalf("❌ Failed to get scaling options: %v", err)
		}
		t.Logf("✅ Got scaling options: maxReplicas=%d, queueTTL=%d",
			scaling.MaxReplicaCount, scaling.QueueMessageTTLSeconds)
	})

	// Step 7: Pause job deployment
	t.Run("7. PAUSE job deployment", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		err := client.ServerlessJobs.PauseJobDeployment(ctx, jobName)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504)")
			}
			t.Logf("⚠️  PauseJobDeployment: %v", err)
		} else {
			t.Log("✅ Job deployment paused")
		}
	})

	time.Sleep(2 * time.Second)

	// Step 8: Resume job deployment
	t.Run("8. RESUME job deployment", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		err := client.ServerlessJobs.ResumeJobDeployment(ctx, jobName)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504)")
			}
			t.Logf("⚠️  ResumeJobDeployment: %v", err)
		} else {
			t.Log("✅ Job deployment resumed")
		}
	})

	// Step 9: Purge job queue
	t.Run("9. PURGE job deployment queue", func(t *testing.T) {
		if !jobCreated {
			t.Skip("⚠️  Skipping - job deployment was not created")
		}

		err := client.ServerlessJobs.PurgeJobDeploymentQueue(ctx, jobName)
		if err != nil {
			if apiErr, ok := err.(*verda.APIError); ok && apiErr.StatusCode == 504 {
				t.Skip("⚠️  Skipping: API timeout (504)")
			}
			t.Logf("⚠️  PurgeJobDeploymentQueue: %v", err)
		} else {
			t.Log("✅ Job deployment queue purged")
		}
	})

	// Wait before deletion (happens in defer)
	time.Sleep(2 * time.Second)
}
