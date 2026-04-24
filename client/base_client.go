package client

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

// NewBaseHTTPClient creates a shared resty HTTP client used by all service clients.
// Settings: 15s timeout, JSON headers, retry once on network error or 5xx.
func NewBaseHTTPClient() *resty.Client {
	return resty.New().
		SetTimeout(15 * time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetRetryCount(1).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			return err != nil || r.StatusCode() >= http.StatusInternalServerError
		})
}