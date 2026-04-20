# data-hub — Build Prompt

---

## Project Overview

Build a Go project called `data-hub`. This is a GraphQL aggregation gateway that calls existing REST APIs from multiple backend services and returns clean, trimmed data to consumers like GQ Brain (AI customer support agent).

The gateway talks to multiple upstream services — each service has its own HTTP client. Adding a new service in future means adding one new client file and its base URL in config. Nothing else changes.

I am new to Go so keep the structure simple, well commented, and easy to follow. Start with ONE working API end to end — disbursement. Other resolvers can return nil for now. Once disbursement works, the same pattern repeats for everything else.

---

## Tech Stack

| Component | Choice |
|---|---|
| Language | Go 1.22+ |
| HTTP Server | Gin |
| GraphQL | gqlgen |
| HTTP Client | resty |
| Cache | Redis (go-redis) |
| Config | viper |
| Logging | zap |

---

## Project Structure

```
data-hub/
├── cmd/
│   └── main.go                      # entry point, starts server, graceful shutdown
├── config/
│   └── config.go                    # ALL service base URLs and endpoints here
├── graph/
│   ├── schema.graphqls              # GraphQL schema definition
│   ├── resolver.go                  # root resolver
│   └── resolvers/
│       ├── loan_resolver.go         # loan data resolver (disbursement first)
│       ├── repayment_resolver.go    # repayment resolver (return nil for now)
│       ├── mandate_resolver.go      # mandate resolver (return nil for now)
│       └── payments_resolver.go     # payments resolver (return nil for now)
├── client/
│   ├── base_client.go               # shared resty setup, timeout, retry, headers
│   ├── admin_client.go              # calls to gqadmin_backend
│   ├── customer_client.go           # calls to gqcustomer_backend
│   └── razorpay_client.go           # calls to Razorpay service
├── cache/
│   └── redis.go                     # Redis get/set helpers
├── middleware/
│   ├── auth.go                      # X-API-Key validation
│   └── request_id.go                # adds unique request ID to every request
├── model/
│   ├── upstream/
│   │   └── admin.go                 # raw response structs from upstream APIs
│   └── graph/
│       └── loan.go                  # GraphQL model structs
├── tests/
│   ├── resolver_test.go             # resolver unit tests
│   └── client_test.go               # client unit tests
├── gqlgen.yml                       # gqlgen config
├── go.mod
├── go.sum
├── .env                             # actual env values — never commit this
├── .env.example                     # template showing all required env keys
└── .gitignore                       # must include .env
```

---

## Environment Variables (.env)

```env
PORT=8080
GATEWAY_API_KEY=gqbrain-secret-key

ADMIN_BASE_URL=http://gqadmin_backend
ADMIN_API_KEY=your-admin-key

CUSTOMER_BASE_URL=http://gqcustomer_backend
CUSTOMER_API_KEY=your-customer-key

RAZORPAY_BASE_URL=http://razorpay-service
RAZORPAY_API_KEY=your-razorpay-key

REDIS_ADDR=localhost:6379
CACHE_ENABLED=true
```

Add a new service in future = add its BASE_URL and API_KEY here. Nothing else changes.

Also create `.env.example` with the same keys but empty values:
```env
PORT=
GATEWAY_API_KEY=

ADMIN_BASE_URL=
ADMIN_API_KEY=

CUSTOMER_BASE_URL=
CUSTOMER_API_KEY=

RAZORPAY_BASE_URL=
RAZORPAY_API_KEY=

REDIS_ADDR=
CACHE_ENABLED=
```

Create `.gitignore` with at minimum:
```
.env
*.log
```

---

## Config (config/config.go)

All service base URLs and endpoints must be defined in ONE place here.
No URLs anywhere else in the project — always read from config.
Each service has its own section in the config struct.

