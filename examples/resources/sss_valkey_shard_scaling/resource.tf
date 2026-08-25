resource "sss_valkey_shard_scaling" "example" {
  service_id                 = "example-valkey-replication-group"
  region                     = "eu-west-1"
  scale_up_lead_time_minutes = 15

  capacity = {
    low = {
      min_shard_count = 1
      max_shard_count = 2
    }
    medium = {
      min_shard_count = 2
      max_shard_count = 3
    }
    high = {
      min_shard_count = 3
      max_shard_count = 4
    }
    extreme = {
      min_shard_count = 4
      max_shard_count = 5
    }
  }
}
