package util

import (
	"errors"
	"math"
	"net/http"
	"time"
)

// SendHttpRequestWithExpRetry executes sendRequest and based on shouldRetry either returns the result or
// executes beforeRetry and sendRequest with exponential backoff retry until sendRequest is either successful or
// maxRetries is reached
func SendHttpRequestWithExpRetry(sendRequest func() (*http.Response, error),
	shouldRetry func(resp *http.Response, err error) bool,
	beforeRetry func(resp *http.Response, err error) error,
	maxRetries int) (*http.Response, error) {
	for retry := 0; retry <= maxRetries; retry++ {
		resp, err := sendRequest()

		if !shouldRetry(resp, err) {
			return resp, err
		}

		if err = beforeRetry(resp, err); err != nil {
			return nil, err
		}

		retryDelaySeconds := math.Pow(2, float64(retry))
		time.Sleep(time.Duration(float32(retryDelaySeconds) * float32(time.Second)))
	}

	return nil, errors.New("max retry exceeded")
}
