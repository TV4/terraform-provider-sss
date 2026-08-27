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
	_ resource.Resource                = &valkeyReplicaScalingResource{}
	_ resource.ResourceWithConfigure   = &valkeyReplicaScalingResource{}
	_ resource.ResourceWithImportState = &valkeyReplicaScalingResource{}
)

type valkeyReplicaCountModel struct {
	Low     types.Int64 `tfsdk:"low"`
	Medium  types.Int64 `tfsdk:"medium"`
	High    types.Int64 `tfsdk:"high"`
	Extreme types.Int64 `tfsdk:"extreme"`
}

type valkeyReplicaScalingResourceModel struct {
	ServiceID              types.String             `tfsdk:"service_id"`
	Region                 types.String             `tfsdk:"region"`
	ScaleUpLeadTimeMinutes types.Int64              `tfsdk:"scale_up_lead_time_minutes"`
	ReplicaCount           *valkeyReplicaCountModel `tfsdk:"replica_count"`
	LastUpdated            types.String             `tfsdk:"last_updated"`
}

func (m *valkeyReplicaScalingResourceModel) ToClientModel() (string, client.ValkeyReplicaScalingPostBody) {
	return m.ServiceID.ValueString(), client.ValkeyReplicaScalingPostBody{
		Region:                 m.Region.ValueString(),
		ScaleUpLeadTimeMinutes: m.ScaleUpLeadTimeMinutes.ValueInt64(),
		ReplicaCountLow:        m.ReplicaCount.Low.ValueInt64(),
		ReplicaCountMedium:     m.ReplicaCount.Medium.ValueInt64(),
		ReplicaCountHigh:       m.ReplicaCount.High.ValueInt64(),
		ReplicaCountExtreme:    m.ReplicaCount.Extreme.ValueInt64(),
	}
}

func ToValkeyReplicaScalingResourceModel(m *client.ValkeyReplicaScalingResponse) valkeyReplicaScalingResourceModel {
	return valkeyReplicaScalingResourceModel{
		ServiceID:              types.StringValue(m.ServiceID),
		Region:                 types.StringValue(m.Region),
		ScaleUpLeadTimeMinutes: types.Int64Value(m.ScaleUpLeadTimeMinutes),
		ReplicaCount: &valkeyReplicaCountModel{
			Low:     types.Int64Value(m.ReplicaCountLow),
			Medium:  types.Int64Value(m.ReplicaCountMedium),
			High:    types.Int64Value(m.ReplicaCountHigh),
			Extreme: types.Int64Value(m.ReplicaCountExtreme),
		},
	}
}

func NewValkeyReplicaScalingResource() resource.Resource {
	return &valkeyReplicaScalingResource{}
}

type valkeyReplicaScalingResource struct {
	client *client.SssClient
}

func (r *valkeyReplicaScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *valkeyReplicaScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_valkey_replica_scaling"
}

func (r *valkeyReplicaScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages scheduled replica scaling for an ElastiCache Valkey replication group.",
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
			"replica_count": schema.SingleNestedAttribute{
				Description: "Replica counts for each schedule level. API-enforced: each value must be 0..5 and low <= medium <= high <= extreme; cluster-mode-enabled groups apply the count per shard; ElastiCache topology constraints apply.",
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

func (r *valkeyReplicaScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan valkeyReplicaScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.CreateValkeyReplicaScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to create Valkey replica scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyReplicaScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state valkeyReplicaScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetValkeyReplicaScaling(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Valkey replica scaling", "Could not read scaling for service "+state.ServiceID.ValueString()+": "+err.Error())
		return
	}
	newState := ToValkeyReplicaScalingResourceModel(response)
	if !state.LastUpdated.IsNull() {
		newState.LastUpdated = state.LastUpdated
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyReplicaScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan valkeyReplicaScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, body := plan.ToClientModel()
	if err := r.client.UpdateValkeyReplicaScaling(serviceID, body); err != nil {
		resp.Diagnostics.AddError("Failed to update Valkey replica scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *valkeyReplicaScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state valkeyReplicaScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteValkeyReplicaScaling(state.ServiceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Valkey replica scaling", err.Error())
	}
}

func (r *valkeyReplicaScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}