```go
type Config struct {
    Port          string
    GatewayAPIKey string

    // Redis
    RedisAddr    string
    CacheEnabled bool

    // gqadmin_backend — base URL + API key + its endpoints
    Admin struct {
        BaseURL string
        APIKey  string
        Endpoints struct {
            Disbursement  string  // /v0.1/disbursal-requests/tranch-details/fetch
            Repayment     string  // /v1/service-wrapper/action/fetch-repayment-schedule
            Mandate       string  // /v0.1/mandates/fetch
            Payments      string  // /v1/payments/transactions/fetch
            Tracker       string  // /v1/trackers/customer/{app_id}/fetch
            Summary       string  // /v1/applications/summary/{app_id}/fetch
            KYC           string  // /v1/service-wrapper/action/kyc-fetch-status
            Refund        string  // /v1/refunds/requests/fetch
            SOA           string  // /v1/service-wrapper/action/fetch-soa
            NDC           string  // /v1/service-wrapper/action/fetch-ndc
            OverdueList   string  // /v1/service-wrapper/action/fetch-over-due-list
        }
    }

    // gqcustomer_backend — base URL + API key + its endpoints
    Customer struct {
        BaseURL string
        APIKey  string
        Endpoints struct {
            Profile  string  // add customer endpoints here
        }
    }

    // Razorpay service — base URL + API key + its endpoints
    Razorpay struct {
        BaseURL string
        APIKey  string
        Endpoints struct {
            PaymentDetails string  // /v1/razorpay/payment_details/{app_id}/fetch
            CreateLink     string  // /v1/razorpay/payment_link/create
        }
    }
}

// Load reads all config from .env using viper
func Load() (*Config, error)
```

Adding a new service = add a new struct section here + its env vars in .env. No other files change.

---

## GraphQL Schema (graph/schema.graphqls)

```graphql
type Query {
  loanData(applicationId: String!): LoanData
}

type LoanData {
  disbursement: DisbursementInfo
  repayment:    RepaymentInfo
  mandate:      MandateInfo
  payments:     [PaymentEntry]
}

type DisbursementInfo {
  utr:    String
  date:   String
  amount: Float
  status: String
  mode:   String
}

type RepaymentInfo {
  totalEmis:            Int
  emisPaid:             Int
  nextEmiDate:          String
  nextEmiAmount:        Float
  outstandingPrincipal: Float
  overdueAmount:        Float
  dpd:                  Int
}

type MandateInfo {
  status:       String
  type:         String
  bank:         String
  accountLast4: String
  startDate:    String
  endDate:      String
}

type PaymentEntry {
  date:         String
  amount:       Float
  utr:          String
  status:       String
  mode:         String
  bounceReason: String
}
```

---

## Resolver Behaviour (graph/resolvers/)

### Root Resolver (resolver.go)
- Receives `applicationId`
- Returns `LoanData` struct with `applicationId` stored inside
- Makes ZERO API calls
- Just passes `applicationId` down to child resolvers

### Child Resolvers — one per section
Each child resolver is called ONLY if that section was requested in the query.

Every child resolver must follow this exact pattern:

```
1. Check if CACHE_ENABLED is true
   YES → check Redis: key = gqdg:{appId}:{section}
         HIT  → return cached data immediately
         MISS → proceed to step 2
   NO  → proceed to step 2 directly

2. Call upstream API via the correct service client
   (admin_client, customer_client, or razorpay_client
    — whichever service owns that data)

3. Map response to GraphQL model struct

4. Check if CACHE_ENABLED is true
   YES → store result in Redis with TTL for this section
   NO  → skip

5. Return struct — gqlgen trims to requested fields automatically
```

### Cache TTL Per Section

| Section | TTL |
|---|---|
| disbursement | 2 hours |
| repayment | 30 minutes |
| mandate | 15 minutes |
| payments | 5 minutes |
| kyc | 1 hour |
| closure | 10 minutes |
| application | 5 minutes |
| refund | 15 minutes |
| documents | 1 hour |
| overdueLink | 5 minutes |

---

## HTTP Clients (client/)

One file per upstream service. All share a common base setup.
All URLs and API keys come from config — never hardcoded.

### client/base_client.go
Shared resty setup used by all clients:
- Default timeout: 10 seconds
- JSON content type header
- Retry once on network error or 5xx response
- Log every request: service name, endpoint, appId, duration, status code

```go
func NewBaseHTTPClient() *resty.Client
```

### client/admin_client.go — calls to gqadmin_backend

