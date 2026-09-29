// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Exercise the framework's planning RPC, not just the attribute validators.
// Terraform Core and a Terraform CLI installation are not needed for these tests.
func TestEcsScalingProtocolPlanPreservesUnknowns(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()

	for _, operation := range []string{"create", "update"} {
		for _, unknown := range []string{"rate", "lead", "both"} {
			t.Run(operation+"/"+unknown, func(t *testing.T) {
				config := ecsScalingTestModel(2, 7)
				if unknown == "rate" || unknown == "both" {
					config.ScaleUpTasksPerMinute = types.Int64Unknown()
				}
				if unknown == "lead" || unknown == "both" {
					config.ScaleUpLeadTimeMinutes = types.Int64Unknown()
				}
				proposed := config
				proposed.LastUpdated = types.StringUnknown()
				prior := &tfprotov6.DynamicValue{JSON: []byte("null")}
				if operation == "update" {
					prior = ecsScalingProtocolValue(t, ecsScalingTestModel(2, 7))
				}

				response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: "sss_ecs_scaling", PriorState: prior,
					Config: ecsScalingProtocolValue(t, config), ProposedNewState: ecsScalingProtocolValue(t, proposed),
				})
				if err != nil {
					t.Fatalf("PlanResourceChange: %v", err)
				}
				ecsScalingCheckProtocolDiagnostics(t, response.Diagnostics)
				planned := ecsScalingProtocolModel(t, response.PlannedState)
				if !planned.ScaleUpTasksPerMinute.Equal(config.ScaleUpTasksPerMinute) ||
					!planned.ScaleUpLeadTimeMinutes.Equal(config.ScaleUpLeadTimeMinutes) {
					t.Fatalf("plan changed configured scale-up values: got %v/%v, want %v/%v",
						planned.ScaleUpTasksPerMinute, planned.ScaleUpLeadTimeMinutes,
						config.ScaleUpTasksPerMinute, config.ScaleUpLeadTimeMinutes)
				}
				if len(response.RequiresReplace) != 0 {
					t.Fatalf("plan unexpectedly requires replacement: %v", response.RequiresReplace)
				}
			})
		}
	}
}

// This fixture has the pre-feature schema (version 0), with neither new field.
// It tests the provider's upgrade/read/plan RPCs, not a released provider binary.
func TestEcsScalingProtocolLegacyStateRefreshHasNoDiff(t *testing.T) {
	ctx := context.Background()
	const legacyState = `{"service_id":"service","region":"eu-west-1","min_tasks":{"low":4,"medium":10,"high":18,"extreme":18},"last_updated":"Mon Jan 02 15:04:05 MST 2006"}`

	for _, apiFields := range []string{"absent", "zero"} {
		t.Run(apiFields, func(t *testing.T) {
			var reads, writes atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writes.Add(1)
					t.Errorf("upgrade/refresh/plan must not write: %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				reads.Add(1)
				body := map[string]any{
					"Name": "service", "Region": "eu-west-1",
					"MinLowCapacity": 4, "MinMediumCapacity": 10, "MinHighCapacity": 18, "MinExtremeCapacity": 18,
				}
				if apiFields == "zero" {
					body["ScaleUpTasksPerMinute"] = 0
					body["ScaleUpLeadTimeMinutes"] = 0
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Errorf("encode GET response: %v", err)
				}
			}))
			defer api.Close()

			apiURL, err := url.Parse(api.URL)
			if err != nil {
				t.Fatal(err)
			}
			providerConfig, err := json.Marshal(map[string]string{
				"host": apiURL.Host, "protocol": apiURL.Scheme, "auth_username": "user", "auth_password": "pass",
			})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(New("test")())()
			configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{
				Config: &tfprotov6.DynamicValue{JSON: providerConfig},
			})
			if err != nil {
				t.Fatalf("ConfigureProvider: %v", err)
			}
			ecsScalingCheckProtocolDiagnostics(t, configured.Diagnostics)

			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "sss_ecs_scaling", Version: 0, RawState: &tfprotov6.RawState{JSON: []byte(legacyState)},
			})
			if err != nil {
				t.Fatalf("UpgradeResourceState: %v", err)
			}
			ecsScalingCheckProtocolDiagnostics(t, upgraded.Diagnostics)
			refreshed, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{
				TypeName: "sss_ecs_scaling", CurrentState: upgraded.UpgradedState,
			})
			if err != nil {
				t.Fatalf("ReadResource: %v", err)
			}
			ecsScalingCheckProtocolDiagnostics(t, refreshed.Diagnostics)
			want := ecsScalingTestModel(0, 0)
			want.LastUpdated = types.StringValue("Mon Jan 02 15:04:05 MST 2006")
			if got := ecsScalingProtocolModel(t, refreshed.NewState); !reflect.DeepEqual(got, want) {
				t.Fatalf("refreshed state = %#v, want %#v", got, want)
			}

			config := ecsScalingTestModel(0, 0)
			config.ScaleUpTasksPerMinute = types.Int64Null()
			config.ScaleUpLeadTimeMinutes = types.Int64Null()
			// Core keeps prior values for omitted optional/computed attributes
			// when constructing the proposed state. Config still contains nulls.
			planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "sss_ecs_scaling", PriorState: refreshed.NewState,
				Config: ecsScalingProtocolValue(t, config), ProposedNewState: refreshed.NewState,
			})
			if err != nil {
				t.Fatalf("PlanResourceChange: %v", err)
			}
			ecsScalingCheckProtocolDiagnostics(t, planned.Diagnostics)
			if got := ecsScalingProtocolModel(t, planned.PlannedState); !reflect.DeepEqual(got, want) {
				t.Fatalf("plan differs from refreshed state: got %#v, want %#v", got, want)
			}
			if len(planned.RequiresReplace) != 0 {
				t.Fatalf("plan unexpectedly requires replacement: %v", planned.RequiresReplace)
			}
			if reads.Load() != 1 || writes.Load() != 0 {
				t.Fatalf("API calls: %d reads and %d writes, want 1 read and no writes", reads.Load(), writes.Load())
			}
		})
	}
}

func ecsScalingProtocolSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	NewEcsScalingResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", response.Diagnostics)
	}
	return response.Schema
}

func ecsScalingProtocolValue(t *testing.T, model ecsScalingResourceModel) *tfprotov6.DynamicValue {
	t.Helper()
	plan := ecsScalingTestPlan(t, ecsScalingProtocolSchema(t), model)
	value, err := tfprotov6.NewDynamicValue(plan.Raw.Type(), plan.Raw)
	if err != nil {
		t.Fatalf("encode protocol value: %v", err)
	}
	return &value
}

func ecsScalingProtocolModel(t *testing.T, value *tfprotov6.DynamicValue) ecsScalingResourceModel {
	t.Helper()
	if value == nil {
		t.Fatal("missing protocol state/plan")
	}
	resourceSchema := ecsScalingProtocolSchema(t)
	raw, err := value.Unmarshal(resourceSchema.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatalf("decode protocol value: %v", err)
	}
	return ecsScalingReadState(t, tfsdk.State{Schema: resourceSchema, Raw: raw})
}

func ecsScalingCheckProtocolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("protocol error: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}
