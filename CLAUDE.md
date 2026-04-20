# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

A standalone Go service that acts as a **GraphQL aggregation layer** between consumers and our existing backends. It fans out to multiple upstream REST services and merges results into a single GraphQL response.

## Commands

Use `make` as the primary interface (run `make help` to see all targets):

```bash
# Development
make run            # Run locally (reads .env)
make dev            # Live reload via air (install: go install github.com/air-verse/air@latest)
make generate       # Regenerate gqlgen bindings — MUST run after any schema.graphqls change
make build          # Build binary (outputs ./data-hub)

# Testing & Quality
make test           # Run all tests
make test-verbose   # Run tests with verbose output
make test-single TEST=TestFoo PKG=./tests/   # Run a single test
make vet            # go vet
make lint           # golangci-lint (install: brew install golangci-lint)

# Docker
make docker-build   # Build Docker image
make docker-run     # Run container (uses .env)
make up             # Start with docker-compose (API only)
make up-cache       # Start with docker-compose + Redis
make down           # Stop docker-compose services
make stop           # Stop and remove standalone container
make logs           # Tail container logs
make shell          # Open shell in running container

# Monitoring
make health         # curl /health on localhost:8080
make status         # Show container and pod status
make info           # Show environment info

# Cleanup
make clean          # Remove binary, containers, and images
```

> **Important**: Any change to `graph/schema.graphqls` requires `make generate` (runs `go run github.com/99designs/gqlgen generate`) before the server will compile correctly. Skipping this causes "internal system error" at runtime.

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
5. Run `make generate`

### Adding a new upstream endpoint
1. Add the path constant in `config/config.go` inside the relevant `Endpoints` struct
2. Add the method to the relevant client in `client/`
3. Add raw response structs to `model/upstream/admin.go`
4. Add graph model fields to `model/graph/loan.go`
5. Add a resolver in `graph/resolvers/`
6. Wire it in `graph/schema.graphqls` and regenerate

## Environment

Copy `.env.example` to `.env`. Current `.env.example` variables:

| Variable | Required | Notes |
|---|---|---|
| `PORT` | no | Defaults to `8080` |
| `X_API_KEY` | yes | Validated against incoming `X-Api-Key` header |

As the service is built out, these will be added:

| Variable | Notes |
|---|---|
| `ADMIN_BASE_URL` / `ADMIN_API_KEY` | Admin service |
| `CUSTOMER_BASE_URL` / `CUSTOMER_API_KEY` | Customer service |
| `RAZORPAY_BASE_URL` / `RAZORPAY_API_KEY` | Razorpay wrapper service |
| `CACHE_ENABLED` | Set `true` to enable Redis caching |
| `REDIS_ADDR` | e.g. `localhost:6379` (required if cache enabled) |

In Kubernetes, all secrets are pulled from AWS Secrets Manager secret `data-hub-secret` via ExternalSecrets — no `.env` file is used in deployed environments.

## Upstream API notes

- All upstream services expect `gq-api-key` header (not `X-API-Key`)
- Admin disbursement endpoint uses `application_id` as query param (not `app_id`)
- Repayment endpoint is a POST with `{"app_id": "..."}` body
- Upstream returns HTML error pages (not JSON) on auth failure — if you see `invalid character '<'` errors, the API key or base URL is wrong

## Jenkins CI

The `Jenkinsfile` defines a pipeline triggered against release branches (`release_uat`, `release_staging`, `release_prod`). Key notes:
- Jenkins agent label: `gq_arm_` (ARM build node)
- ECR repo name in Jenkins uses underscores: `data_hub_${TARGET_ENVIRONMENT}` — differs from the Makefile default which uses hyphens (`data-hub-${ENVIRONMENT}`)
- Image tag format: `${BRANCH_NAME}-${COMMIT_ID}` (e.g. `release_uat-abc1234`)
- `DEPLOYMENT_MODE=rollback` skips build/push stages and runs `kubectl rollout undo`

## Docker & Kubernetes

**Docker build note**: The Dockerfile builds for `GOARCH=arm64` and outputs the binary as `api_main` (not `data-hub`). Change `GOARCH` if deploying to x86 hosts.

**Deployment pipeline** (AWS ECR → EKS, `ap-south-1`, account `579897422692`):
```bash
make deploy-pipeline ENVIRONMENT=uat IMAGE_TAG=<tag>   # build → push → deploy
make eks-wait                                           # wait for rollout
make eks-rollback                                       # undo if needed
```
`ENVIRONMENT` maps to EKS cluster: `uat`/`staging` → `sandbox-tools`, `preprod` → `pre-prod-cluster`, `prod` → `production-cluster`.

**Helm chart** (`helm/`): Manages namespace, deployment, service, ingress (AWS ALB), HPA (2–10 replicas), and ExternalSecrets (pulls from AWS Secrets Manager secret `data-hub-secret` via `ClusterSecretStore`). Key values to override per environment: `image.repository`, `image.tag`, `ingress.hosts`, `ingress.annotations.alb.ingress.kubernetes.io/certificate-arn`.