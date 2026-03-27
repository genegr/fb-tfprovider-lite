# Project: Terraform Provider for Pure Storage FlashBlade

A native Terraform provider written in Go for managing FlashBlade S3 resources (object store accounts and buckets) via the FlashBlade REST API 2.12.

## Architecture

```
terraform plan/apply/destroy/import
        │
        ▼
terraform-provider-purefb  (Go binary, Plugin Framework)
        │
        ▼
FlashBlade REST API 2.12  (fb_url + api_token)
```

The provider authenticates via `POST /api/2.12/login` with the `api-token` header, receives an `x-auth-token` session token, and uses it for all subsequent requests. On teardown it calls `/api/2.12/logout`.

### Key design decisions

- **Handwritten REST client** (`internal/fbclient/`) — only 8 endpoints needed (CRUD for accounts and buckets). No generated SDK.
- **Terraform Plugin Framework** (not SDKv2) — the modern recommended approach.
- **Human-readable quota strings** ("100G", "1T") — parsed to bytes by `HumanToBytes()` before sending to the API.
- **Two-step bucket delete** — PATCH `{destroyed: true}` (soft-delete), then optionally DELETE (eradicate). Controlled by `eradicate_on_destroy` attribute.
- **Two-step account create** — POST creates the account (name only), PATCH sets quota/hard_limit.

## Project Structure

```
terraform-provider-purefb/
├── main.go                          # Plugin server entry point
├── go.mod / go.sum
├── Makefile                         # build, install, test, clean, fmt
│
├── internal/
│   ├── provider/
│   │   └── provider.go              # Provider schema (fb_url, api_token, verify_ssl) + Configure
│   │
│   ├── fbclient/
│   │   ├── client.go                # HTTP client, login/logout, doRequest, HumanToBytes/BytesToHuman
│   │   ├── client_test.go           # Unit tests for HumanToBytes/BytesToHuman
│   │   ├── object_store_accounts.go # Get/Create/Update/Delete ObjectStoreAccount
│   │   └── buckets.go               # Get/Create/Update/Delete Bucket
│   │
│   └── resources/
│       ├── s3_account_resource.go   # purefb_s3_account: CRUD + Import + schema
│       └── bucket_resource.go       # purefb_bucket: CRUD + Import + schema
│
└── examples/
    ├── providers.tf
    ├── variables.tf
    ├── main.tf
    └── terraform.tfvars.example
```

## Environment

- **Go**: >= 1.22 (used 1.23.8 during initial development; `go mod tidy` may download a newer toolchain)
- **Terraform**: >= 1.14
- **Target API**: FlashBlade REST API 2.12 (constant `apiVersion` in `client.go`)

## Build and Test

```bash
make build     # compile the binary
make install   # build + install to ~/.terraform.d/plugins/
make test      # run unit tests
make testacc   # run acceptance tests (requires PUREFB_URL + PUREFB_API env vars)
make fmt       # gofmt
make clean     # remove binary
```

The provider binary installs to `~/.terraform.d/plugins/registry.terraform.io/purestorage/purefb/0.1.0/linux_amd64/`.

## FlashBlade REST API Endpoints Used

| Resource | Operation | Method | Path |
|----------|-----------|--------|------|
| Account | Create | POST | `/api/2.12/object-store-accounts?names=<name>` |
| Account | Read | GET | `/api/2.12/object-store-accounts?names=<name>` |
| Account | Update | PATCH | `/api/2.12/object-store-accounts?names=<name>` |
| Account | Delete | DELETE | `/api/2.12/object-store-accounts?names=<name>` |
| Bucket | Create | POST | `/api/2.12/buckets?names=<name>` |
| Bucket | Read | GET | `/api/2.12/buckets?names=<name>` |
| Bucket | Update | PATCH | `/api/2.12/buckets?names=<name>` |
| Bucket | Soft-delete | PATCH | `/api/2.12/buckets?names=<name>` body: `{destroyed: true}` |
| Bucket | Eradicate | DELETE | `/api/2.12/buckets?names=<name>` |

## Code Style

- Standard Go formatting (`gofmt -s`)
- Terraform HCL formatting for `.tf` files (`terraform fmt`)
- Resource files follow the pattern: Schema, Configure, Create, Read, Update, Delete, ImportState, readIntoState

## Sensitive Variables

`api_token` is `Sensitive: true` in the provider schema. Never commit real credentials. Use `terraform.tfvars` (gitignored) or `PUREFB_API` environment variable.
