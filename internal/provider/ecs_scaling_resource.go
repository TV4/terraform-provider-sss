// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-sss/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ecsScalingResource{}
	_ resource.ResourceWithConfigure   = &ecsScalingResource{}
	_ resource.ResourceWithImportState = &ecsScalingResource{}
)

type ecsScalingResourceModel struct {
	ServiceID              types.String             `tfsdk:"service_id"`
	Region                 types.String             `tfsdk:"region"`
	MinTasks               *ecsScalingCapacityModel `tfsdk:"min_tasks"`
	ScaleUpTasksPerMinute  types.Int64              `tfsdk:"scale_up_tasks_per_minute"`
	ScaleUpLeadTimeMinutes types.Int64              `tfsdk:"scale_up_lead_time_minutes"`
	LastUpdated            types.String             `tfsdk:"last_updated"`
}

type ecsScalingCapacityModel struct {
	Min     types.Int64 `tfsdk:"low"`
	Medium  types.Int64 `tfsdk:"medium"`
	High    types.Int64 `tfsdk:"high"`
	Extreme types.Int64 `tfsdk:"extreme"`
}

func (m *ecsScalingResourceModel) ToClientModel() (string, client.EcsServicePostBody) {
	return m.ServiceID.ValueString(), client.EcsServicePostBody{
		MinLowCapacity:         m.MinTasks.Min.ValueInt64(),
		MinMediumCapacity:      m.MinTasks.Medium.ValueInt64(),
		MinHighCapacity:        m.MinTasks.High.ValueInt64(),
		MinExtremeCapacity:     m.MinTasks.Extreme.ValueInt64(),
		ScaleUpTasksPerMinute:  m.ScaleUpTasksPerMinute.ValueInt64(),
		ScaleUpLeadTimeMinutes: m.ScaleUpLeadTimeMinutes.ValueInt64(),
		Region:                 m.Region.ValueString(),
	}
}

func ToECSResourceModel(m *client.EcsServiceResponse) ecsScalingResourceModel {
	return ecsScalingResourceModel{
		ServiceID:              types.StringValue(m.Name),
		Region:                 types.StringValue(m.Region),
		ScaleUpTasksPerMinute:  types.Int64Value(m.ScaleUpTasksPerMinute),
		ScaleUpLeadTimeMinutes: types.Int64Value(m.ScaleUpLeadTimeMinutes),
		MinTasks: &ecsScalingCapacityModel{
			Min:     types.Int64Value(m.MinLowCapacity),
			Medium:  types.Int64Value(m.MinMediumCapacity),
			High:    types.Int64Value(m.MinHighCapacity),
			Extreme: types.Int64Value(m.MinExtremeCapacity),
		},
	}
}

// NewcsScalingResource is a helper function to simplify the provider implementation.
func NewEcsScalingResource() resource.Resource {
	return &ecsScalingResource{}
}

func (r *ecsScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Terraform sets this after it calls ConfigureProvider
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.SssClient)

	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.SssClient, got: %T. Please report this issue to the provider developers.", req.ProviderData))

		return
	}
	r.client = client
}

// ecsScalingResource is the resource implementation.
type ecsScalingResource struct {
	client *client.SssClient
}

// Metadata returns the resource type name.
func (r *ecsScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ecs_scaling"
}

// Schema defines the schema for the resource.
func (r *ecsScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages scheduled minimum task capacity for ECS services. SSS applies configured capacities to Application Auto Scaling MinCapacity; schedules are configured separately.

A nonzero scale_up_tasks_per_minute paces scheduled increases. For a settled increase from 4 to 18 at a rate of 2 tasks per minute, the increase takes seven minutes. A lead time of 7 starts that ramp seven minutes before the schedule boundary; a shorter lead does not accelerate it. Lead time alone advances the selected capacity as a jump. During its look-ahead window SSS selects the highest capacity, so short down/up cycles can retain the higher minimum.

SSS recalculates using elapsed time every 20 seconds. Missed ticks or restarts can cause catch-up jumps, and decreases in the computed minimum are not ramped. Pacing does not guarantee task readiness or actual task start times and does not limit deployments or independent autoscaling.

Both settings default to zero and zero disables each setting, preserving legacy behavior. If either setting is nonzero, SSS requires all four capacities to be non-negative; for a positive rate, the spread between the highest and lowest configured capacities must take no more than 120 minutes at that rate. Capacities need not be monotonic. These cross-field rules are API-enforced before writes; invalid requests return HTTP 422.

Updates use PUT, which replaces the configuration: removing either setting resets it to zero while retaining the other configured value. Older providers omit both fields and therefore reset both on update. Configure imported nonzero values to retain them.`,
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "The service ID. Should be in format CLUSTER_NAME/SERICE_NAME",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region the service is located in. E.g. eu-west-1",
				Required:    true,
			},
			"scale_up_tasks_per_minute": schema.Int64Attribute{
				Description: "Maximum rate of scheduled minimum-capacity increases, in tasks per minute. Defaults to 0, which disables pacing. Must be non-negative. When positive, the spread across all four configured capacities must take no more than 120 minutes at this rate; SSS enforces this and conditional non-negative capacities through HTTP 422.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"scale_up_lead_time_minutes": schema.Int64Attribute{
				Description: "Minutes ahead of a scheduled boundary to select the highest capacity in the look-ahead window. Defaults to 0, which disables advance scaling; valid range is 0..10080. This advances a ramp but does not accelerate it; with no pacing, the higher minimum is applied as a jump. Short down/up cycles may retain the higher minimum.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators:  []validator.Int64{int64validator.Between(0, 10080)},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"min_tasks": schema.SingleNestedAttribute{
				Description: "The minimum number of tasks to have during different schedules.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"low": schema.Int64Attribute{
						Required: true,
					},
					"medium":  schema.Int64Attribute{Required: true},
					"high":    schema.Int64Attribute{Required: true},
					"extreme": schema.Int64Attribute{Required: true},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ecsScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ecsScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceName, capacities := plan.ToClientModel()

	err := r.client.CreateEcsService(serviceName, capacities)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ECS service scaling", err.Error())
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ecsScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ecsScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetEcsService(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read ECS service scaling", "Could not read scaling for service "+state.ServiceID.ValueString()+": "+err.Error())
		return
	}

	newState := ToECSResourceModel(response)

	if !state.LastUpdated.IsNull() {
		newState.LastUpdated = state.LastUpdated
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ecsScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ecsScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceName, capacities := plan.ToClientModel()
	err := r.client.UpdateEcsService(serviceName, capacities)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update ECS service scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ecsScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ecsScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteEcsService(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete ECS service scaling", err.Error())
		return
	}
}

func (r *ecsScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}
