resource "sss_valkey_shard_scaling" "example" {
  service_id                 = "example-valkey-replication-group"
  region                     = "eu-west-1"
  scale_up_lead_time_minutes = 15

  min_shard_count = {
    low     = 1
    medium  = 2
    high    = 3
    extreme = 4
  }
}
