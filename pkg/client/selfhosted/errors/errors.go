package errors

type HTTPError struct {
	Body       []byte
	StatusCode int
}

func NewHTTPError(statusCode int, body []byte) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Body:       body,
	}
}

func (h *HTTPError) Error() string {
	return string(h.Body)
}

func IsHTTPError(err error) (*HTTPError, bool) {
	httpError, ok := err.(*HTTPError)
	return httpError, ok
}
