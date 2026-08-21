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

var (
	_ resource.Resource                = &auroraReaderScalingResource{}
	_ resource.ResourceWithConfigure   = &auroraReaderScalingResource{}
	_ resource.ResourceWithImportState = &auroraReaderScalingResource{}
)

type auroraReaderCapacityModel struct {
	MinReaders types.Int64 `tfsdk:"min_readers"`
	MaxReaders types.Int64 `tfsdk:"max_readers"`
}

type auroraReaderScalingCapacityModel struct {
	Low     auroraReaderCapacityModel `tfsdk:"low"`
	Medium  auroraReaderCapacityModel `tfsdk:"medium"`
	High    auroraReaderCapacityModel `tfsdk:"high"`
	Extreme auroraReaderCapacityModel `tfsdk:"extreme"`
}

type auroraReaderScalingResourceModel struct {
	ServiceID              types.String                      `tfsdk:"service_id"`
	Region                 types.String                      `tfsdk:"region"`
	ScaleUpLeadTimeMinutes types.Int64                       `tfsdk:"scale_up_lead_time_minutes"`
	Capacity               *auroraReaderScalingCapacityModel `tfsdk:"capacity"`
	LastUpdated            types.String                      `tfsdk:"last_updated"`
}

func (m *auroraReaderScalingResourceModel) ToClientModel() (string, client.AuroraReaderScalingPostBody) {
	return m.ServiceID.ValueString(), client.AuroraReaderScalingPostBody{
		Region:                 m.Region.ValueString(),
		ScaleUpLeadTimeMinutes: m.ScaleUpLeadTimeMinutes.ValueInt64(),
		LowCapacity: client.AuroraReaderCapacity{
			MinReaders: m.Capacity.Low.MinReaders.ValueInt64(),
			MaxReaders: m.Capacity.Low.MaxReaders.ValueInt64(),
		},
		MediumCapacity: client.AuroraReaderCapacity{
			MinReaders: m.Capacity.Medium.MinReaders.ValueInt64(),
			MaxReaders: m.Capacity.Medium.MaxReaders.ValueInt64(),
		},
		HighCapacity: client.AuroraReaderCapacity{
			MinReaders: m.Capacity.High.MinReaders.ValueInt64(),
			MaxReaders: m.Capacity.High.MaxReaders.ValueInt64(),
		},
		ExtremeCapacity: client.AuroraReaderCapacity{
			MinReaders: m.Capacity.Extreme.MinReaders.ValueInt64(),
			MaxReaders: m.Capacity.Extreme.MaxReaders.ValueInt64(),
		},
	}
}

func ToAuroraReaderScalingResourceModel(m *client.AuroraReaderScalingResponse) auroraReaderScalingResourceModel {
	return auroraReaderScalingResourceModel{
		ServiceID:              types.StringValue(m.ServiceID),
		Region:                 types.StringValue(m.Region),
		ScaleUpLeadTimeMinutes: types.Int64Value(m.ScaleUpLeadTimeMinutes),
		Capacity: &auroraReaderScalingCapacityModel{
			Low: auroraReaderCapacityModel{
				MinReaders: types.Int64Value(m.LowCapacity.MinReaders),
				MaxReaders: types.Int64Value(m.LowCapacity.MaxReaders),
			},
			Medium: auroraReaderCapacityModel{
				MinReaders: types.Int64Value(m.MediumCapacity.MinReaders),
				MaxReaders: types.Int64Value(m.MediumCapacity.MaxReaders),
			},
			High: auroraReaderCapacityModel{
				MinReaders: types.Int64Value(m.HighCapacity.MinReaders),
				MaxReaders: types.Int64Value(m.HighCapacity.MaxReaders),
			},
			Extreme: auroraReaderCapacityModel{
				MinReaders: types.Int64Value(m.ExtremeCapacity.MinReaders),
				MaxReaders: types.Int64Value(m.ExtremeCapacity.MaxReaders),
			},
		},
	}
}

func NewAuroraReaderScalingResource() resource.Resource {
	return &auroraReaderScalingResource{}
}

type auroraReaderScalingResource struct {
	client *client.SssClient
}

func (r *auroraReaderScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.SssClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.SssClient, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *auroraReaderScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aurora_reader_scaling"
}

func (r *auroraReaderScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	capacitySchema := schema.SingleNestedAttribute{
		Description: "Reader capacity for a schedule level. API-enforced: 1 <= min_readers <= max_readers <= 15, and both fields must be monotonic across levels.",
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"min_readers": schema.Int64Attribute{Required: true},
			"max_readers": schema.Int64Attribute{Required: true},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages scheduled reader scaling for an Aurora DB cluster. API-enforced: SSS owns the RDS Application Auto Scaling target; the cluster must contain at least one reader; configured maximums should be at least the number of fixed readers.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "The Aurora DB cluster identifier.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region containing the DB cluster.",
				Required:    true,
			},
			"scale_up_lead_time_minutes": schema.Int64Attribute{
				Description: "Minutes before a scheduled boundary to begin scale-up. Defaults to 0, which begins scaling at the boundary on the next service tick.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators:  []validator.Int64{int64validator.Between(0, 10080)},
			},
			"last_updated": schema.StringAttribute{Computed: true},
			"capacity": schema.SingleNestedAttribute{
				Description: "Reader capacity for each schedule level. Constraints are API-enforced.",
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

func (r *auroraReaderScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan auroraReaderScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.CreateAuroraReaderScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to create Aurora reader scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *auroraReaderScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state auroraReaderScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetAuroraReaderScaling(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Aurora reader scaling", "Could not read scaling for service "+state.ServiceID.ValueString()+": "+err.Error())
		return
	}
	newState := ToAuroraReaderScalingResourceModel(response)
	if !state.LastUpdated.IsNull() {
		newState.LastUpdated = state.LastUpdated
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *auroraReaderScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan auroraReaderScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.UpdateAuroraReaderScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to update Aurora reader scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *auroraReaderScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state auroraReaderScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAuroraReaderScaling(state.ServiceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Aurora reader scaling", err.Error())
	}
}

func (r *auroraReaderScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}
