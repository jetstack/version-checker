package util

import (
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
)

func TestHTTPBackOff(t *testing.T) {
	tests := []struct {
		name       string
		min        time.Duration
		max        time.Duration
		attemptNum int
		resp       *http.Response
		expSleep   time.Duration
	}{
		{
			name:       "sleep within max duration",
			min:        100 * time.Millisecond,
			max:        1 * time.Second,
			attemptNum: 1,
			resp:       nil,
			expSleep:   retryablehttp.DefaultBackoff(100*time.Millisecond, 1*time.Second, 1, nil),
		},
		{
			name:       "sleep exceeds max duration (too many requests)",
			min:        100 * time.Millisecond,
			max:        500 * time.Millisecond,
			attemptNum: 10,
			resp: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": []string{"484289h20m8s"}}},
			expSleep: 500 * time.Millisecond,
		},
		{
			name:       "zero max duration",
			min:        100 * time.Millisecond,
			max:        0,
			attemptNum: 1,
			resp:       nil,
			expSleep:   0,
		},
		{
			name:       "negative max duration",
			min:        100 * time.Millisecond,
			max:        -1 * time.Second,
			attemptNum: 1,
			resp:       nil,
			expSleep:   time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSleep := HTTPBackOff(test.min, test.max, test.attemptNum, test.resp)
			assert.Equal(t, test.expSleep, gotSleep, "unexpected sleep duration")
		})
	}
}
