package verda

import (
	"context"
	"testing"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda/testutil"
)

func TestServerlessJobsService_GetJobDeployments(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	ctx := context.Background()
	jobs, err := client.ServerlessJobs.GetJobDeployments(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job deployment, got %d", len(jobs))
	}

	job := jobs[0]
	if job.Name != "flux-training" {
		t.Fatalf("expected job name flux-training, got %s", job.Name)
	}
	if job.CreatedAt.IsZero() {
		t.Fatal("expected job to have CreatedAt populated")
	}
	if job.Compute == nil || job.Compute.Name != "H100" {
		t.Fatalf("expected compute H100, got %+v", job.Compute)
	}
}

func TestServerlessJobsService_CreateJobDeployment(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("create job deployment", func(t *testing.T) {
		ctx := context.Background()
		req := &CreateJobDeploymentRequest{
			Name: "flux-training",
			ContainerRegistrySettings: &ContainerRegistrySettings{
				Credentials: &RegistryCredentialsRef{
					Name: "dockerhub-credentials",
				},
			},
			Containers: []CreateDeploymentContainer{
				{
					Image:       "registry-1.docker.io/chentex/random-logger:v1.0.1",
					ExposedPort: 8080,
					Healthcheck: &ContainerHealthcheck{
						Enabled: true,
						Port:    8081,
						Path:    "/health",
					},
					EntrypointOverrides: &ContainerEntrypointOverrides{
						Enabled:    true,
						Entrypoint: []string{"python3", "main.py"},
						Cmd:        []string{"--port", "8080"},
					},
					Env: []ContainerEnvVar{
						{
							Name:                     "MY_ENV_VAR",
							ValueOrReferenceToSecret: "my-value",
							Type:                     "plain",
						},
					},
					VolumeMounts: []ContainerVolumeMount{
						{
							Type:       "scratch",
							MountPath:  "/data",
							SecretName: "my-secret",
							SizeInMB:   64,
							VolumeID:   "fa4a0338-65b2-4819-8450-821190fbaf6d",
						},
					},
				},
			},
			Compute: &ContainerCompute{
				Name: "H100",
				Size: 1,
			},
			Scaling: &JobScalingOptions{
				MaxReplicaCount:        1,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600,
			},
		}

		job, err := client.ServerlessJobs.CreateJobDeployment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job == nil {
			t.Fatal("expected job, got nil")
		}
		if job.Name != "flux-training" {
			t.Fatalf("expected job name flux-training, got %s", job.Name)
		}
	})

	t.Run("validation - container without exposed port", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.ServerlessJobs.CreateJobDeployment(ctx, &CreateJobDeploymentRequest{
			Name: "invalid-job",
			Containers: []CreateDeploymentContainer{
				{
					Image: "registry-1.docker.io/python:3.9.19",
				},
			},
			Compute: &ContainerCompute{
				Name: "H100",
				Size: 1,
			},
			Scaling: &JobScalingOptions{
				MaxReplicaCount:        1,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600,
			},
		})
		if err == nil {
			t.Fatal("expected error for missing exposed port")
		}
	})
}

func TestServerlessJobsService_GetJobDeploymentByName(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	ctx := context.Background()
	job, err := client.ServerlessJobs.GetJobDeploymentByName(ctx, "flux-training")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job == nil {
		t.Fatal("expected job, got nil")
	}
	if job.Name != "flux-training" {
		t.Fatalf("expected job name flux-training, got %s", job.Name)
	}
	if len(job.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(job.Containers))
	}
}

func TestServerlessJobsService_UpdateJobDeployment(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("validation - nil request", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.ServerlessJobs.UpdateJobDeployment(ctx, "test-job", nil)
		if err == nil {
			t.Error("expected error for nil request")
		}
	})

	t.Run("validation - empty job name", func(t *testing.T) {
		ctx := context.Background()
		req := &UpdateJobDeploymentRequest{
			Scaling: &JobScalingOptions{
				MaxReplicaCount:        2,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600,
			},
		}
		_, err := client.ServerlessJobs.UpdateJobDeployment(ctx, "", req)
		if err == nil {
			t.Error("expected error for empty job name")
		}
	})

	t.Run("partial update - scaling only", func(t *testing.T) {
		ctx := context.Background()
		req := &UpdateJobDeploymentRequest{
			Scaling: &JobScalingOptions{
				MaxReplicaCount:        2,
				QueueMessageTTLSeconds: 300,
				DeadlineSeconds:        3600,
			},
		}
		job, err := client.ServerlessJobs.UpdateJobDeployment(ctx, "test-job", req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job == nil {
			t.Fatal("expected job, got nil")
		}
	})

	t.Run("container update - with name", func(t *testing.T) {
		ctx := context.Background()
		req := &UpdateJobDeploymentRequest{
			Containers: []CreateDeploymentContainer{
				{
					Name:        "random-logger-0",
					Image:       "registry-1.docker.io/chentex/random-logger:v1.0.1",
					ExposedPort: 8080,
				},
			},
		}
		job, err := client.ServerlessJobs.UpdateJobDeployment(ctx, "test-job", req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job == nil {
			t.Fatal("expected job, got nil")
		}
	})

	t.Run("validation - updating container without name", func(t *testing.T) {
		ctx := context.Background()
		req := &UpdateJobDeploymentRequest{
			Containers: []CreateDeploymentContainer{
				{
					Image:       "registry-1.docker.io/chentex/random-logger:v1.0.1",
					ExposedPort: 8080,
				},
			},
		}
		_, err := client.ServerlessJobs.UpdateJobDeployment(ctx, "test-job", req)
		if err == nil {
			t.Error("expected error when container name is missing")
		}
	})
}

func TestServerlessJobsService_DeleteJobDeployment(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	ctx := context.Background()
	if err := client.ServerlessJobs.DeleteJobDeployment(ctx, "test-job-delete", 0); err != nil {
		t.Fatalf("unexpected error deleting job deployment: %v", err)
	}
}

func TestServerlessJobsService_GetJobDeploymentScaling(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	ctx := context.Background()
	scaling, err := client.ServerlessJobs.GetJobDeploymentScaling(ctx, "flux-training")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scaling.MaxReplicaCount != 2 {
		t.Fatalf("expected max replica count 2, got %d", scaling.MaxReplicaCount)
	}
	if scaling.DeadlineSeconds != 3600 {
		t.Fatalf("expected deadline seconds 3600, got %d", scaling.DeadlineSeconds)
	}
}

func TestServerlessJobsService_GetJobDeploymentStatus(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	ctx := context.Background()
	status, err := client.ServerlessJobs.GetJobDeploymentStatus(ctx, "flux-training")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "running" {
		t.Fatalf("expected status running, got %s", status.Status)
	}
}

func TestServerlessJobsService_JobOperations(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)
	ctx := context.Background()

	if err := client.ServerlessJobs.PauseJobDeployment(ctx, "flux-training"); err != nil {
		t.Fatalf("unexpected error pausing job deployment: %v", err)
	}
	if err := client.ServerlessJobs.ResumeJobDeployment(ctx, "flux-training"); err != nil {
		t.Fatalf("unexpected error resuming job deployment: %v", err)
	}
	if err := client.ServerlessJobs.PurgeJobDeploymentQueue(ctx, "flux-training"); err != nil {
		t.Fatalf("unexpected error purging job deployment queue: %v", err)
	}
}
