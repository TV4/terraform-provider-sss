resource "sss_valkey_replica_scaling" "example" {
  service_id                 = "example-valkey-replication-group"
  region                     = "eu-west-1"
  scale_up_lead_time_minutes = 15

  replica_count = {
    low     = 0
    medium  = 1
    high    = 2
    extreme = 3
  }
}