```go
type AdminClient struct {
    cfg        *config.Config
    httpClient *resty.Client
}

func NewAdminClient(cfg *config.Config) *AdminClient

// Implement this first
func (c *AdminClient) FetchDisbursement(ctx context.Context, appID string) (*DisbursementResponse, error)

// Add after disbursement works
func (c *AdminClient) FetchRepaymentSchedule(ctx context.Context, appID string) (*RepaymentResponse, error)
func (c *AdminClient) FetchMandate(ctx context.Context, appID string) (*MandateResponse, error)
func (c *AdminClient) FetchPayments(ctx context.Context, appID string) (*PaymentsResponse, error)
func (c *AdminClient) FetchTracker(ctx context.Context, appID string) (*TrackerResponse, error)
func (c *AdminClient) FetchKYC(ctx context.Context, appID string) (*KYCResponse, error)
func (c *AdminClient) FetchRefund(ctx context.Context, appID string) (*RefundResponse, error)
func (c *AdminClient) FetchSOA(ctx context.Context, appID string) (*SOAResponse, error)
func (c *AdminClient) FetchNDC(ctx context.Context, appID string) (*NDCResponse, error)
```

### client/customer_client.go — calls to gqcustomer_backend

```go
type CustomerClient struct {
    cfg        *config.Config
    httpClient *resty.Client
}

func NewCustomerClient(cfg *config.Config) *CustomerClient

// Add customer service calls here
func (c *CustomerClient) FetchProfile(ctx context.Context, appID string) (*ProfileResponse, error)
```

### client/razorpay_client.go — calls to Razorpay service

```go
type RazorpayClient struct {
    cfg        *config.Config
    httpClient *resty.Client
}

func NewRazorpayClient(cfg *config.Config) *RazorpayClient

func (c *RazorpayClient) FetchPaymentDetails(ctx context.Context, appID string) (*PaymentLinkResponse, error)
func (c *RazorpayClient) CreatePaymentLink(ctx context.Context, appID string) (*PaymentLinkResponse, error)
```

Every function in every client must:
- Read base URL and API key from its section in `cfg`
- Set `X-API-Key` header
- Handle non-200 responses with clear error message
- Return parsed struct and error

---

## Cache (cache/redis.go)

```go
type Cache struct {
    client  *redis.Client
    enabled bool   // read from config.CacheEnabled
}

func NewCache(cfg *config.Config) *Cache

// Get returns empty string and nil error if caching is disabled
func (c *Cache) Get(key string) (string, error)

// Set does nothing and returns nil if caching is disabled
func (c *Cache) Set(key string, value string, ttl time.Duration) error
```

Key pattern: `gqdg:{appId}:{section}`

Examples:
```
gqdg:APP123:disbursement
gqdg:APP123:repayment
gqdg:APP123:mandate
gqdg:APP123:payments
```

When `CACHE_ENABLED=false`:
- Do not connect to Redis at all
- `Get` always returns empty string
- `Set` does nothing
- No errors related to Redis ever surface

---

## Auth Middleware (middleware/auth.go)

```go
func APIKeyAuth(cfg *config.Config) gin.HandlerFunc
```

- Read `X-API-Key` header from every request
- Compare against `cfg.GatewayAPIKey`
- Return `403` with JSON error if missing or invalid
- Pass request to next handler if valid

```json
// 403 response
{
  "error": "unauthorized",
  "message": "invalid or missing API key"
}
```

---

## Logging (JSON structured logs via zap)

Use `go.uber.org/zap` in **production mode** — this outputs structured JSON by default.
No plain-text logs anywhere in the project.

```go
// In cmd/main.go — build the logger once and pass it down
logger, _ := zap.NewProduction()
defer logger.Sync()
```

Every log line must be a valid JSON object. Do not use `fmt.Println` or `log.Printf` anywhere.

### Logger injection
Pass `*zap.Logger` as a field on each client and resolver struct.
Do not use a global logger variable.

```go
type AdminClient struct {
    cfg        *config.Config
    httpClient *resty.Client
    logger     *zap.Logger   // injected at construction
}
```

### Required fields per upstream API call log line

| Field | Type | Value |
|---|---|---|
| `request_id` | string | from context (set by request_id middleware) |
| `app_id` | string | applicationId for this request |
| `service` | string | "admin" / "customer" / "razorpay" |
| `endpoint` | string | exact URL path called |
| `duration_ms` | int64 | elapsed time in milliseconds |
| `status_code` | int | HTTP status returned by upstream |
| `error` | string | error message if call failed, omit if nil |

```go
// Example log call inside a client function
logger.Info("upstream call",
    zap.String("request_id", requestIDFromCtx(ctx)),
    zap.String("app_id", appID),
    zap.String("service", "admin"),
    zap.String("endpoint", cfg.Admin.Endpoints.Disbursement),
    zap.Int64("duration_ms", elapsed.Milliseconds()),
    zap.Int("status_code", resp.StatusCode()),
)
```

