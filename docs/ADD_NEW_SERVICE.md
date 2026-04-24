# Adding a New Service or API Endpoint

## Steps

### 1. `config/config.go` — add service block and TTL

Add the service struct:
```go
Customer struct {
    BaseURL string
    APIKey  string
    Endpoints struct {
        Repayment string
        // add more endpoints here as you integrate them
    }
}
```

Add TTL field to the existing `CacheTTL` struct:
```go
CacheTTL struct {
    Disbursement time.Duration
    Repayment    time.Duration  // add this
    // ...
}
```

In `Load()`:
```go
cfg.Customer.BaseURL = viper.GetString("CUSTOMER_BASE_URL")
cfg.Customer.APIKey  = viper.GetString("CUSTOMER_API_KEY")
cfg.Customer.Endpoints.Repayment = "/v1/repayment/schedule"

cfg.CacheTTL.Repayment = viper.GetDuration("CACHE_TTL_REPAYMENT")
if cfg.CacheTTL.Repayment == 0 {
    cfg.CacheTTL.Repayment = 30 * time.Minute // default if not set in env
}
```

---

### 2. `.env` and `.env.example` — add credentials and TTL

```
CUSTOMER_BASE_URL=https://...
CUSTOMER_API_KEY=...
CACHE_TTL_REPAYMENT=30m
```

TTL values use Go duration format: `2h`, `30m`, `5m`, etc. If not set, the default from `config.go` is used.

---

### 3. `model/upstream/customer.go` — raw API response struct

Mirrors exactly what the upstream API returns. Never returned to GraphQL consumers.

```go
package upstream

type RepaymentAPIResponse struct {
    Success bool              `json:"success"`
    Data    []RepaymentRecord `json:"data"`
}

type RepaymentRecord struct {
    // verify field names by checking the debug log "raw upstream response" on first run
}
```

---

### 4. `model/graph/loan.go` — GraphQL model struct

What you want to expose to consumers. All fields are pointers (`*string`, `*float64`)
so absent data returns `null` in GraphQL instead of empty string or zero.

```go
type Repayment struct {
    TotalEmis *int
    NextEmiDate *string
    // ...
}
```

---

### 5. `client/customer_client.go` — typed client method

No HTTP boilerplate needed — `HTTPClient` in `client/http.go` handles everything.
Just map the endpoint to a typed response.

```go
package client

import (
    "context"
    "data-hub/config"
    "data-hub/model/upstream"
    "go.uber.org/zap"
)

type CustomerClient struct {
    http *HTTPClient
    cfg  *config.Config
}

func NewCustomerClient(cfg *config.Config, logger *zap.Logger) *CustomerClient {
    return &CustomerClient{
        http: NewHTTPClient("customer", cfg.Customer.BaseURL, cfg.Customer.APIKey, logger),
        cfg:  cfg,
    }
}

func (c *CustomerClient) FetchRepayment(ctx context.Context, appID string) (*upstream.RepaymentAPIResponse, error) {
    var result upstream.RepaymentAPIResponse
    err := c.http.Get(ctx, appID, c.cfg.Customer.Endpoints.Repayment,
        map[string]string{"application_id": appID},
        &result,
    )
    return &result, err
}
```

> For POST endpoints use `c.http.Post(ctx, appID, endpoint, requestBody, &result)` instead.

---

### 6. `graph/schema.graphqls` — add field and type

```graphql
type LoanData {
  disbursement: Disbursements
  repayment:    Repayment      # add new field here
}

type Repayment {
  totalEmis:    Int
  nextEmiDate:  String
  # ...
}
```

Then run:
```bash
make generate
```

gqlgen will add a new resolver stub in `graph/schema.resolvers.go` automatically.

---

### 7. `graph/schema.resolvers.go` — implement the new stub

gqlgen generates a stub like:
```go
func (r *loanDataResolver) Repayment(ctx context.Context, obj *model.LoanData) (*model.Repayment, error) {
    panic(fmt.Errorf("not implemented"))
}
```

