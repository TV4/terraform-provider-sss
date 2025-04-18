---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sss_ecs_scaling Resource - sss"
subcategory: ""
description: |-
  Manages scaling for ECS services.
---

# sss_ecs_scaling (Resource)

Manages scaling for ECS services.

## Example Usage

```terraform
resource "sss_ecs_scaling" "test" {
  service_id = "service/coreecs-general-cluster-fargate-main-ew1/corecwbatcher-general-app"
  region     = "eu-west-1"
  min_tasks = {
    low     = 3
    medium  = 4
    high    = 5
    extreme = 6
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `min_tasks` (Attributes) The minimum number of tasks to have during different schedules. (see [below for nested schema](#nestedatt--min_tasks))
- `region` (String) The AWS region the service is located in. E.g. eu-west-1
- `service_id` (String) The service ID. Should be in format CLUSTER_NAME/SERICE_NAME

### Read-Only

- `last_updated` (String)

<a id="nestedatt--min_tasks"></a>
### Nested Schema for `min_tasks`

Required:

- `extreme` (Number)
- `high` (Number)
- `low` (Number)
- `medium` (Number)

## Import

Import is supported using the following syntax:

```shell
# Scaling can be imported by specifying the service identifier.
tofu import sss_ecs_scaling.example cluster_name/service_name
```
