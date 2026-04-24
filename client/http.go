package client

import (
	"context"
	"fmt"
	"time"

	"data-hub/middleware"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// HTTPClient is a shared HTTP client used by all service clients (admin, customer, razorpay, etc.).
// It handles all the common boilerplate: timeouts, retries, headers, and structured logging.
//
// Each upstream service creates one of these with its own name, base URL, and API key.
// The service-specific client files (admin_client.go, customer_client.go, etc.) just call
// Get() or Post() with the right endpoint and typed result struct.
type HTTPClient struct {
	name    string // "admin", "customer", "razorpay" — used in log lines
	baseURL string
	apiKey  string
	resty   *resty.Client
	logger  *zap.Logger
}

// NewHTTPClient creates an HTTPClient for a single upstream service.
// name is used in log lines so you can tell which service is being called.
func NewHTTPClient(name, baseURL, apiKey string, logger *zap.Logger) *HTTPClient {
	return &HTTPClient{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		resty:   NewBaseHTTPClient(),
		logger:  logger,
	}
}

// Get makes a GET request to the given endpoint with query params.
// result must be a pointer to a struct — resty will unmarshal the JSON response into it.
// appID is only used for logging (pass "" if the call has no application ID).
func (c *HTTPClient) Get(ctx context.Context, appID, endpoint string, params map[string]string, result interface{}) error {
	start := time.Now()
	requestID := middleware.GetRequestID(ctx)

	resp, err := c.resty.R().
		SetContext(ctx).
		SetHeader("GQ-API-Key", c.apiKey).
		SetQueryParams(params).
		SetResult(result).
		Get(c.baseURL + endpoint)

	elapsed := time.Since(start)

	if err != nil {
		c.logger.Error("upstream call failed",
			zap.String("request_id", requestID),
			zap.String("app_id", appID),
			zap.String("service", c.name),
			zap.String("endpoint", endpoint),
			zap.Int64("duration_ms", elapsed.Milliseconds()),
			zap.Error(err),
		)
		return fmt.Errorf("%s GET %s: %w", c.name, endpoint, err)
	}

	c.logger.Info("upstream call",
		zap.String("request_id", requestID),
		zap.String("app_id", appID),
		zap.String("service", c.name),
		zap.String("endpoint", endpoint),
		zap.Int64("duration_ms", elapsed.Milliseconds()),
		zap.Int("status_code", resp.StatusCode()),
	)

	// Logs raw response body — useful for verifying upstream struct field names on first run.
	// Remove once model/upstream structs are confirmed correct.
	c.logger.Debug("raw upstream response", zap.String("body", resp.String()), zap.String("app_id", appID))

	if resp.IsError() {
		return fmt.Errorf("%s GET %s: HTTP %d", c.name, endpoint, resp.StatusCode())
	}

	return nil
}

// Post makes a POST request with a JSON body.
// body is marshalled to JSON automatically by resty.
// result must be a pointer to a struct for the response.
func (c *HTTPClient) Post(ctx context.Context, appID, endpoint string, body interface{}, result interface{}) error {
	start := time.Now()
	requestID := middleware.GetRequestID(ctx)

	resp, err := c.resty.R().
		SetContext(ctx).
		SetHeader("GQ-API-Key", c.apiKey).
		SetBody(body).
		SetResult(result).
		Post(c.baseURL + endpoint)

	elapsed := time.Since(start)

	if err != nil {
		c.logger.Error("upstream call failed",
			zap.String("request_id", requestID),
			zap.String("app_id", appID),
			zap.String("service", c.name),
			zap.String("endpoint", endpoint),
			zap.Int64("duration_ms", elapsed.Milliseconds()),
			zap.Error(err),
		)
		return fmt.Errorf("%s POST %s: %w", c.name, endpoint, err)
	}

	c.logger.Info("upstream call",
		zap.String("request_id", requestID),
		zap.String("app_id", appID),
		zap.String("service", c.name),
		zap.String("endpoint", endpoint),
		zap.Int64("duration_ms", elapsed.Milliseconds()),
		zap.Int("status_code", resp.StatusCode()),
	)

	c.logger.Debug("raw upstream response", zap.String("body", resp.String()), zap.String("app_id", appID))

	if resp.IsError() {
		return fmt.Errorf("%s POST %s: HTTP %d", c.name, endpoint, resp.StatusCode())
	}

	return nil
}