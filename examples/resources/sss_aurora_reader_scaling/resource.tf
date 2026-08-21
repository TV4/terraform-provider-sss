resource "sss_aurora_reader_scaling" "example" {
  service_id                 = "example-aurora-cluster"
  region                     = "eu-west-1"
  scale_up_lead_time_minutes = 15

  capacity = {
    low = {
      min_readers = 1
      max_readers = 2
    }
    medium = {
      min_readers = 2
      max_readers = 3
    }
    high = {
      min_readers = 3
      max_readers = 4
    }
    extreme = {
      min_readers = 4
      max_readers = 5
    }
  }
}
