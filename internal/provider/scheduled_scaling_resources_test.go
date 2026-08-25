// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"terraform-provider-sss/internal/client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValkeyAndAuroraToClientModelMappings(t *testing.T) {
	t.Run("valkey replicas", func(t *testing.T) {
		model := valkeyReplicaScalingResourceModel{
			ServiceID: types.StringValue("replication-group"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			ReplicaCount: &valkeyReplicaCountModel{Low: types.Int64Value(0), Medium: types.Int64Value(1), High: types.Int64Value(2), Extreme: types.Int64Value(3)},
		}
		serviceID, got := model.ToClientModel()
		want := client.ValkeyReplicaScalingPostBody{Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3}
		if serviceID != "replication-group" || !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %q, %#v; want %q, %#v", serviceID, got, "replication-group", want)
		}
	})

	t.Run("valkey shards", func(t *testing.T) {
		model := valkeyShardScalingResourceModel{
			ServiceID: types.StringValue("replication-group"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			Capacity: &valkeyShardScalingCapacityModel{
				Low: valkeyShardCapacityModel{MinShardCount: types.Int64Value(11), MaxShardCount: types.Int64Value(12)}, Medium: valkeyShardCapacityModel{MinShardCount: types.Int64Value(21), MaxShardCount: types.Int64Value(22)},
				High: valkeyShardCapacityModel{MinShardCount: types.Int64Value(31), MaxShardCount: types.Int64Value(32)}, Extreme: valkeyShardCapacityModel{MinShardCount: types.Int64Value(41), MaxShardCount: types.Int64Value(42)},
			},
		}
		serviceID, got := model.ToClientModel()
		want := client.ValkeyShardScalingPostBody{
			Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
			LowCapacity: client.ValkeyShardCapacity{MinShardCount: 11, MaxShardCount: 12}, MediumCapacity: client.ValkeyShardCapacity{MinShardCount: 21, MaxShardCount: 22}, HighCapacity: client.ValkeyShardCapacity{MinShardCount: 31, MaxShardCount: 32}, ExtremeCapacity: client.ValkeyShardCapacity{MinShardCount: 41, MaxShardCount: 42},
		}
		if serviceID != "replication-group" || !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %q, %#v; want %q, %#v", serviceID, got, "replication-group", want)
		}
	})

	t.Run("aurora readers", func(t *testing.T) {
		model := auroraReaderScalingResourceModel{
			ServiceID: types.StringValue("cluster"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			Capacity: &auroraReaderScalingCapacityModel{
				Low: auroraReaderCapacityModel{MinReaders: types.Int64Value(1), MaxReaders: types.Int64Value(2)}, Medium: auroraReaderCapacityModel{MinReaders: types.Int64Value(2), MaxReaders: types.Int64Value(3)},
				High: auroraReaderCapacityModel{MinReaders: types.Int64Value(3), MaxReaders: types.Int64Value(4)}, Extreme: auroraReaderCapacityModel{MinReaders: types.Int64Value(4), MaxReaders: types.Int64Value(5)},
			},
		}
		serviceID, got := model.ToClientModel()
		want := client.AuroraReaderScalingPostBody{
			Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
			LowCapacity: client.AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: client.AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: client.AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: client.AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5},
		}
		if serviceID != "cluster" || !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %q, %#v; want %q, %#v", serviceID, got, "cluster", want)
		}
	})
}

