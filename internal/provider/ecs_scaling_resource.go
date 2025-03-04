package provider

import (
	"context"
	"fmt"
	"terraform-provider-sss/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ecsScalingResource{}
	_ resource.ResourceWithConfigure = &ecsScalingResource{}
)

type ecsScalingResourceModel struct {
	ServiceID   types.String            `tfsdk:"service_id"`
	MinTasks    ecsScalingCapacityModel `tfsdk:"min_tasks"`
	LastUpdated types.String            `tfsdk:"last_updated"`
}

type ecsScalingCapacityModel struct {
	Min     types.Int64 `tfsdk:"low"`
	Medium  types.Int64 `tfsdk:"medium"`
	High    types.Int64 `tfsdk:"high"`
	Extreme types.Int64 `tfsdk:"extreme"`
}

func (m *ecsScalingResourceModel) ToClientModel() (string, client.EcsServicePostBody) {
	return m.ServiceID.ValueString(), client.EcsServicePostBody{
		MinLowCapacity:     m.MinTasks.Min.ValueInt64(),
		MinMediumCapacity:  m.MinTasks.Medium.ValueInt64(),
		MinHighCapacity:    m.MinTasks.High.ValueInt64(),
		MinExtremeCapacity: m.MinTasks.Extreme.ValueInt64(),
	}
}

func ToResourceModel(m *client.EcsServiceResponse) ecsScalingResourceModel {
	return ecsScalingResourceModel{
		ServiceID: types.StringValue(m.Name),
		MinTasks: ecsScalingCapacityModel{
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
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"min_tasks": schema.SingleNestedAttribute{
				Required: true,
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

	responseModel := ToResourceModel(response)
	state.MinTasks = responseModel.MinTasks
	diags = resp.State.Set(ctx, state)
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
