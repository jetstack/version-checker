package util

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestRateLimitedBackoffLimiter(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	maxWait := 5 * time.Second

	t.Run("default backoff delay is used when rate limiter allows immediate execution", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(1*time.Millisecond), 1)
		backoff := RateLimitedBackoffLimiter(logger, limiter, maxWait)

		min := 100 * time.Millisecond
		max := 200 * time.Millisecond
		attemptNum := 1

		delay := backoff(min, max, attemptNum, nil)
		assert.GreaterOrEqual(t, delay, min)
		assert.LessOrEqual(t, delay, max)
	})

	t.Run("rate limiter delay is used when it exceeds default backoff", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 1)
		backoff := RateLimitedBackoffLimiter(logger, limiter, maxWait)

		min := 100 * time.Millisecond
		max := 500 * time.Millisecond
		attemptNum := 3

		delay := backoff(min, max, attemptNum, nil)
		assert.GreaterOrEqual(t, delay, 500*time.Millisecond)
		assert.LessOrEqual(t, delay, maxWait)
	})

	t.Run("maxWait is used when delay exceeds maxWait", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(10*time.Second), 1)
		backoff := RateLimitedBackoffLimiter(logger, limiter, maxWait)

		min := maxWait - time.Second
		max := maxWait
		attemptNum := 3

		delay := backoff(min, max, attemptNum, nil)
		assert.Equal(t, maxWait, delay)
	})

	t.Run("rate limiter reservation fails", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(1*time.Second), 0) // No tokens available
		backoff := RateLimitedBackoffLimiter(logger, limiter, maxWait)

		min := 100 * time.Millisecond
		max := 500 * time.Millisecond
		attemptNum := 3

		delay := backoff(min, max, attemptNum, nil)
		assert.Equal(t, maxWait, delay)
	})
}
