// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-sss/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dynamoTableScalingResource{}
	_ resource.ResourceWithConfigure   = &dynamoTableScalingResource{}
	_ resource.ResourceWithImportState = &dynamoTableScalingResource{}
)

type dynamoTableCapacityValue struct {
	MinWriteCapacity types.Int64 `tfsdk:"min_write"`
	MinReadCapacity  types.Int64 `tfsdk:"min_read"`
	MaxWriteCapacity types.Int64 `tfsdk:"max_write"`
	MaxReadCapacity  types.Int64 `tfsdk:"max_read"`
}

type dynamoTableCapacityModel struct {
	Min     dynamoTableCapacityValue `tfsdk:"low"`
	Medium  dynamoTableCapacityValue `tfsdk:"medium"`
	High    dynamoTableCapacityValue `tfsdk:"high"`
	Extreme dynamoTableCapacityValue `tfsdk:"extreme"`
}

type dynamoTableScalingResourceModel struct {
	TableName   types.String             `tfsdk:"table_name"`
	Region      types.String             `tfsdk:"region"`
	Capacity    dynamoTableCapacityModel `tfsdk:"capacity"`
	LastUpdated types.String             `tfsdk:"last_updated"`
}

func (m *dynamoTableScalingResourceModel) ToClientModel() (string, client.DynamoTablePostBody) {
	return m.TableName.ValueString(), client.DynamoTablePostBody{
		Region: m.Region.ValueString(),
		LowCapacity: client.DynamoTableCapacity{
			MinWriteCapacity: m.Capacity.Min.MinWriteCapacity.ValueInt64(),
			MinReadCapacity:  m.Capacity.Min.MinReadCapacity.ValueInt64(),
			MaxWriteCapacity: m.Capacity.Min.MaxWriteCapacity.ValueInt64(),
			MaxReadCapacity:  m.Capacity.Min.MaxReadCapacity.ValueInt64(),
		},
		MediumCapacity: client.DynamoTableCapacity{
			MinWriteCapacity: m.Capacity.Medium.MinWriteCapacity.ValueInt64(),
			MinReadCapacity:  m.Capacity.Medium.MinReadCapacity.ValueInt64(),
			MaxWriteCapacity: m.Capacity.Medium.MaxWriteCapacity.ValueInt64(),
			MaxReadCapacity:  m.Capacity.Medium.MaxReadCapacity.ValueInt64(),
		},
		HighCapacity: client.DynamoTableCapacity{
			MinWriteCapacity: m.Capacity.High.MinWriteCapacity.ValueInt64(),
			MinReadCapacity:  m.Capacity.High.MinReadCapacity.ValueInt64(),
			MaxWriteCapacity: m.Capacity.High.MaxWriteCapacity.ValueInt64(),
			MaxReadCapacity:  m.Capacity.High.MaxReadCapacity.ValueInt64(),
		},
		ExtremeCapacity: client.DynamoTableCapacity{
			MinWriteCapacity: m.Capacity.Extreme.MinWriteCapacity.ValueInt64(),
			MinReadCapacity:  m.Capacity.Extreme.MinReadCapacity.ValueInt64(),
			MaxWriteCapacity: m.Capacity.Extreme.MaxWriteCapacity.ValueInt64(),
			MaxReadCapacity:  m.Capacity.Extreme.MaxReadCapacity.ValueInt64(),
		},
	}
}

