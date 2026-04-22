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
	_ resource.Resource                = &eksHpaScalingResource{}
	_ resource.ResourceWithConfigure   = &eksHpaScalingResource{}
	_ resource.ResourceWithImportState = &eksHpaScalingResource{}
)

type eksHpaScalingResourceModel struct {
	ServiceID   types.String                `tfsdk:"service_id"`
	Cluster     types.String                `tfsdk:"cluster"`
	Region      types.String                `tfsdk:"region"`
	Namespace   types.String                `tfsdk:"namespace"`
	Name        types.String                `tfsdk:"name"`
	Kind        types.String                `tfsdk:"kind"`
	MinReplicas *eksHpaMinReplicasModel     `tfsdk:"min_replicas"`
	LastUpdated types.String                `tfsdk:"last_updated"`
}

type eksHpaMinReplicasModel struct {
	Low     types.Int64 `tfsdk:"low"`
	Medium  types.Int64 `tfsdk:"medium"`
	High    types.Int64 `tfsdk:"high"`
	Extreme types.Int64 `tfsdk:"extreme"`
}

func (m *eksHpaScalingResourceModel) ToClientModel() (string, client.EksHpaPostBody) {
	return m.ServiceID.ValueString(), client.EksHpaPostBody{
		Cluster:    m.Cluster.ValueString(),
		Region:     m.Region.ValueString(),
		Namespace:  m.Namespace.ValueString(),
		Name:       m.Name.ValueString(),
		Kind:       m.Kind.ValueString(),
		MinLow:     m.MinReplicas.Low.ValueInt64(),
		MinMedium:  m.MinReplicas.Medium.ValueInt64(),
		MinHigh:    m.MinReplicas.High.ValueInt64(),
		MinExtreme: m.MinReplicas.Extreme.ValueInt64(),
	}
}

func ToEksHpaResourceModel(m *client.EksHpaResponse) eksHpaScalingResourceModel {
	return eksHpaScalingResourceModel{
		ServiceID: types.StringValue(m.ID),
		Cluster:   types.StringValue(m.Cluster),
		Region:    types.StringValue(m.Region),
		Namespace: types.StringValue(m.Namespace),
		Name:      types.StringValue(m.Name),
		Kind:      types.StringValue(m.Kind),
		MinReplicas: &eksHpaMinReplicasModel{
			Low:     types.Int64Value(m.MinLow),
			Medium:  types.Int64Value(m.MinMedium),
			High:    types.Int64Value(m.MinHigh),
			Extreme: types.Int64Value(m.MinExtreme),
		},
	}
}

func NewEksHpaScalingResource() resource.Resource {
	return &eksHpaScalingResource{}
}

type eksHpaScalingResource struct {
	client *client.SssClient
}

func (r *eksHpaScalingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *eksHpaScalingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_eks_hpa_scaling"
}

func (r *eksHpaScalingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages scheduled minReplicas for an EKS HorizontalPodAutoscaler or KEDA ScaledObject.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "The SSS scalable ID used as the URL path component. The provider convention is \"{namespace}/{name}@{cluster}\", but any unique string is accepted.",
				Required:    true,
			},
			"cluster": schema.StringAttribute{
				Description: "The EKS cluster name containing the target resource.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region of the EKS cluster. E.g. eu-west-1.",
				Required:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "The Kubernetes namespace of the HPA or ScaledObject.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the HorizontalPodAutoscaler or ScaledObject.",
				Required:    true,
			},
			"kind": schema.StringAttribute{
				Description: "The Kubernetes kind to scale. Must be \"HPA\" or \"ScaledObject\" — StatefulSet is deliberately unsupported.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"min_replicas": schema.SingleNestedAttribute{
				Description: "The minimum number of replicas to enforce at each schedule level.",
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

func (r *eksHpaScalingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eksHpaScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId, body := plan.ToClientModel()

	err := r.client.CreateEksHpa(serviceId, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create EKS HPA scaling", err.Error())
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eksHpaScalingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eksHpaScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.GetEksHpa(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read EKS HPA scaling", "Could not read scaling for "+state.ServiceID.ValueString()+": "+err.Error())
		return
	}

	newState := ToEksHpaResourceModel(response)

	if !state.LastUpdated.IsNull() {
		newState.LastUpdated = state.LastUpdated
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eksHpaScalingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan eksHpaScalingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId, body := plan.ToClientModel()
	err := r.client.UpdateEksHpa(serviceId, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update EKS HPA scaling", err.Error())
		return
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eksHpaScalingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eksHpaScalingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteEksHpa(state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete EKS HPA scaling", err.Error())
		return
	}
}

func (r *eksHpaScalingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}
