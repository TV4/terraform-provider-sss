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
  service_id = "coreecs-general-cluster-fargate-main-ew1/corecwbatcher-general-app"
  min_tasks = {
    low     = 3
    medium  = 4
    high    = 5
    extreme = 6
  }
}
