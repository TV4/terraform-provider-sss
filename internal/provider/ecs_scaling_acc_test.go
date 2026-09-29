// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"terraform-provider-sss/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/config"
	helperresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const ecsAcceptanceResourceName = "sss_ecs_scaling.test"

func TestAccEcsScalingPacedScaleUp(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" || os.Getenv("SSS_ECS_ACCEPTANCE") != "1" {
		t.Skip("requires both TF_ACC=1 and SSS_ECS_ACCEPTANCE=1; live ECS fixture access is opt-in")
	}

	endpoint := requiredEcsAcceptanceEnv(t, "SSS_ECS_ENDPOINT")
	username := requiredEcsAcceptanceEnv(t, "SSS_ECS_AUTH_USERNAME")
	password := requiredEcsAcceptanceEnv(t, "SSS_ECS_AUTH_PASSWORD")
	region := requiredEcsAcceptanceEnv(t, "SSS_ECS_REGION")
	serviceID := requiredEcsAcceptanceEnv(t, "SSS_ECS_SERVICE_ID")
	protocol := os.Getenv("SSS_ECS_PROTOCOL")
	if protocol == "" {
		protocol = "https"
	}
	if strings.Contains(endpoint, "://") {
		t.Fatal("SSS_ECS_ENDPOINT must be a host name without a scheme; set SSS_ECS_PROTOCOL separately")
	}

	apiClient := client.NewSssClient(endpoint, protocol, username, password)
	variables := config.Variables{
		"sss_endpoint": config.StringVariable(endpoint), "sss_protocol": config.StringVariable(protocol),
		"sss_auth_username": config.StringVariable(username), "sss_auth_password": config.StringVariable(password),
		"sss_region": config.StringVariable(region), "sss_service_id": config.StringVariable(serviceID),
	}
	configVariables := func() config.Variables { return variables }
	providerFactory := providerserver.NewProtocol6(New("test")())
	providerFactories := map[string]func() (tfprotov6.ProviderServer, error){
		"sss": func() (tfprotov6.ProviderServer, error) { return providerFactory(), nil },
	}

	legacyConfig := ecsAcceptanceConfig([4]int{4, 10, 18, 18}, "")
	exampleConfig := ecsAcceptanceConfig([4]int{4, 10, 18, 18}, "scale_up_tasks_per_minute = 2\n  scale_up_lead_time_minutes = 7")
	rateRemovedConfig := ecsAcceptanceConfig([4]int{4, 10, 18, 18}, "scale_up_lead_time_minutes = 7")
	leadRemovedConfig := ecsAcceptanceConfig([4]int{4, 10, 18, 18}, "scale_up_tasks_per_minute = 2")
	invalidSpreadConfig := ecsAcceptanceConfig([4]int{4, 10, 18, 125}, "scale_up_tasks_per_minute = 1")

	expectedUnchanged := client.EcsServicePostBody{
		Region: region, MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18,
	}
	checkUnchanged := func() {
		response, err := apiClient.GetEcsService(serviceID)
		if err != nil {
			t.Fatalf("verify rejected update left API configuration readable: %v", err)
		}
		if response.Region != expectedUnchanged.Region || response.MinLowCapacity != expectedUnchanged.MinLowCapacity ||
			response.MinMediumCapacity != expectedUnchanged.MinMediumCapacity || response.MinHighCapacity != expectedUnchanged.MinHighCapacity ||
			response.MinExtremeCapacity != expectedUnchanged.MinExtremeCapacity || response.ScaleUpTasksPerMinute != 0 || response.ScaleUpLeadTimeMinutes != 0 {
			t.Fatalf("rejected update changed stored API configuration: %#v", response)
		}
	}

	helperresource.Test(t, helperresource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []helperresource.TestStep{
			{
				Config:          legacyConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "0"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "0"),
				),
			},
			{Config: legacyConfig, ConfigVariables: configVariables(), PlanOnly: true},
			{
				Config:          exampleConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "2"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "7"),
					ecsAcceptanceCheckAPIValues(apiClient, serviceID, 2, 7),
				),
			},
			{Config: exampleConfig, ConfigVariables: configVariables(), PlanOnly: true},
			{
				Config:                               exampleConfig,
				ConfigVariables:                      configVariables(),
				ResourceName:                         ecsAcceptanceResourceName,
				ImportState:                          true,
				ImportStateId:                        serviceID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "service_id",
				ImportStateVerifyIgnore:              []string{"last_updated"},
			},
			{
				Config:          rateRemovedConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "0"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "7"),
					ecsAcceptanceCheckAPIValues(apiClient, serviceID, 0, 7),
				),
			},
			{
				Config:          leadRemovedConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "2"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "0"),
					ecsAcceptanceCheckAPIValues(apiClient, serviceID, 2, 0),
				),
			},
			{
				Config:          legacyConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "0"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "0"),
				),
			},
			{
				ConfigVariables: configVariables(),
				PreConfig: func() {
					if err := apiClient.UpdateEcsService(serviceID, client.EcsServicePostBody{
						Region: region, MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18,
						ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
					}); err != nil {
						t.Fatalf("set API drift for refresh test: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "2"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "7"),
				),
			},
			{
				Config:          legacyConfig,
				ConfigVariables: configVariables(),
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "0"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "0"),
				),
			},
			{
				Config:          invalidSpreadConfig,
				ConfigVariables: configVariables(),
				ExpectError:     regexp.MustCompile(`422 Unprocessable Entity`),
			},
			{
				Config:          legacyConfig,
				ConfigVariables: configVariables(),
				PreConfig:       checkUnchanged,
				Check: helperresource.ComposeTestCheckFunc(
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_tasks_per_minute", "0"),
					helperresource.TestCheckResourceAttr(ecsAcceptanceResourceName, "scale_up_lead_time_minutes", "0"),
				),
			},
		},
		CheckDestroy: func(_ *terraform.State) error {
			_, err := apiClient.GetEcsService(serviceID)
			if err == nil {
				return fmt.Errorf("test-owned ECS scaling configuration %q still exists after destroy", serviceID)
			}
			if !strings.Contains(err.Error(), "404 Not Found") {
				return fmt.Errorf("checking test-owned ECS scaling configuration after destroy: %w", err)
			}
			return nil
		},
	})
}

