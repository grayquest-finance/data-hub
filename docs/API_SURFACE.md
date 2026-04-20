The API surface is defined across a few files:

Route wiring — cmd/main.go

-   Line ~64: POST /graphql — the actual GraphQL data endpoint (auth-protected, allows introspection)
-   Line ~57: GET /graphql — Playground UI (public)
-   Line ~82: GET /health — liveness check
-   Line ~86: GET /schema — JSON schema tree (public)

GraphQL contract — graph/schema.graphqls  
The single source of truth for what the API accepts and returns. Every type/field lives here.

Query handlers — graph/schema.resolvers.go  
Where each GraphQL field is mapped to an upstream call:

-   LoanData — entrypoint resolver (line ~329)
-   Disbursements, ApplicationDetails, Kyc, Refunds, Payments, RepaymentAndEmi — one function per field

Upstream API calls — client/

-   admin_client.go — Admin service (disbursements, applications, payments, refunds)
-   kyc_client.go — KYC service
-   repayment_client.go — Repayment service
-   base_client.go, http.go — shared HTTP plumbing

Middleware — middleware/

-   auth.go — API key validation + introspection bypass
-   request_id.go — request ID propagation

Config — config/config.go  
Base URLs, API keys, endpoint paths, cache TTLs.

So: cmd/main.go is the "entry", graph/schema.graphqls is the "contract", graph/schema.resolvers.go is the "business logic", and client/ is the "outbound calls."