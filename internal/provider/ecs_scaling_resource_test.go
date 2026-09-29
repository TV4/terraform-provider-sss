// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"terraform-provider-sss/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEcsScalingModelMappings(t *testing.T) {
	model := ecsScalingTestModel(2, 7)
	model.MinTasks = &ecsScalingCapacityModel{
		Min: types.Int64Value(10), Medium: types.Int64Value(0), High: types.Int64Value(7), Extreme: types.Int64Value(1),
	}
	serviceID, got := model.ToClientModel()
	want := client.EcsServicePostBody{
		Region: "eu-west-1", MinLowCapacity: 10, MinMediumCapacity: 0, MinHighCapacity: 7, MinExtremeCapacity: 1,
		ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
	}
	if serviceID != "service" || !reflect.DeepEqual(got, want) {
		t.Errorf("ToClientModel() = %q, %#v; want %q, %#v", serviceID, got, "service", want)
	}

	zeroModel := ecsScalingTestModel(0, 0)
	_, zeroBody := zeroModel.ToClientModel()
	if zeroBody.ScaleUpTasksPerMinute != 0 || zeroBody.ScaleUpLeadTimeMinutes != 0 {
		t.Errorf("zero ToClientModel() scale-up fields = %d/%d, want 0/0", zeroBody.ScaleUpTasksPerMinute, zeroBody.ScaleUpLeadTimeMinutes)
	}

	gotModel := ToECSResourceModel(&client.EcsServiceResponse{
		Name: "service", Region: "eu-west-1", MinLowCapacity: 10, MinMediumCapacity: 0, MinHighCapacity: 7, MinExtremeCapacity: 1,
		ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
	})
	if !reflect.DeepEqual(gotModel, model) {
		t.Errorf("ToECSResourceModel() = %#v, want %#v", gotModel, model)
	}

	legacyModel := ToECSResourceModel(&client.EcsServiceResponse{Name: "service", Region: "eu-west-1"})
	if legacyModel.ScaleUpTasksPerMinute.IsNull() || legacyModel.ScaleUpTasksPerMinute.IsUnknown() || legacyModel.ScaleUpTasksPerMinute.ValueInt64() != 0 ||
		legacyModel.ScaleUpLeadTimeMinutes.IsNull() || legacyModel.ScaleUpLeadTimeMinutes.IsUnknown() || legacyModel.ScaleUpLeadTimeMinutes.ValueInt64() != 0 {
		t.Errorf("legacy response mapped scale-up fields to %#v/%#v, want known zeros", legacyModel.ScaleUpTasksPerMinute, legacyModel.ScaleUpLeadTimeMinutes)
	}
}

func TestEcsScalingSchemaDefaultsAndBounds(t *testing.T) {
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	NewEcsScalingResource().Schema(ctx, resource.SchemaRequest{}, &schemaResponse)

	for _, testCase := range []struct {
		name   string
		bounds []struct {
			value int64
			valid bool
		}
	}{
		{name: "scale_up_tasks_per_minute", bounds: []struct {
			value int64
			valid bool
		}{{-1, false}, {0, true}, {2, true}}},
		{name: "scale_up_lead_time_minutes", bounds: []struct {
			value int64
			valid bool
		}{{-1, false}, {0, true}, {10080, true}, {10081, false}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			attribute, ok := schemaResponse.Schema.Attributes[testCase.name].(schema.Int64Attribute)
			if !ok {
				t.Fatalf("attribute has type %T, want Int64Attribute", schemaResponse.Schema.Attributes[testCase.name])
			}
			if !attribute.Optional || !attribute.Computed {
				t.Error("attribute must be optional and computed")
			}
			if len(attribute.Validators) != 1 {
				t.Fatalf("validator count = %d, want 1", len(attribute.Validators))
			}

			var defaultResponse defaults.Int64Response
			attribute.Default.DefaultInt64(ctx, defaults.Int64Request{Path: path.Root(testCase.name)}, &defaultResponse)
			if defaultResponse.Diagnostics.HasError() || defaultResponse.PlanValue.ValueInt64() != 0 {
				t.Errorf("default = %v, diagnostics %v; want zero without errors", defaultResponse.PlanValue, defaultResponse.Diagnostics)
			}

			for _, bound := range testCase.bounds {
				var validationResponse validator.Int64Response
				attribute.Validators[0].ValidateInt64(ctx, validator.Int64Request{
					Path: path.Root(testCase.name), ConfigValue: types.Int64Value(bound.value),
				}, &validationResponse)
				if got := !validationResponse.Diagnostics.HasError(); got != bound.valid {
					t.Errorf("validation for %d = %t, want %t (diagnostics %v)", bound.value, got, bound.valid, validationResponse.Diagnostics)
				}
			}

			for _, value := range []types.Int64{types.Int64Null(), types.Int64Unknown()} {
				var validationResponse validator.Int64Response
				attribute.Validators[0].ValidateInt64(ctx, validator.Int64Request{Path: path.Root(testCase.name), ConfigValue: value}, &validationResponse)
				if validationResponse.Diagnostics.HasError() {
					t.Errorf("validation for %v produced diagnostics: %v", value, validationResponse.Diagnostics)
				}
			}
		})
	}

	minTasks, ok := schemaResponse.Schema.Attributes["min_tasks"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("min_tasks is not a nested attribute")
	}
	for level, capacity := range minTasks.Attributes {
		attribute, ok := capacity.(schema.Int64Attribute)
		if !ok {
			t.Fatalf("min_tasks.%s has type %T, want Int64Attribute", level, capacity)
		}
		if len(attribute.Validators) != 0 {
			t.Errorf("min_tasks.%s has unexpected capacity validation", level)
		}
	}
}