func TestValkeyAndAuroraToResourceModelMappings(t *testing.T) {
	t.Run("valkey replicas", func(t *testing.T) {
		got := ToValkeyReplicaScalingResourceModel(&client.ValkeyReplicaScalingResponse{
			ServiceID: "replication-group", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3,
		})
		want := valkeyReplicaScalingResourceModel{
			ServiceID: types.StringValue("replication-group"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			ReplicaCount: &valkeyReplicaCountModel{Low: types.Int64Value(0), Medium: types.Int64Value(1), High: types.Int64Value(2), Extreme: types.Int64Value(3)},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %#v, want %#v", got, want)
		}
	})

	t.Run("valkey shards", func(t *testing.T) {
		got := ToValkeyShardScalingResourceModel(&client.ValkeyShardScalingResponse{
			ServiceID: "replication-group", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
			LowCapacity: client.ValkeyShardCapacity{MinShardCount: 11, MaxShardCount: 12}, MediumCapacity: client.ValkeyShardCapacity{MinShardCount: 21, MaxShardCount: 22}, HighCapacity: client.ValkeyShardCapacity{MinShardCount: 31, MaxShardCount: 32}, ExtremeCapacity: client.ValkeyShardCapacity{MinShardCount: 41, MaxShardCount: 42},
		})
		want := valkeyShardScalingResourceModel{
			ServiceID: types.StringValue("replication-group"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			Capacity: &valkeyShardScalingCapacityModel{
				Low: valkeyShardCapacityModel{MinShardCount: types.Int64Value(11), MaxShardCount: types.Int64Value(12)}, Medium: valkeyShardCapacityModel{MinShardCount: types.Int64Value(21), MaxShardCount: types.Int64Value(22)},
				High: valkeyShardCapacityModel{MinShardCount: types.Int64Value(31), MaxShardCount: types.Int64Value(32)}, Extreme: valkeyShardCapacityModel{MinShardCount: types.Int64Value(41), MaxShardCount: types.Int64Value(42)},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %#v, want %#v", got, want)
		}
	})

	t.Run("aurora readers", func(t *testing.T) {
		got := ToAuroraReaderScalingResourceModel(&client.AuroraReaderScalingResponse{
			ServiceID: "cluster", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
			LowCapacity: client.AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: client.AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: client.AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: client.AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5},
		})
		want := auroraReaderScalingResourceModel{
			ServiceID: types.StringValue("cluster"), Region: types.StringValue("eu-west-1"), ScaleUpLeadTimeMinutes: types.Int64Value(60),
			Capacity: &auroraReaderScalingCapacityModel{
				Low: auroraReaderCapacityModel{MinReaders: types.Int64Value(1), MaxReaders: types.Int64Value(2)}, Medium: auroraReaderCapacityModel{MinReaders: types.Int64Value(2), MaxReaders: types.Int64Value(3)},
				High: auroraReaderCapacityModel{MinReaders: types.Int64Value(3), MaxReaders: types.Int64Value(4)}, Extreme: auroraReaderCapacityModel{MinReaders: types.Int64Value(4), MaxReaders: types.Int64Value(5)},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("mapping = %#v, want %#v", got, want)
		}
	})
}

func TestValkeyAndAuroraLeadTimeSchema(t *testing.T) {
	resources := []struct {
		name     string
		resource resource.Resource
	}{
		{name: "valkey replicas", resource: NewValkeyReplicaScalingResource()},
		{name: "valkey shards", resource: NewValkeyShardScalingResource()},
		{name: "aurora readers", resource: NewAuroraReaderScalingResource()},
	}

	for _, testCase := range resources {
		t.Run(testCase.name, func(t *testing.T) {
			var schemaResponse resource.SchemaResponse
			testCase.resource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
			leadTime, ok := schemaResponse.Schema.Attributes["scale_up_lead_time_minutes"].(schema.Int64Attribute)
			if !ok {
				t.Fatal("lead time schema is not an Int64 attribute")
			}
			if !leadTime.Optional || !leadTime.Computed {
				t.Error("lead time must be optional with a provider default")
			}
			if len(leadTime.Validators) != 1 {
				t.Fatalf("lead time validator count = %d, want 1", len(leadTime.Validators))
			}

			var defaultResponse defaults.Int64Response
			leadTime.Default.DefaultInt64(context.Background(), defaults.Int64Request{Path: path.Root("scale_up_lead_time_minutes")}, &defaultResponse)
			if defaultResponse.Diagnostics.HasError() || defaultResponse.PlanValue.ValueInt64() != 0 {
				t.Errorf("default = %v with diagnostics %v, want 0 without errors", defaultResponse.PlanValue, defaultResponse.Diagnostics)
			}

			for _, value := range []struct {
				value int64
				valid bool
			}{{0, true}, {10080, true}, {-1, false}, {10081, false}} {
				var validationResponse frameworkvalidator.Int64Response
				leadTime.Validators[0].ValidateInt64(context.Background(), frameworkvalidator.Int64Request{
					Path:        path.Root("scale_up_lead_time_minutes"),
					ConfigValue: types.Int64Value(value.value),
				}, &validationResponse)
				if got := !validationResponse.Diagnostics.HasError(); got != value.valid {
					t.Errorf("validation for %d = %t, want %t", value.value, got, value.valid)
				}
			}

			assertNoCapacityValidators(t, schemaResponse.Schema.Attributes)
		})
	}
}

func TestValkeyShardCapacitySchema(t *testing.T) {
	var schemaResponse resource.SchemaResponse
	NewValkeyShardScalingResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)

	if _, exists := schemaResponse.Schema.Attributes["min_shard_count"]; exists {
		t.Error("legacy min_shard_count attribute is present")
	}
	capacity, ok := schemaResponse.Schema.Attributes["capacity"].(schema.SingleNestedAttribute)
	if !ok || !capacity.Required {
		t.Fatal("capacity is not a required nested attribute")
	}
	for _, level := range []string{"low", "medium", "high", "extreme"} {
		levelCapacity, ok := capacity.Attributes[level].(schema.SingleNestedAttribute)
		if !ok || !levelCapacity.Required {
			t.Fatalf("capacity.%s is not a required nested attribute", level)
		}
		if len(levelCapacity.Attributes) != 2 {
			t.Fatalf("capacity.%s field count = %d, want 2", level, len(levelCapacity.Attributes))
		}
		for _, field := range []string{"min_shard_count", "max_shard_count"} {
			bound, ok := levelCapacity.Attributes[field].(schema.Int64Attribute)
			if !ok || !bound.Required {
				t.Errorf("capacity.%s.%s is not a required Int64 attribute", level, field)
			}
			if len(bound.Validators) != 0 {
				t.Errorf("capacity.%s.%s has capacity validation", level, field)
			}
		}
	}
}

func assertNoCapacityValidators(t *testing.T, attributes map[string]schema.Attribute) {
	t.Helper()
	for _, name := range []string{"replica_count"} {
		if attribute, exists := attributes[name]; exists {
			nested, ok := attribute.(schema.SingleNestedAttribute)
			if !ok {
				t.Fatalf("%s is not a nested attribute", name)
			}
			for level, nestedAttribute := range nested.Attributes {
				capacity, ok := nestedAttribute.(schema.Int64Attribute)
				if !ok {
					t.Fatalf("%s.%s is not an Int64 attribute", name, level)
				}
				if len(capacity.Validators) != 0 {
					t.Errorf("%s.%s has capacity validation", name, level)
				}
			}
			return
		}
	}

	capacityAttribute, exists := attributes["capacity"]
	if !exists {
		t.Fatal("capacity attribute is missing")
	}
	capacity, ok := capacityAttribute.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("capacity is not a nested attribute")
	}
	for level, nestedAttribute := range capacity.Attributes {
		readers, ok := nestedAttribute.(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("capacity.%s is not a nested attribute", level)
		}
		for field, readerAttribute := range readers.Attributes {
			reader, ok := readerAttribute.(schema.Int64Attribute)
			if !ok {
				t.Fatalf("capacity.%s.%s is not an Int64 attribute", level, field)
			}
			if len(reader.Validators) != 0 {
				t.Errorf("capacity.%s.%s has capacity validation", level, field)
			}
		}
	}
}

type scalingResourceLifecycleTest struct {
	name         string
	resource     resource.Resource
	configure    func(*testing.T, resource.Resource, *client.SssClient)
	partialState any
	response     string
	assertState  func(*testing.T, tfsdk.State)
	lastUpdated  func(*testing.T, tfsdk.State) types.String
}

func TestScalingResourcesLifecycleFromPartialImportState(t *testing.T) {
	ctx := context.Background()
	for _, testCase := range scalingResourceLifecycleTests() {
		t.Run(testCase.name, func(t *testing.T) {
			requests := make(map[string]int)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests[r.Method]++
				switch r.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(testCase.response))
				case http.MethodPost:
					w.WriteHeader(http.StatusCreated)
				case http.MethodPut, http.MethodDelete:
					w.WriteHeader(http.StatusOK)
				default:
					t.Errorf("unexpected method %q", r.Method)
				}
			}))
			defer server.Close()

			testCase.configure(t, testCase.resource, testResourceClient(t, server))
			schemaResponse := resource.SchemaResponse{}
			testCase.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			partialState := tfsdk.State{Schema: schemaResponse.Schema}
			if diags := partialState.Set(ctx, testCase.partialState); diags.HasError() {
				t.Fatalf("set partial import state: %v", diags)
			}

			importingResource, ok := testCase.resource.(resource.ResourceWithImportState)
			if !ok {
				t.Fatal("resource does not support import")
			}
			importResponse := resource.ImportStateResponse{State: partialState}
			importingResource.ImportState(ctx, resource.ImportStateRequest{ID: "service"}, &importResponse)
			if importResponse.Diagnostics.HasError() {
				t.Fatalf("import: %v", importResponse.Diagnostics)
			}

			readResponse := resource.ReadResponse{State: importResponse.State}
			testCase.resource.Read(ctx, resource.ReadRequest{State: importResponse.State}, &readResponse)
			if readResponse.Diagnostics.HasError() {
				t.Fatalf("read partial import state: %v", readResponse.Diagnostics)
			}
			testCase.assertState(t, readResponse.State)

			createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			testCase.resource.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Raw: readResponse.State.Raw, Schema: readResponse.State.Schema}}, &createResponse)
			if createResponse.Diagnostics.HasError() {
				t.Fatalf("create: %v", createResponse.Diagnostics)
			}
			if testCase.lastUpdated(t, createResponse.State).IsNull() {
				t.Error("create did not set last_updated")
			}

			refreshResponse := resource.ReadResponse{State: createResponse.State}
			testCase.resource.Read(ctx, resource.ReadRequest{State: createResponse.State}, &refreshResponse)
			if refreshResponse.Diagnostics.HasError() {
				t.Fatalf("refresh: %v", refreshResponse.Diagnostics)
			}
			if got, want := testCase.lastUpdated(t, refreshResponse.State), testCase.lastUpdated(t, createResponse.State); !got.Equal(want) {
				t.Errorf("last_updated after refresh = %v, want %v", got, want)
			}

			updateResponse := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			testCase.resource.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Raw: refreshResponse.State.Raw, Schema: refreshResponse.State.Schema}}, &updateResponse)
			if updateResponse.Diagnostics.HasError() {
				t.Fatalf("update: %v", updateResponse.Diagnostics)
			}
			if testCase.lastUpdated(t, updateResponse.State).IsNull() {
				t.Error("update did not set last_updated")
			}

			deleteResponse := resource.DeleteResponse{State: updateResponse.State}
			testCase.resource.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
			if deleteResponse.Diagnostics.HasError() {
				t.Fatalf("delete: %v", deleteResponse.Diagnostics)
			}
			for method, want := range map[string]int{http.MethodGet: 2, http.MethodPost: 1, http.MethodPut: 1, http.MethodDelete: 1} {
				if got := requests[method]; got != want {
					t.Errorf("%s requests = %d, want %d", method, got, want)
				}
			}
		})
	}
}

