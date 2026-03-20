package verda

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"testing"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda/testutil"
)

func TestInstanceService_Get(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("get all instances", func(t *testing.T) {
		ctx := context.Background()
		instances, err := client.Instances.Get(ctx, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(instances) != 1 {
			t.Errorf("expected 1 instance, got %d", len(instances))
		}

		instance := instances[0]
		if instance.ID != "inst_123" {
			t.Errorf("expected instance ID 'inst_123', got '%s'", instance.ID)
		}

		if instance.InstanceType != "1V100.6V" {
			t.Errorf("expected instance type '1V100.6V', got '%s'", instance.InstanceType)
		}

		if instance.Status != StatusRunning {
			t.Errorf("expected status '%s', got '%s'", StatusRunning, instance.Status)
		}
	})

	t.Run("get instances with status filter", func(t *testing.T) {
		ctx := context.Background()
		instances, err := client.Instances.Get(ctx, StatusRunning)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(instances) != 1 {
			t.Errorf("expected 1 instance, got %d", len(instances))
		}
	})
}

func TestInstanceService_GetByID(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("get instance by ID", func(t *testing.T) {
		ctx := context.Background()
		instanceID := "test_instance_123"
		instance, err := client.Instances.GetByID(ctx, instanceID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if instance == nil {
			t.Fatal("expected instance, got nil")
		}

		if instance.ID != instanceID {
			t.Errorf("expected instance ID '%s', got '%s'", instanceID, instance.ID)
		}

		if instance.InstanceType != "1V100.6V" {
			t.Errorf("expected instance type '1V100.6V', got '%s'", instance.InstanceType)
		}
	})
}

func TestInstanceService_Create(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("create instance with minimal config", func(t *testing.T) {
		req := CreateInstanceRequest{
			InstanceType: "1V100.6V",
			Image:        "ubuntu-24.04-cuda-12.8-open-docker",
			Hostname:     "test-instance",
			Description:  "Test instance",
		}

		ctx := context.Background()
		instance, err := client.Instances.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if instance == nil {
			t.Fatal("expected instance, got nil")
		}

		if instance.ID != "inst_new_123" {
			t.Errorf("expected instance ID 'inst_new_123', got '%s'", instance.ID)
		}

		if instance.InstanceType != req.InstanceType {
			t.Errorf("expected instance type '%s', got '%s'", req.InstanceType, instance.InstanceType)
		}

		if instance.Hostname != req.Hostname {
			t.Errorf("expected hostname '%s', got '%s'", req.Hostname, instance.Hostname)
		}

		// Should set default location
		if instance.Location != LocationFIN01 {
			t.Errorf("expected location '%s', got '%s'", LocationFIN01, instance.Location)
		}
	})

	t.Run("create instance with full config", func(t *testing.T) {
		req := CreateInstanceRequest{
			InstanceType:    "8V100.48V",
			Image:           "custom-image",
			Hostname:        "ml-server",
			Description:     "ML training server",
			SSHKeyIDs:       []string{"key_123", "key_456"},
			LocationCode:    "US-01",
			Contract:        "PAY_AS_YOU_GO",
			Pricing:         "FIXED_PRICE",
			StartupScriptID: stringPtr("script_123"),
			Volumes: []VolumeCreateRequest{
				{Size: 500, Type: VolumeTypeNVMe, Name: "data"},
			},
			ExistingVolumes: []string{"vol_123"},
			OSVolume:        &OSVolumeCreateRequest{Size: 100, Name: "os-vol"},
			IsSpot:          true,
			Coupon:          stringPtr("DISCOUNT20"),
		}

		ctx := context.Background()
		instance, err := client.Instances.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if instance == nil {
			t.Fatal("expected instance, got nil")
		}

		if instance.Location != "US-01" {
			t.Errorf("expected location 'US-01', got '%s'", instance.Location)
		}

		if len(instance.SSHKeyIDs) != 2 {
			t.Errorf("expected 2 SSH keys, got %d", len(instance.SSHKeyIDs))
		}
	})
}

func TestInstanceService_CreateSpotWithDiscontinuePolicy(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("create spot instance with on_spot_discontinue on os_volume", func(t *testing.T) {
		req := CreateInstanceRequest{
			InstanceType: "CPU.4V.16G",
			Image:        "ubuntu-24.04",
			Hostname:     "test-spot-instance",
			SSHKeyIDs:    []string{"key_123"},
			LocationCode: "FIN-03",
			IsSpot:       true,
			OSVolume: &OSVolumeCreateRequest{
				Name:              "test-os-volume",
				Size:              55,
				OnSpotDiscontinue: SpotDiscontinueDeletePermanent,
			},
		}

		ctx := context.Background()
		instance, err := client.Instances.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if instance == nil {
			t.Fatal("expected instance, got nil")
		}
	})

	t.Run("create spot instance with on_spot_discontinue on additional volumes", func(t *testing.T) {
		req := CreateInstanceRequest{
			InstanceType: "CPU.4V.16G",
			Image:        "ubuntu-24.04",
			Hostname:     "test-spot-volumes",
			SSHKeyIDs:    []string{"key_123"},
			LocationCode: "FIN-03",
			IsSpot:       true,
			OSVolume: &OSVolumeCreateRequest{
				Name:              "test-os-volume",
				Size:              55,
				OnSpotDiscontinue: SpotDiscontinueKeepDetached,
			},
			Volumes: []VolumeCreateRequest{
				{
					Size:              500,
					Type:              VolumeTypeNVMe,
					Name:              "data-vol",
					OnSpotDiscontinue: SpotDiscontinueMoveToTrash,
				},
			},
		}

		ctx := context.Background()
		instance, err := client.Instances.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if instance == nil {
			t.Fatal("expected instance, got nil")
		}
	})
}

func TestInstanceService_Action(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("202 single instance returns results", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action: ActionShutdown,
			ID:     []string{"inst_123"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].InstanceID != "inst_123" {
			t.Errorf("expected instanceId 'inst_123', got '%s'", results[0].InstanceID)
		}
		if results[0].Status != "success" {
			t.Errorf("expected status 'success', got '%s'", results[0].Status)
		}
	})

	t.Run("202 multiple instances returns results", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action: ActionShutdown,
			ID:     []string{"inst_123", "inst_456"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("action delete with volumes", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action:    ActionDelete,
			ID:        []string{"inst_123"},
			VolumeIDs: []string{"vol_123"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("action delete with empty volume_ids (no volumes deleted)", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action:    ActionDelete,
			ID:        []string{"inst_123"},
			VolumeIDs: []string{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("action delete with nil volume_ids (API default)", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action: ActionDelete,
			ID:     []string{"inst_123"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("action delete with volumes permanently", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action:            ActionDelete,
			ID:                []string{"inst_123"},
			VolumeIDs:         []string{"vol_123"},
			DeletePermanently: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("action discontinue with volumes permanently", func(t *testing.T) {
		ctx := context.Background()
		results, err := client.Instances.Action(ctx, InstanceActionRequest{
			Action:            ActionDiscontinue,
			ID:                []string{"inst_123"},
			VolumeIDs:         []string{"vol_123"},
			DeletePermanently: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

}

func TestInstanceService_Action_204_AlreadyInState(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	mockServer.SetHandler("PUT", "/instances", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := NewTestClient(mockServer)
	ctx := context.Background()

	results, err := client.Instances.Action(ctx, InstanceActionRequest{
		Action: ActionBoot,
		ID:     []string{"inst_123"},
	})
	if err != nil {
		t.Fatalf("unexpected error for 204: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for 204, got %v", results)
	}
}

func TestInstanceService_Action_207_PartialFailure(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	mockServer.SetHandler("PUT", "/instances", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMultiStatus)
		writeTestJSON(w, []map[string]interface{}{
			{"action": "shutdown", "instanceId": "inst_123", "status": "success"},
			{"action": "shutdown", "instanceId": "inst_456", "status": "error", "error": "instance not found", "statusCode": 404},
		})
	})

	client := NewTestClient(mockServer)
	ctx := context.Background()

	results, err := client.Instances.Action(ctx, InstanceActionRequest{
		Action: ActionShutdown,
		ID:     []string{"inst_123", "inst_456"},
	})

	if err != nil {
		t.Fatalf("unexpected error for 207: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "success" {
		t.Errorf("expected first result success, got %s", results[0].Status)
	}
	if results[1].Status != "error" {
		t.Errorf("expected second result error, got %s", results[1].Status)
	}
	if results[1].Error != "instance not found" {
		t.Errorf("expected error message 'instance not found', got '%s'", results[1].Error)
	}
	if results[1].StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", results[1].StatusCode)
	}
}

func TestInstanceService_Action_400_BadRequest(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	mockServer.SetHandler("PUT", "/instances", func(w http.ResponseWriter, _ *http.Request) {
		testutil.ErrorResponse(w, http.StatusBadRequest, "action not allowed in current state")
	})

	client := NewTestClient(mockServer)
	ctx := context.Background()

	results, err := client.Instances.Action(ctx, InstanceActionRequest{
		Action: ActionBoot,
		ID:     []string{"inst_123"},
	})

	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}

	if results != nil {
		t.Errorf("expected nil results for 400, got %v", results)
	}
}

func TestInstanceService_Action_404_NotFound(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	mockServer.SetHandler("PUT", "/instances", func(w http.ResponseWriter, _ *http.Request) {
		testutil.ErrorResponse(w, http.StatusNotFound, "instance not found")
	})

	client := NewTestClient(mockServer)
	ctx := context.Background()

	results, err := client.Instances.Action(ctx, InstanceActionRequest{
		Action: ActionDelete,
		ID:     []string{"inst_nonexistent"},
	})

	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}

	if results != nil {
		t.Errorf("expected nil results for 404, got %v", results)
	}
}

func TestInstanceService_ConvenienceMethods(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	client := NewTestClient(mockServer)

	t.Run("boot instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Boot(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("start instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Start(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("shutdown instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Shutdown(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("delete instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Delete(ctx, []string{"inst_123"}, []string{"vol_123"}, false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("delete instance permanently", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Delete(ctx, []string{"inst_123"}, []string{"vol_123"}, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("discontinue instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Discontinue(ctx, []string{"inst_123"}, nil, false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("discontinue instance with volumes permanently", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Discontinue(ctx, []string{"inst_123"}, []string{"vol_123"}, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("hibernate instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Hibernate(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("configure spot instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.ConfigureSpot(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("force shutdown instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.ForceShutdown(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("delete stuck instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.DeleteStuck(ctx, []string{"vol_123"}, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("deploy instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Deploy(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("transfer instance", func(t *testing.T) {
		ctx := context.Background()
		err := client.Instances.Transfer(ctx, "inst_123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func writeTestJSON(w http.ResponseWriter, v interface{}) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("test: failed to encode JSON: %v", err)
	}
}

func stringPtr(s string) *string {
	return &s
}
