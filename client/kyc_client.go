package client

import (
	"context"
	"fmt"

	"data-hub/config"
	"data-hub/model/upstream"

	"go.uber.org/zap"
)

// KycClient handles all calls to the svc-kyc service.
type KycClient struct {
	http *HTTPClient
	cfg  *config.Config
}

// NewKycClient creates a KycClient.
func NewKycClient(cfg *config.Config, logger *zap.Logger) *KycClient {
	return &KycClient{
		http: NewHTTPClient("kyc", cfg.Kyc.BaseURL, cfg.Kyc.APIKey, logger),
		cfg:  cfg,
	}
}

// FetchKycStatus fetches KYC status entries (okyc, ckyc, pkyc, vkyc) for an application.
// GET /v1/misc/kyc/{appID}/status
func (c *KycClient) FetchKycStatus(ctx context.Context, appID string) (*upstream.KycStatusAPIResponse, error) {
	var result upstream.KycStatusAPIResponse
	endpoint := fmt.Sprintf(c.cfg.Kyc.Endpoints.Status, appID)
	err := c.http.Get(ctx, appID, endpoint, map[string]string{}, &result)
	return &result, err
}
