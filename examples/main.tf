resource "purefb_s3_account" "account" {
  for_each = var.s3_accounts

  name               = each.key
  quota              = each.value.quota != "" ? each.value.quota : null
  hard_limit_enabled = each.value.hard_limit
}

resource "purefb_bucket" "bucket" {
  for_each = var.buckets

  name                 = each.key
  account_name         = each.value.account_name
  versioning           = each.value.versioning
  quota                = each.value.quota != "" ? each.value.quota : null
  hard_limit_enabled   = each.value.hard_limit
  eradicate_on_destroy = each.value.eradicate

  depends_on = [purefb_s3_account.account]
}
