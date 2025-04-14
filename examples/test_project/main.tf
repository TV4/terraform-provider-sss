terraform {
  required_providers {
    sss = {
      source = "tv4/sss"
    }
  }
}

variable "sss_auth_password" {
  type      = string
  sensitive = true
}

variable "sss_host" {
  type = string
}

variable "sss_auth_username" {
  type = string
}

variable "sss_protocol" {
  type    = string
  default = "https"
}

provider "sss" {
  host          = var.sss_host
  protocol      = var.sss_protocol
  auth_username = var.sss_auth_username
  auth_password = var.sss_auth_password
}

resource "sss_ecs_scaling" "test" {
  service_id = "service/coreecs-general-cluster-fargate-main-en1/tv4testing-general-myservice"
  region     = "eu-north-1"
  min_tasks = {
    low     = 3
    medium  = 4
    high    = 5
    extreme = 6
  }
}

resource "sss_dynamo_table_scaling" "table_test_entry" {
  table_name = "table/a2dcmsapi-mtvsync-entrytable"
  region     = "eu-west-1"
  capacity   = {
    low      = {
      min_write = 1
      max_write = 1
      min_read  = 1
      max_read  = 1
    }

    medium      = {
      min_write = 1
      max_write = 1
      min_read  = 1
      max_read  = 1
    }
    high    = {
      min_write = 1
      max_write = 1
      min_read  = 1
      max_read  = 1
    }
    extreme = {
      min_write = 1
      max_write = 2
      min_read  = 1
      max_read  = 1
    }
  }
}
