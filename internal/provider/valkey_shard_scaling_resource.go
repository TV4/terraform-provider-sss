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
	_ resource.Resource                = &valkeyShardScalingResource{}
	_ resource.ResourceWithConfigure   = &valkeyShardScalingResource{}
	_ resource.ResourceWithImportState = &valkeyShardScalingResource{}
)

type valkeyShardCountModel struct {
	Low     types.Int64 `tfsdk:"low"`
	Medium  types.Int64 `tfsdk:"medium"`
	High    types.Int64 `tfsdk:"high"`
	Extreme types.Int64 `tfsdk:"extreme"`
}

type valkeyShardScalingResourceModel struct {
	ServiceID              types.String           `tfsdk:"service_id"`
	Region                 types.String           `tfsdk:"region"`
	ScaleUpLeadTimeMinutes types.Int64            `tfsdk:"scale_up_lead_time_minutes"`
	MinShardCount          *valkeyShardCountModel `tfsdk:"min_shard_count"`
	LastUpdated            types.String           `tfsdk:"last_updated"`
}

func (m *valkeyShardScalingResourceModel) ToClientModel() (string, client.ValkeyShardScalingPostBody) {
	return m.ServiceID.ValueString(), client.ValkeyShardScalingPostBody{
		Region:                 m.Region.ValueString(),
		ScaleUpLeadTimeMinutes: m.ScaleUpLeadTimeMinutes.ValueInt64(),
		MinShardCountLow:       m.MinShardCount.Low.ValueInt64(),
		MinShardCountMedium:    m.MinShardCount.Medium.ValueInt64(),
		MinShardCountHigh:      m.MinShardCount.High.ValueInt64(),
		MinShardCountExtreme:   m.MinShardCount.Extreme.ValueInt64(),
	}
}

func ToValkeyShardScalingResourceModel(m *client.ValkeyShardScalingResponse) valkeyShardScalingResourceModel {
	return valkeyShardScalingResourceModel{
		ServiceID:              types.StringValue(m.ServiceID),
		Region:                 types.StringValue(m.Region),
		ScaleUpLeadTimeMinutes: types.Int64Value(m.ScaleUpLeadTimeMinutes),
		MinShardCount: &valkeyShardCountModel{
			Low:     types.Int64Value(m.MinShardCountLow),
			Medium:  types.Int64Value(m.MinShardCountMedium),
			High:    types.Int64Value(m.MinShardCountHigh),
			Extreme: types.Int64Value(m.MinShardCountExtreme),
		},
	}
}

func NewValkeyShardScalingResource() resource.Resource {
	return &valkeyShardScalingResource{}
}

type valkeyShardScalingResource struct {
	client *client.SssClient
}

func (r *valkeyShardScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *valkeyShardScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_valkey_shard_scaling"
}

func (r *valkeyShardScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages scheduled minimum shard scaling for an ElastiCache Valkey replication group.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "The ElastiCache replication group identifier.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region containing the replication group.",
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
			"min_shard_count": schema.SingleNestedAttribute{
				Description: "Minimum shard counts for each schedule level. API-enforced: values must be greater than zero and low <= medium <= high <= extreme. The Application Auto Scaling target must already exist; SSS changes only its minimum capacity. Replica and shard resources may share a service_id.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"low":     schema.Int64Attribute{Required: true},
					"medium":  schema.Int64Attribute{Required: true},
					"high":    schema.Int64Attribute{Required: true},
					"extreme": schema.Int64Attribute{Required: true},
				},
			},
		},
	}
}

func (r *valkeyShardScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan valkeyShardScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.CreateValkeyShardScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to create Valkey shard scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyShardScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state valkeyShardScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetValkeyShardScaling(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Valkey shard scaling", "Could not read scaling for service "+state.ServiceID.ValueString()+": "+err.Error())
		return
	}
	newState := ToValkeyShardScalingResourceModel(response)
	if !state.LastUpdated.IsNull() {
		newState.LastUpdated = state.LastUpdated
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyShardScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan valkeyShardScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.UpdateValkeyShardScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to update Valkey shard scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyShardScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state valkeyShardScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteValkeyShardScaling(state.ServiceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Valkey shard scaling", err.Error())
	}
}

func (r *valkeyShardScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}
