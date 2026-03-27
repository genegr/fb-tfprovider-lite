terraform {
  required_version = ">= 1.14.0"

  required_providers {
    purefb = {
      source  = "purestorage/purefb"
      version = "~> 0.1"
    }
  }
}

provider "purefb" {
  fb_url    = var.fb_url
  api_token = var.api_token
}
