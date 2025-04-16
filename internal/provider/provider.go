// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"terraform-provider-sss/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &SssProvider{}
var _ provider.ProviderWithFunctions = &SssProvider{}
var _ provider.ProviderWithEphemeralResources = &SssProvider{}

// SssProvider defines the provider implementation.
type SssProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SssProviderModel describes the provider data model.
type SssProviderModel struct {
	Host         types.String `tfsdk:"host"`
	AuthUsername types.String `tfsdk:"auth_username"`
	AuthPassword types.String `tfsdk:"auth_password"`
	Protocol     types.String `tfsdk:"protocol"`
}

func (p *SssProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sss"
	resp.Version = p.version
}

func (p *SssProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the TV4 Media AB Scheduled Scaling Service.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The Scheduled Scaling Service API endpoint to connect to.",
				Required:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol to use when connecting to the Scheduled Scaling Service API.",
				Optional:            true,
			},
			"auth_username": schema.StringAttribute{
				MarkdownDescription: "The basicauth username to authenticate with.",
				Required:            true,
			},
			"auth_password": schema.StringAttribute{
				MarkdownDescription: "The basicauth password to authenticate with.",
				Sensitive:           true,
				Required:            true,
			},
		},
	}
}

func (p *SssProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SssProviderModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	client := client.NewSssClient(
		data.Host.ValueString(),
		data.Protocol.ValueString(), data.AuthUsername.ValueString(), data.AuthPassword.ValueString(),
	)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *SssProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEcsScalingResource,
		NewDynamoTableScalingResource,
	}
}

func (p *SssProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *SssProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *SssProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SssProvider{
			version: version,
		}
	}
}
