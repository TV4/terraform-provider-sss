resource "sss_eks_hpa_scaling" "alloy_metrics" {
  service_id = "alloy/alloy-metrics@coreeks-main"
  cluster    = "coreeks-main"
  region     = "eu-west-1"
  namespace  = "alloy"
  name       = "alloy-metrics"
  kind       = "HPA"
  min_replicas = {
    low     = 4
    medium  = 6
    high    = 10
    extreme = 15
  }
}

resource "sss_eks_hpa_scaling" "tempo_distributor" {
  service_id = "tempo/tempo-distributor@coreeks-main"
  cluster    = "coreeks-main"
  region     = "eu-west-1"
  namespace  = "tempo"
  name       = "tempo-distributor"
  kind       = "ScaledObject"
  min_replicas = {
    low     = 2
    medium  = 3
    high    = 6
    extreme = 10
  }
}
