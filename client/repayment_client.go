package client

import (
	"context"

	"data-hub/config"
	"data-hub/model/upstream"

	"go.uber.org/zap"
)

// RepaymentClient handles all calls to the repayment service.
type RepaymentClient struct {
	http *HTTPClient
	cfg  *config.Config
}

// NewRepaymentClient creates a RepaymentClient.
func NewRepaymentClient(cfg *config.Config, logger *zap.Logger) *RepaymentClient {
	return &RepaymentClient{
		http: NewHTTPClient("repayment", cfg.Repayment.BaseURL, cfg.Repayment.APIKey, logger),
		cfg:  cfg,
	}
}

// FetchSchedule fetches the repayment schedule for an application.
// GET /v1/repayment-schedule/fetch?application_id={appID}
func (c *RepaymentClient) FetchSchedule(ctx context.Context, appID string) (*upstream.RepaymentScheduleAPIResponse, error) {
	var result upstream.RepaymentScheduleAPIResponse
	err := c.http.Get(ctx, appID, c.cfg.Repayment.Endpoints.Schedule,
		map[string]string{"application_id": appID},
		&result,
	)
	return &result, err
}