Replace it with the standard pattern:
```go
func (r *loanDataResolver) Repayment(ctx context.Context, obj *model.LoanData) (*model.Repayment, error) {
    cacheKey := fmt.Sprintf("gqdg:%s:repayment", obj.ApplicationID)

    // 1. cache check
    cached, _ := r.Cache.Get(ctx, cacheKey)
    if cached != "" {
        var info model.Repayment
        if err := json.Unmarshal([]byte(cached), &info); err == nil {
            return &info, nil
        }
    }

    // 2. upstream call
    apiResp, err := r.CustomerClient.FetchRepayment(ctx, obj.ApplicationID)
    if err != nil {
        r.Logger.Warn("repayment fetch failed", zap.String("app_id", obj.ApplicationID), zap.Error(err))
        return nil, nil // return nil not error so other fields still resolve
    }

    // 3. map upstream → graph model
    info := &model.Repayment{
        // map fields here
    }

    // 4. cache with TTL from config (set via CACHE_TTL_REPAYMENT env var)
    if bytes, err := json.Marshal(info); err == nil {
        _ = r.Cache.Set(ctx, cacheKey, string(bytes), r.Config.CacheTTL.Repayment)
    }

    return info, nil
}
```

---

### 8. `graph/resolver.go` — add the new client

```go
type Resolver struct {
    AdminClient    *client.AdminClient
    CustomerClient *client.CustomerClient  // add
    Cache          *cache.Cache
    Config         *config.Config
    Logger         *zap.Logger
}
```

---

### 9. `cmd/main.go` — wire it up

```go
customerClient := client.NewCustomerClient(cfg, logger)

resolver := &graph.Resolver{
    AdminClient:    adminClient,
    CustomerClient: customerClient,  // add
    Cache:          cacheStore,
    Config:         cfg,
    Logger:         logger,
}
```

---

## Cache TTL Reference

TTLs are configured via env vars using Go duration format (`2h`, `30m`, `5m`). Defaults are used if the env var is not set.

| Section       | Env Var                    | Default     |
|---------------|----------------------------|-------------|
| disbursement  | `CACHE_TTL_DISBURSEMENT`   | 2 hours     |
| repayment     | `CACHE_TTL_REPAYMENT`      | 30 minutes  |
| mandate       | `CACHE_TTL_MANDATE`        | 15 minutes  |
| payments      | `CACHE_TTL_PAYMENTS`       | 5 minutes   |
| kyc           | `CACHE_TTL_KYC`            | 1 hour      |
| application   | `CACHE_TTL_APPLICATION`    | 5 minutes   |
| refund        | `CACHE_TTL_REFUND`         | 15 minutes  |
| documents     | `CACHE_TTL_DOCUMENTS`      | 1 hour      |

---

## What You Actually Write (the 4 real steps)

| File | What to write |
|---|---|
| `model/upstream/` | copy field names from the API JSON response |
| `model/graph/loan.go` | define what fields to expose in GraphQL |
| `client/X_client.go` | 5 lines — call `c.http.Get(...)` or `c.http.Post(...)` |
| `graph/schema.resolvers.go` | cache → fetch → map → cache (same pattern every time) |

Everything else (config, schema, wiring) is the same mechanical edits each time.

---

## Upstream API Notes

- All upstream services use `GQ-API-Key` header (not `X-API-Key`)
- Admin disbursement uses `application_id` query param (not `app_id`)
- If the response body starts with `<`, the API key or base URL is wrong — upstream returns HTML on auth failure
- Check real field names by looking at the `raw upstream response` debug log on first run

If the upstream API adds a new field and you don't care about it, you do nothing. Go's JSON decoder ignores unknown fields by default.

You only touch the upstream model when you want to read a new field from the API response, and you only touch the graph model + schema when you want to expose it.

API adds new field                                                                                                                                                   
↓                                                                                                                                                             
Do you want to show it to consumers?                                                                                                                                 
├── No  → do nothing                                                                                                                                          
└── Yes → add to upstream model → add to graph model → add to schema → map in resolver                                                                        
                            