### Request ID extraction helper
Add a small helper in `middleware/request_id.go` to store and retrieve the request ID from context:

```go
type contextKey string
const requestIDKey contextKey = "request_id"

// SetRequestID stores the request ID in context
func SetRequestID(ctx context.Context, id string) context.Context

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string
```

### Log levels to use

| Situation | Level |
|---|---|
| Server started, Redis connected | `Info` |
| Upstream call succeeded | `Info` |
| Cache hit | `Debug` |
| Cache miss | `Debug` |
| Upstream call failed (after retry) | `Error` |
| Auth rejected | `Warn` |
| Resolver returned partial failure | `Warn` |
| Panic recovered | `Error` |

---

## Entry Point (cmd/main.go)

```
1.  Load config from .env using viper
2.  Set up zap logger (zap.NewProduction() — JSON output)
3.  Connect to Redis only if CACHE_ENABLED=true
4.  Create AdminClient with config
5.  Create CustomerClient with config
6.  Create RazorpayClient with config
7.  Create Cache with config
8.  Build gqlgen schema — inject all clients into resolvers
9.  Set up Gin router
10. Apply request ID middleware to all routes (adds unique ID to every request)
11. Apply auth middleware to all routes
12. Mount GraphQL handler at POST /graphql
13. Mount health check at GET /health (returns 200 OK always if server is running)
14. Start server on config.Port
15. Listen for OS signals (SIGTERM, SIGINT) for graceful shutdown
16. On shutdown: stop accepting new requests, wait for in-flight requests to finish
```

Keep main.go clean — only wiring, no business logic.

---

## Important Rules

1.  All endpoint URLs defined in `config.go` only — nowhere else
2.  Each resolver is independent — no resolver calls another resolver
3.  `CACHE_ENABLED=false` must work perfectly with no Redis running
4.  Keep functions small — one function does one thing
5.  Add a comment on every function explaining what it does
6.  Use `context.Context` in all functions for timeout and cancellation
7.  Return proper errors — never use panic
8.  No business logic in `main.go` — only wiring and startup
9.  Log every upstream API call as structured JSON (zap) — fields: request_id, app_id, service, endpoint, duration_ms, status_code, error
10. If one resolver fails, all other resolvers must still return their data
    — never let one failed section break the entire response
11. Upstream response structs go in `model/upstream/` — GraphQL model structs go in `model/graph/`
    — never mix them
12. Every request must have a unique request ID in logs
    — generated by request_id middleware, passed via context
13. `.env` must be in `.gitignore` — never committed
14. Retry upstream API call once on failure before returning error

---

## What to Build First

```
Phase A — get disbursement working end to end:
  1. Config loads from .env
  2. Auth middleware rejects invalid keys
  3. GraphQL schema defined
  4. Root resolver returns LoanData with applicationId
  5. Disbursement resolver calls admin_client
  6. admin_client calls upstream API (mock response ok for now)
  7. Cache stores and retrieves disbursement data
  8. Response trimmed to requested fields by gqlgen
  9. Test with curl

Phase B — repeat same pattern for:
  repayment → mandate → payments

Phase C — add remaining sections from tech spec

Phase D — tests:
  write unit test for disbursement resolver
  mock the admin_client so no real API call is made
  test cache hit path and cache miss path separately
  same pattern repeats for all other resolvers
```

---

## Test curl Once Running

```bash
curl -X POST http://localhost:8080/graphql \
  -H "X-API-Key: gqbrain-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { loanData(applicationId: \"APP123\") { disbursement { utr date status } } }"
  }'
```

Expected response:
```json
{
  "data": {
    "loanData": {
      "disbursement": {
        "utr": "RTGS25121067890",
        "date": "2025-12-10",
        "status": "Completed"
      }
    }
  }
}
```

---

## Reference Documents

| Document | Path |
|---|---|
| Technical Specification | `docs/TECH_SPEC.md` |
| API Flow Visual | `docs/API_FLOW_VISUAL.md` |
| Field to API Mapping | `docs/models.md` |
| Caching Strategy | `docs/CACHING_STRATEGY.md` |
| Flow Summary | `docs/flow_summary.md` |
| Sample Python Demo | `sample/app.py` |