func ecsAcceptanceCheckAPIValues(apiClient *client.SssClient, serviceID string, rate, lead int64) helperresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		response, err := apiClient.GetEcsService(serviceID)
		if err != nil {
			return fmt.Errorf("read ECS scaling configuration for API/state comparison: %w", err)
		}
		if response.ScaleUpTasksPerMinute != rate || response.ScaleUpLeadTimeMinutes != lead {
			return fmt.Errorf("API scale-up values = %d/%d, want %d/%d", response.ScaleUpTasksPerMinute, response.ScaleUpLeadTimeMinutes, rate, lead)
		}
		return nil
	}
}

func requiredEcsAcceptanceEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("live ECS acceptance requires %s to be set", name)
	}
	return value
}

func ecsAcceptanceConfig(capacities [4]int, settings string) string {
	return fmt.Sprintf(`variable "sss_endpoint" {}
variable "sss_protocol" {}
variable "sss_auth_username" {}
variable "sss_auth_password" { sensitive = true }
variable "sss_region" {}
variable "sss_service_id" {}

provider "sss" {
  host          = var.sss_endpoint
  protocol      = var.sss_protocol
  auth_username = var.sss_auth_username
  auth_password = var.sss_auth_password
}

resource "sss_ecs_scaling" "test" {
  service_id = var.sss_service_id
  region     = var.sss_region
  min_tasks = {
    low     = %d
    medium  = %d
    high    = %d
    extreme = %d
  }
%s
}
`, capacities[0], capacities[1], capacities[2], capacities[3], settings)
}
