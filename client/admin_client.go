package client

import (
	"context"
	"fmt"

	"data-hub/config"
	"data-hub/model/upstream"

	"go.uber.org/zap"
)

// AdminClient handles all calls to the gqadmin_backend service.
// Add new methods here as you integrate more admin API endpoints.
// All HTTP logic (logging, retries, error handling) is in HTTPClient — not here.
type AdminClient struct {
	http *HTTPClient
	cfg  *config.Config
}

// NewAdminClient creates an AdminClient.
func NewAdminClient(cfg *config.Config, logger *zap.Logger) *AdminClient {
	return &AdminClient{
		http: NewHTTPClient("admin", cfg.Admin.BaseURL, cfg.Admin.APIKey, logger),
		cfg:  cfg,
	}
}

// FetchDisbursement fetches disbursement tranche details for an application.
// GET /v0.1/disbursal-requests/tranch-details/fetch?application_id={appID}
func (c *AdminClient) FetchDisbursement(ctx context.Context, appID string) (*upstream.DisbursementAPIResponse, error) {
	var result upstream.DisbursementAPIResponse
	err := c.http.Get(ctx, appID, c.cfg.Admin.Endpoints.Disbursement,
		map[string]string{"application_id": appID}, // must be application_id, not app_id
		&result,
	)
	return &result, err
}

// FetchPayments fetches payment transactions for an application.
// GET /v1/payments/transactions/fetch?application_id={appID}&page=1
func (c *AdminClient) FetchPayments(ctx context.Context, appID string) (*upstream.PaymentsAPIResponse, error) {
	var result upstream.PaymentsAPIResponse
	err := c.http.Get(ctx, appID, c.cfg.Admin.Endpoints.Payments,
		map[string]string{"application_id": appID, "page": "1"},
		&result,
	)
	return &result, err
}

// FetchRefunds fetches all refund records for an application.
// GET /v1/refunds/requests/fetch?application_id={appID}&page_num=1
func (c *AdminClient) FetchRefunds(ctx context.Context, appID string) (*upstream.RefundsAPIResponse, error) {
	var result upstream.RefundsAPIResponse
	err := c.http.Get(ctx, appID, c.cfg.Admin.Endpoints.Refunds,
		map[string]string{"application_id": appID, "page_num": "1"},
		&result,
	)
	return &result, err
}

// FetchApplicationSummary fetches stage, product name, tags, and tracker for an application.
// GET /v0.1/applications/summary/{appID}/fetch
func (c *AdminClient) FetchApplicationSummary(ctx context.Context, appID string) (*upstream.ApplicationSummaryAPIResponse, error) {
	var result upstream.ApplicationSummaryAPIResponse
	endpoint := fmt.Sprintf(c.cfg.Admin.Endpoints.ApplicationSummary, appID)
	err := c.http.Get(ctx, appID, endpoint, map[string]string{}, &result)
	return &result, err
}