func TestEcsScalingCRUDResetsAndRejectedUpdate(t *testing.T) {
	ctx := context.Background()
	resourceImpl, ok := NewEcsScalingResource().(*ecsScalingResource)
	if !ok {
		t.Fatal("NewEcsScalingResource returned an unexpected resource type")
	}
	var schemaResponse resource.SchemaResponse
	resourceImpl.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)

	apiRate, apiLead := int64(0), int64(0)
	rejectNextWrite := false
	var writes []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode %s body: %v", r.Method, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			writes = append(writes, body)
			if rejectNextWrite {
				rejectNextWrite = false
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"detail":"rejected"}`))
				return
			}
			rate, rateOK := body["scaleUpTasksPerMinute"].(float64)
			lead, leadOK := body["scaleUpLeadTimeMinutes"].(float64)
			if !rateOK || !leadOK {
				t.Errorf("scale-up fields must be present and numeric: %#v", body)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			apiRate = int64(rate)
			apiLead = int64(lead)
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"Name":"service","Region":"eu-west-1","MinLowCapacity":10,"MinMediumCapacity":0,"MinHighCapacity":7,"MinExtremeCapacity":1,"ScaleUpTasksPerMinute":%d,"ScaleUpLeadTimeMinutes":%d}`, apiRate, apiLead)
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %q", r.Method)
		}
	}))
	defer server.Close()
	resourceImpl.client = ecsScalingTestClient(t, server)

	initialPlan := ecsScalingTestModel(2, 7)
	initialPlan.MinTasks = &ecsScalingCapacityModel{
		Min: types.Int64Value(10), Medium: types.Int64Value(0), High: types.Int64Value(7), Extreme: types.Int64Value(1),
	}
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	resourceImpl.Create(ctx, resource.CreateRequest{Plan: ecsScalingTestPlan(t, schemaResponse.Schema, initialPlan)}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("Create diagnostics: %v", createResponse.Diagnostics)
	}
	created := ecsScalingReadState(t, createResponse.State)
	if created.ScaleUpTasksPerMinute.ValueInt64() != 2 || created.ScaleUpLeadTimeMinutes.ValueInt64() != 7 || created.LastUpdated.IsNull() {
		t.Errorf("created state = rate %v lead %v last_updated %v", created.ScaleUpTasksPerMinute, created.ScaleUpLeadTimeMinutes, created.LastUpdated)
	}

	readResponse := resource.ReadResponse{State: createResponse.State}
	resourceImpl.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", readResponse.Diagnostics)
	}
	refreshed := ecsScalingReadState(t, readResponse.State)
	if refreshed.ScaleUpTasksPerMinute.ValueInt64() != 2 || refreshed.ScaleUpLeadTimeMinutes.ValueInt64() != 7 {
		t.Errorf("refreshed scale-up values = %d/%d, want 2/7", refreshed.ScaleUpTasksPerMinute.ValueInt64(), refreshed.ScaleUpLeadTimeMinutes.ValueInt64())
	}
	if !refreshed.LastUpdated.Equal(created.LastUpdated) {
		t.Errorf("Read changed last_updated from %v to %v", created.LastUpdated, refreshed.LastUpdated)
	}

	for _, reset := range []struct {
		name       string
		rate, lead int64
	}{
		{name: "reset rate independently", rate: 0, lead: 7},
		{name: "reset lead independently", rate: 2, lead: 0},
		{name: "reset both", rate: 0, lead: 0},
	} {
		t.Run(reset.name, func(t *testing.T) {
			response := resource.UpdateResponse{State: readResponse.State}
			resourceImpl.Update(ctx, resource.UpdateRequest{
				Plan:  ecsScalingTestPlan(t, schemaResponse.Schema, ecsScalingTestModel(reset.rate, reset.lead)),
				State: readResponse.State,
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("Update diagnostics: %v", response.Diagnostics)
			}
			state := ecsScalingReadState(t, response.State)
			if state.ScaleUpTasksPerMinute.ValueInt64() != reset.rate || state.ScaleUpLeadTimeMinutes.ValueInt64() != reset.lead {
				t.Errorf("updated state = rate %d lead %d, want %d/%d", state.ScaleUpTasksPerMinute.ValueInt64(), state.ScaleUpLeadTimeMinutes.ValueInt64(), reset.rate, reset.lead)
			}
			readResponse.State = response.State
		})
	}

	priorState := readResponse.State
	rejectNextWrite = true
	rejectedPlan := ecsScalingTestPlan(t, schemaResponse.Schema, ecsScalingTestModel(2, 7))
	rejectedResponse := resource.UpdateResponse{State: priorState}
	resourceImpl.Update(ctx, resource.UpdateRequest{Plan: rejectedPlan, State: priorState}, &rejectedResponse)
	if !rejectedResponse.Diagnostics.HasError() {
		t.Fatal("rejected update returned no diagnostics")
	}
	if detail := rejectedResponse.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "422 Unprocessable Entity") || strings.Contains(detail, "rejected") {
		t.Errorf("ECS error diagnostic = %q; expected existing status-only 422 behavior", detail)
	}
	unchanged := ecsScalingReadState(t, rejectedResponse.State)
	if unchanged.ScaleUpTasksPerMinute.ValueInt64() != 0 || unchanged.ScaleUpLeadTimeMinutes.ValueInt64() != 0 {
		t.Errorf("failed update committed rejected values: %d/%d", unchanged.ScaleUpTasksPerMinute.ValueInt64(), unchanged.ScaleUpLeadTimeMinutes.ValueInt64())
	}

	if len(writes) != 5 {
		t.Fatalf("write count = %d, want create plus three resets plus rejected update", len(writes))
	}
	for i, want := range [][2]float64{{2, 7}, {0, 7}, {2, 0}, {0, 0}, {2, 7}} {
		rate, rateOK := writes[i]["scaleUpTasksPerMinute"].(float64)
		lead, leadOK := writes[i]["scaleUpLeadTimeMinutes"].(float64)
		if !rateOK || !leadOK {
			t.Fatalf("write %d scale-up fields must be present and numeric: %#v", i, writes[i])
		}
		if got := [2]float64{rate, lead}; got != want {
			t.Errorf("write %d scale-up fields = %v, want %v", i, got, want)
		}
	}
}

