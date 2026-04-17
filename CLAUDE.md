# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

A standalone Go service that acts as a **GraphQL aggregation layer** between consumers and our existing backends. It fans out to multiple upstream REST services and merges results into a single GraphQL response.

## Commands

```bash
# Run the service (requires .env to be populated)
go run ./cmd/main.go

# Regenerate gqlgen bindings — MUST run after any schema.graphqls change
go run github.com/99designs/gqlgen generate

# Run tests
go test ./...

# Run a single test file
go test ./tests/client_test.go

# Build binary
go build -o data-hub ./cmd/main.go
```

> **Important**: Any change to `graph/schema.graphqls` requires running `go generate ./...` (or the gqlgen command above) before the server will compile correctly. Skipping this causes "internal system error" at runtime.

## Architecture

### Request flow
```
Consumer → Gin router → APIKeyAuth middleware → GraphQL handler (gqlgen)
  → schema.resolvers.go (thin, generated)
  → graph/resolvers/ (business logic)
  → client/ (HTTP calls to upstream)
  → upstream REST APIs
```

### Key design rules
- `cmd/main.go` — wiring only (config, deps, routes). No business logic.
- `graph/resolvers/` — one file per resolver. Each follows: cache check → upstream call → map to graph model → cache store.
- `model/upstream/` — raw JSON structs matching upstream API responses exactly. Never returned to GraphQL consumers.
- `model/graph/loan.go` — GraphQL model structs. All fields are pointers so absent data serialises as `null`.
- `client/` — one client per upstream service (`AdminClient`, `CustomerClient`, `RazorpayClient`). All URLs and keys come from `config.Config`, never hardcoded.
- `config/config.go` — single source of truth for all config. Base URLs from `.env`; endpoint paths hardcoded here.

### Adding a new field to an existing type
1. Add the field to `graph/schema.graphqls`
2. Add the JSON field to the relevant struct in `model/upstream/admin.go`
3. Add the field to the graph model struct in `model/graph/loan.go`
4. Map the field in the resolver in `graph/resolvers/`
5. Run `go run github.com/99designs/gqlgen generate`

### Adding a new upstream endpoint
1. Add the path constant in `config/config.go` inside the relevant `Endpoints` struct
2. Add the method to the relevant client in `client/`
3. Add raw response structs to `model/upstream/admin.go`
4. Add graph model fields to `model/graph/loan.go`
5. Add a resolver in `graph/resolvers/`
6. Wire it in `graph/schema.graphqls` and regenerate

## Environment

Copy `.env.example` to `.env`. Required variables:

| Variable | Required | Notes |
|---|---|---|
| `GATEWAY_API_KEY` | yes | Incoming `X-Api-Key` header value |
| `ADMIN_BASE_URL` | yes | e.g. `https://admin-api.uat.graydev.in` |
| `ADMIN_API_KEY` | yes | Sent as `gq-api-key` header to admin service |
| `CUSTOMER_BASE_URL` / `CUSTOMER_API_KEY` | yes | Customer service |
| `RAZORPAY_BASE_URL` / `RAZORPAY_API_KEY` | yes | Razorpay wrapper service |
| `CACHE_ENABLED` | no | Set `true` to enable Redis caching |
| `REDIS_ADDR` | if cache enabled | e.g. `localhost:6379` |

## Upstream API notes

- All upstream services expect `gq-api-key` header (not `X-API-Key`)
- Admin disbursement endpoint uses `application_id` as query param (not `app_id`)
- Repayment endpoint is a POST with `{"app_id": "..."}` body
- Upstream returns HTML error pages (not JSON) on auth failure — if you see `invalid character '<'` errors, the API key or base URL is wrong