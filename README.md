# Terraform Provider for Pure Storage FlashBlade

A native Terraform provider for managing Pure Storage FlashBlade S3 resources (object store accounts and buckets) via the FlashBlade REST API.

---

## Features

- **Native CRUD** — creates, reads, updates, and deletes S3 accounts and buckets directly via the FlashBlade REST API (no Ansible or Python dependencies)
- **Real drift detection** — `terraform plan` reads actual FlashBlade state and shows what differs from your config
- **`terraform import`** — import existing FlashBlade resources into Terraform state
- **Soft-delete / eradicate** — buckets are soft-deleted by default (recoverable from trash); set `eradicate_on_destroy = true` to permanently remove

---

## Requirements

| Tool | Version |
|------|---------|
| Terraform | >= 1.14 |
| Go | >= 1.22 (build only) |
| FlashBlade | Purity//FB with REST API >= 2.12 |

---

## Building and Installing

```bash
# Build
make build

# Install to local Terraform plugin directory
make install

# Run unit tests
make test
```

The `make install` target places the binary at `~/.terraform.d/plugins/registry.terraform.io/purestorage/purefb/0.1.0/linux_amd64/`.

---

## Provider Configuration

```hcl
terraform {
  required_providers {
    purefb = {
      source  = "purestorage/purefb"
      version = "~> 0.1"
    }
  }
}

provider "purefb" {
  fb_url    = "10.225.112.185"       # or set PUREFB_URL env var
  api_token = "T-xxxxxxxx-xxxx-..."  # or set PUREFB_API env var
  # verify_ssl = false               # default: false (FlashBlade typically uses self-signed certs)
}
```

### Authentication

The provider authenticates via the FlashBlade API token. Credentials can be set in the provider block or via environment variables:

| Attribute | Env Var | Description |
|-----------|---------|-------------|
| `fb_url` | `PUREFB_URL` | FlashBlade management IP or hostname |
| `api_token` | `PUREFB_API` | API token (Settings > API Tokens on the FlashBlade UI) |
| `verify_ssl` | — | Whether to verify TLS certificates (default: `false`) |

---

## Resources

### `purefb_s3_account`

Manages a FlashBlade S3 object store account.

```hcl
resource "purefb_s3_account" "myteam" {
  name               = "myteam"
  quota              = "1T"
  hard_limit_enabled = true
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | String | Yes | Account name (immutable after creation) |
| `quota` | String | No | Quota in human-readable format ("100G", "1T"). Omit for unlimited |
| `hard_limit_enabled` | Bool | No | Enforce quota as hard limit (default: `false`) |
| `id` | String | Computed | FlashBlade internal ID |
| `quota_limit` | Int64 | Computed | Effective quota in bytes |
| `object_count` | Int64 | Computed | Number of objects |
| `created` | Int64 | Computed | Creation timestamp |

**Import:**
```bash
terraform import purefb_s3_account.myteam myteam
```

### `purefb_bucket`

Manages a FlashBlade S3 bucket.

```hcl
resource "purefb_bucket" "logs" {
  name                 = "logs-bucket"
  account_name         = "myteam"
  versioning           = "enabled"
  quota                = "200G"
  hard_limit_enabled   = false
  eradicate_on_destroy = false
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | String | Yes | Bucket name (immutable after creation) |
| `account_name` | String | Yes | Parent S3 account (immutable after creation) |
| `versioning` | String | No | `"none"`, `"enabled"`, or `"suspended"` (default: `"none"`) |
| `quota` | String | No | Quota in human-readable format. Omit for unlimited |
| `hard_limit_enabled` | Bool | No | Enforce quota as hard limit (default: `false`) |
| `eradicate_on_destroy` | Bool | No | Permanently eradicate on destroy (default: `false`) |
| `id` | String | Computed | FlashBlade internal ID |
| `quota_limit` | Int64 | Computed | Effective quota in bytes |
| `bucket_type` | String | Computed | `"classic"` or `"multi-site-writable"` |
| `object_count` | Int64 | Computed | Number of objects |
| `destroyed` | Bool | Computed | Whether bucket is in trash |
| `created` | Int64 | Computed | Creation timestamp |

**Import:**
```bash
terraform import purefb_bucket.logs logs-bucket
```

---

## Full Example

```hcl
provider "purefb" {
  fb_url    = "10.225.112.185"
  api_token = "T-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "purefb_s3_account" "myteam" {
  name = "myteam"
}

resource "purefb_bucket" "logs" {
  name         = "logs-bucket"
  account_name = purefb_s3_account.myteam.name
  versioning   = "enabled"
  quota        = "200G"
}

resource "purefb_bucket" "backups" {
  name                 = "backups-bucket"
  account_name         = purefb_s3_account.myteam.name
  quota                = "1T"
  hard_limit_enabled   = true
  eradicate_on_destroy = false
}
```

```bash
terraform init
terraform plan
terraform apply
```

---

## Importing Existing Resources

```bash
# Import an existing account
terraform import 'purefb_s3_account.myteam' myteam

# Import an existing bucket
terraform import 'purefb_bucket.logs' logs-bucket

# Verify no drift
terraform plan
```

---

## Sensitive Data

`api_token` is marked `sensitive` in the provider schema. Terraform redacts it in plan/apply output. Never commit real credentials — use `terraform.tfvars` (gitignored) or environment variables.
