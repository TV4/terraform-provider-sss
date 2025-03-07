resource "sss_ecs_scaling" "test" {
  service_id = "coreecs-general-cluster-fargate-main-ew1/corecwbatcher-general-app"
  min_tasks = {
    low     = 3
    medium  = 4
    high    = 5
    extreme = 6
  }
}
