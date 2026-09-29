resource "sss_ecs_scaling" "example" {
  service_id = "service/example-cluster/example-service"
  region     = "eu-west-1"

  min_tasks = {
    low     = 4
    medium  = 10
    high    = 18
    extreme = 18
  }

  scale_up_tasks_per_minute  = 2
  scale_up_lead_time_minutes = 7
}