func TestScalingResourcesCreate422Diagnostic(t *testing.T) {
	ctx := context.Background()
	for _, testCase := range scalingResourceLifecycleTests() {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"detail":"invalid capacity","errors":[{"location":"capacity.low","message":"must be valid"}]}`))
			}))
			defer server.Close()

			testCase.configure(t, testCase.resource, testResourceClient(t, server))
			schemaResponse := resource.SchemaResponse{}
			testCase.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			plan := tfsdk.Plan{Schema: schemaResponse.Schema}
			if diags := plan.Set(ctx, testCaseStateModel(testCase)); diags.HasError() {
				t.Fatalf("set plan: %v", diags)
			}
			createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			testCase.resource.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)
			if !createResponse.Diagnostics.HasError() {
				t.Fatal("create succeeded, want 422 diagnostic")
			}
			if detail := createResponse.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "invalid capacity") || !strings.Contains(detail, "capacity.low: must be valid") {
				t.Errorf("diagnostic = %q, want 422 details", detail)
			}
		})
	}
}

func testResourceClient(t *testing.T, server *httptest.Server) *client.SssClient {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client.NewSssClient(serverURL.Host, serverURL.Scheme, "user", "pass")
}

func scalingResourceLifecycleTests() []scalingResourceLifecycleTest {
	return []scalingResourceLifecycleTest{
		{
			name:     "valkey replicas",
			resource: NewValkeyReplicaScalingResource(),
			configure: func(t *testing.T, r resource.Resource, c *client.SssClient) {
				t.Helper()
				configured, ok := r.(*valkeyReplicaScalingResource)
				if !ok {
					t.Fatalf("resource type = %T, want *valkeyReplicaScalingResource", r)
				}
				configured.client = c
			},
			partialState: valkeyReplicaScalingResourceModel{ServiceID: types.StringNull(), Region: types.StringNull(), ScaleUpLeadTimeMinutes: types.Int64Null(), LastUpdated: types.StringNull()},
			response:     `{"serviceId":"service","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"replicaCountLow":0,"replicaCountMedium":1,"replicaCountHigh":2,"replicaCountExtreme":3}`,
			assertState: func(t *testing.T, state tfsdk.State) {
				var got valkeyReplicaScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				want := ToValkeyReplicaScalingResourceModel(&client.ValkeyReplicaScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3})
				if !reflect.DeepEqual(got, want) {
					t.Errorf("state = %#v, want %#v", got, want)
				}
			},
			lastUpdated: func(t *testing.T, state tfsdk.State) types.String {
				var got valkeyReplicaScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				return got.LastUpdated
			},
		},
		{
			name:     "valkey shards",
			resource: NewValkeyShardScalingResource(),
			configure: func(t *testing.T, r resource.Resource, c *client.SssClient) {
				t.Helper()
				configured, ok := r.(*valkeyShardScalingResource)
				if !ok {
					t.Fatalf("resource type = %T, want *valkeyShardScalingResource", r)
				}
				configured.client = c
			},
			partialState: valkeyShardScalingResourceModel{ServiceID: types.StringNull(), Region: types.StringNull(), ScaleUpLeadTimeMinutes: types.Int64Null(), LastUpdated: types.StringNull()},
			response:     `{"serviceId":"service","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"lowCapacity":{"minShardCount":1,"maxShardCount":2},"mediumCapacity":{"minShardCount":2,"maxShardCount":3},"highCapacity":{"minShardCount":3,"maxShardCount":4},"extremeCapacity":{"minShardCount":4,"maxShardCount":5}}`,
			assertState: func(t *testing.T, state tfsdk.State) {
				var got valkeyShardScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				want := ToValkeyShardScalingResourceModel(&client.ValkeyShardScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, LowCapacity: client.ValkeyShardCapacity{MinShardCount: 1, MaxShardCount: 2}, MediumCapacity: client.ValkeyShardCapacity{MinShardCount: 2, MaxShardCount: 3}, HighCapacity: client.ValkeyShardCapacity{MinShardCount: 3, MaxShardCount: 4}, ExtremeCapacity: client.ValkeyShardCapacity{MinShardCount: 4, MaxShardCount: 5}})
				if !reflect.DeepEqual(got, want) {
					t.Errorf("state = %#v, want %#v", got, want)
				}
			},
			lastUpdated: func(t *testing.T, state tfsdk.State) types.String {
				var got valkeyShardScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				return got.LastUpdated
			},
		},
		{
			name:     "aurora readers",
			resource: NewAuroraReaderScalingResource(),
			configure: func(t *testing.T, r resource.Resource, c *client.SssClient) {
				t.Helper()
				configured, ok := r.(*auroraReaderScalingResource)
				if !ok {
					t.Fatalf("resource type = %T, want *auroraReaderScalingResource", r)
				}
				configured.client = c
			},
			partialState: auroraReaderScalingResourceModel{ServiceID: types.StringNull(), Region: types.StringNull(), ScaleUpLeadTimeMinutes: types.Int64Null(), LastUpdated: types.StringNull()},
			response:     `{"serviceId":"service","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"lowCapacity":{"minReaders":1,"maxReaders":2},"mediumCapacity":{"minReaders":2,"maxReaders":3},"highCapacity":{"minReaders":3,"maxReaders":4},"extremeCapacity":{"minReaders":4,"maxReaders":5}}`,
			assertState: func(t *testing.T, state tfsdk.State) {
				var got auroraReaderScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				want := ToAuroraReaderScalingResourceModel(&client.AuroraReaderScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, LowCapacity: client.AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: client.AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: client.AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: client.AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5}})
				if !reflect.DeepEqual(got, want) {
					t.Errorf("state = %#v, want %#v", got, want)
				}
			},
			lastUpdated: func(t *testing.T, state tfsdk.State) types.String {
				var got auroraReaderScalingResourceModel
				if diags := state.Get(context.Background(), &got); diags.HasError() {
					t.Fatalf("get state: %v", diags)
				}
				return got.LastUpdated
			},
		},
	}
}

func testCaseStateModel(testCase scalingResourceLifecycleTest) any {
	switch testCase.name {
	case "valkey replicas":
		return ToValkeyReplicaScalingResourceModel(&client.ValkeyReplicaScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3})
	case "valkey shards":
		return ToValkeyShardScalingResourceModel(&client.ValkeyShardScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, LowCapacity: client.ValkeyShardCapacity{MinShardCount: 1, MaxShardCount: 2}, MediumCapacity: client.ValkeyShardCapacity{MinShardCount: 2, MaxShardCount: 3}, HighCapacity: client.ValkeyShardCapacity{MinShardCount: 3, MaxShardCount: 4}, ExtremeCapacity: client.ValkeyShardCapacity{MinShardCount: 4, MaxShardCount: 5}})
	default:
		return ToAuroraReaderScalingResourceModel(&client.AuroraReaderScalingResponse{ServiceID: "service", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, LowCapacity: client.AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: client.AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: client.AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: client.AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5}})
	}
}
