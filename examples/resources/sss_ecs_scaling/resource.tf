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
