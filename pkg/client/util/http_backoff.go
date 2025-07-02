package util

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// This is a custom Backoff that enforces the Max wait duration.
// If the sleep is greater we only sleep for the max duration.
func HTTPBackOff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	sleep := retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	if sleep.Abs() >= max {
		return max.Abs()
	}

	return sleep.Abs()
}