func ToDynamoTableResourceModel(m *client.DynamoTableResponse) dynamoTableScalingResourceModel {
	return dynamoTableScalingResourceModel{
		TableName: types.StringValue(m.TableName),
		Region:    types.StringValue(m.Region),
		Capacity: dynamoTableCapacityModel{
			Min: dynamoTableCapacityValue{
				MinWriteCapacity: types.Int64Value(m.LowCapacity.MinWriteCapacity),
				MinReadCapacity:  types.Int64Value(m.LowCapacity.MinReadCapacity),
				MaxWriteCapacity: types.Int64Value(m.LowCapacity.MaxWriteCapacity),
				MaxReadCapacity:  types.Int64Value(m.LowCapacity.MaxReadCapacity),
			},
			Medium: dynamoTableCapacityValue{
				MinWriteCapacity: types.Int64Value(m.MediumCapacity.MinWriteCapacity),
				MinReadCapacity:  types.Int64Value(m.MediumCapacity.MinReadCapacity),
				MaxWriteCapacity: types.Int64Value(m.MediumCapacity.MaxWriteCapacity),
				MaxReadCapacity:  types.Int64Value(m.MediumCapacity.MaxReadCapacity),
			},
			High: dynamoTableCapacityValue{
				MinWriteCapacity: types.Int64Value(m.HighCapacity.MinWriteCapacity),
				MinReadCapacity:  types.Int64Value(m.HighCapacity.MinReadCapacity),
				MaxWriteCapacity: types.Int64Value(m.HighCapacity.MaxWriteCapacity),
				MaxReadCapacity:  types.Int64Value(m.HighCapacity.MaxReadCapacity),
			},
			Extreme: dynamoTableCapacityValue{
				MinWriteCapacity: types.Int64Value(m.ExtremeCapacity.MinWriteCapacity),
				MinReadCapacity:  types.Int64Value(m.ExtremeCapacity.MinReadCapacity),
				MaxWriteCapacity: types.Int64Value(m.ExtremeCapacity.MaxWriteCapacity),
				MaxReadCapacity:  types.Int64Value(m.ExtremeCapacity.MaxReadCapacity),
			},
		},
	}
}

// NewDynamoTableScalingResource is a helper function to simplify the provider implementation.
func NewDynamoTableScalingResource() resource.Resource {
	return &dynamoTableScalingResource{}
}

func (r *dynamoTableScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// dynamoTableScalingResource is the resource implementation.
type dynamoTableScalingResource struct {
	client *client.SssClient
}

// Metadata returns the resource type name.
func (r *dynamoTableScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dynamo_table_scaling"
}

// Schema defines the schema for the resource.
func (r *dynamoTableScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	capacitySchema := schema.SingleNestedAttribute{
		Description: "The capacity to use during the different schedules.",
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"min_write": schema.Int64Attribute{Required: true},
			"max_write": schema.Int64Attribute{Required: true},
			"min_read":  schema.Int64Attribute{Required: true},
			"max_read":  schema.Int64Attribute{Required: true},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages scaling for DynamoDB Tables.",
		Attributes: map[string]schema.Attribute{
			"table_name": schema.StringAttribute{
				Description: "The arn of the table",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region the service is located in. E.g. eu-west-1",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"capacity": schema.SingleNestedAttribute{
				Description: "The minimum number of tasks to have during different schedules.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"low":     capacitySchema,
					"medium":  capacitySchema,
					"high":    capacitySchema,
					"extreme": capacitySchema,
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *dynamoTableScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dynamoTableScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tableName, capacities := plan.ToClientModel()

	err := r.client.CreateDynamoTable(tableName, capacities)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dynamo table scaling", err.Error())
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
func (r *dynamoTableScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dynamoTableScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetDynamoTable(state.TableName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Dynamo DB table scaling", "Could not read scaling for table "+state.TableName.ValueString()+": "+err.Error())
		return
	}

	newState := ToDynamoTableResourceModel(response)

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
func (r *dynamoTableScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dynamoTableScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tableName, capacities := plan.ToClientModel()
	err := r.client.UpdateDynamoTable(tableName, capacities)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update dynamo table scaling", err.Error())
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
func (r *dynamoTableScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dynamoTableScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteDynamoTable(state.TableName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to dynamodb table scaling", err.Error())
		return
	}
}

func (r *dynamoTableScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("table_name"), req, resp)
}