func TestEcsScalingImportRefreshDefaultsMissingFields(t *testing.T) {
	ctx := context.Background()
	resourceImpl, ok := NewEcsScalingResource().(*ecsScalingResource)
	if !ok {
		t.Fatal("NewEcsScalingResource returned an unexpected resource type")
	}
	var schemaResponse resource.SchemaResponse
	resourceImpl.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Name":"service","Region":"eu-west-1","MinLowCapacity":4,"MinMediumCapacity":10,"MinHighCapacity":18,"MinExtremeCapacity":18}`))
	}))
	defer server.Close()
	resourceImpl.client = ecsScalingTestClient(t, server)

	partialState := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := partialState.Set(ctx, ecsScalingResourceModel{
		ServiceID: types.StringNull(), Region: types.StringNull(), MinTasks: nil,
		ScaleUpTasksPerMinute: types.Int64Null(), ScaleUpLeadTimeMinutes: types.Int64Null(), LastUpdated: types.StringNull(),
	}); diags.HasError() {
		t.Fatalf("set partial import state: %v", diags)
	}
	importResponse := resource.ImportStateResponse{State: partialState}
	resourceImpl.ImportState(ctx, resource.ImportStateRequest{ID: "service"}, &importResponse)
	if importResponse.Diagnostics.HasError() {
		t.Fatalf("ImportState diagnostics: %v", importResponse.Diagnostics)
	}
	readResponse := resource.ReadResponse{State: importResponse.State}
	resourceImpl.Read(ctx, resource.ReadRequest{State: importResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", readResponse.Diagnostics)
	}
	state := ecsScalingReadState(t, readResponse.State)
	if state.ScaleUpTasksPerMinute.IsNull() || state.ScaleUpTasksPerMinute.IsUnknown() || state.ScaleUpTasksPerMinute.ValueInt64() != 0 ||
		state.ScaleUpLeadTimeMinutes.IsNull() || state.ScaleUpLeadTimeMinutes.IsUnknown() || state.ScaleUpLeadTimeMinutes.ValueInt64() != 0 {
		t.Errorf("legacy imported state = rate %v lead %v, want known zeros", state.ScaleUpTasksPerMinute, state.ScaleUpLeadTimeMinutes)
	}
}

func ecsScalingTestModel(rate, lead int64) ecsScalingResourceModel {
	return ecsScalingResourceModel{
		ServiceID: types.StringValue("service"), Region: types.StringValue("eu-west-1"),
		MinTasks:              &ecsScalingCapacityModel{Min: types.Int64Value(4), Medium: types.Int64Value(10), High: types.Int64Value(18), Extreme: types.Int64Value(18)},
		ScaleUpTasksPerMinute: types.Int64Value(rate), ScaleUpLeadTimeMinutes: types.Int64Value(lead), LastUpdated: types.StringNull(),
	}
}

func ecsScalingTestPlan(t *testing.T, resourceSchema schema.Schema, model ecsScalingResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return plan
}

func ecsScalingReadState(t *testing.T, state tfsdk.State) ecsScalingResourceModel {
	t.Helper()
	var model ecsScalingResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatalf("get resource state: %v", diags)
	}
	return model
}

func ecsScalingTestClient(t *testing.T, server *httptest.Server) *client.SssClient {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client.NewSssClient(serverURL.Host, serverURL.Scheme, "user", "pass")
}
