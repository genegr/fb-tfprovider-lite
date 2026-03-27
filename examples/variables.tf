variable "fb_url" {
  description = "FlashBlade management IP address or hostname."
  type        = string
}

variable "api_token" {
  description = "FlashBlade API token."
  type        = string
  sensitive   = true
}

variable "s3_accounts" {
  description = "Map of S3 object store account names to their configuration."
  type = map(object({
    quota      = optional(string, "")
    hard_limit = optional(bool, false)
  }))
  default = {}
}

variable "buckets" {
  description = "Map of bucket names to their configuration."
  type = map(object({
    account_name = string
    versioning   = optional(string, "none")
    quota        = optional(string, "")
    hard_limit   = optional(bool, false)
    eradicate    = optional(bool, false)
  }))
  default = {}
}
