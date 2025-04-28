package util

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

func RateLimitedBackoffLimiter(
	logger *logrus.Entry,
	limiter *rate.Limiter,
	maxWait time.Duration,
) retryablehttp.Backoff {
	return func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {

		defaultDelay := retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
		// Reserve first to introspect delay
		res := limiter.Reserve()
		if !res.OK() {
			logger.Error(fmt.Errorf("rate limit exceeded"), "Cannot make request")
			return maxWait // fallback
		}
		rateDelay := res.Delay()

		// Choose the larger of the two delays (rate limit or default backoff)
		delay := defaultDelay
		if rateDelay > delay {
			delay = rateDelay
		}

		if delay > maxWait {
			res.Cancel()
			logger.WithFields(logrus.Fields{
				"attempt": attemptNum, "wait": delay, "maxWait": maxWait,
			}).Info("Wait time too long, using max wait instead")
			return maxWait
		}

		logger.WithFields(logrus.Fields{
			"attempt": attemptNum, "wait": delay,
		}).Info("Waiting due to rate limit")
		return delay
	}
}
