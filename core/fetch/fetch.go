// Package fetch does plain HTTP GETs with retries and a timeout. It knows
// nothing about specs, Stripe, or any other caller — just how to reliably
// download bytes from a URL.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxAttempts    = 3
	attemptTimeout = 30 * time.Second
)

// baseBackoff is a variable only so tests can shorten it.
var baseBackoff = 1 * time.Second

// StatusError is returned when a request completes but with a non-200 status,
// so callers can check the exact code (e.g. a 404 meaning "try elsewhere")
// instead of matching on an error string.
type StatusError struct {
	URL        string
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d from %s", e.StatusCode, e.URL)
}

// requestError marks a failure to even build the HTTP request (e.g. a
// malformed URL). It's never transient, so Get fails fast on it instead of
// retrying like it does for network errors and 5xx responses.
type requestError struct{ err error }

func (e *requestError) Error() string { return e.err.Error() }
func (e *requestError) Unwrap() error { return e.err }

// Get downloads url, retrying on network errors and 5xx responses up to
// maxAttempts times with exponential backoff (1s, 2s, 4s...). A 4xx response
// means the request itself is wrong, so it fails immediately instead of
// retrying. headers are set on every attempt; nil is fine for none.
func Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	r, err := get(ctx, url, headers)
	if err == nil && r.status == http.StatusNotModified {
		// Get sends no validators of its own, so a 304 is the server's error,
		// never an empty page.
		return nil, &StatusError{URL: url, StatusCode: r.status}
	}
	return r.body, err
}

// response is one successful attempt: 200 with its body, or 304 Not Modified
// to a conditional request (no body).
type response struct {
	status int
	body   []byte
	header http.Header
}

// get is Get's retry loop, returning the whole response.
func get(ctx context.Context, url string, headers map[string]string) (response, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r, err := attemptGet(ctx, url, headers)
		status := r.status
		if err == nil {
			return r, nil
		}
		lastErr = err

		var reqErr *requestError
		if errors.As(err, &reqErr) {
			return response{}, err // a malformed request won't build any differently on retry
		}

		if status >= 400 && status < 500 {
			return response{}, err // retrying a bad request never helps
		}

		if attempt < maxAttempts {
			backoff := baseBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return response{}, ctx.Err()
			}
		}
	}

	return response{}, fmt.Errorf("fetching %s failed after %d attempts: %w", url, maxAttempts, lastErr)
}

// attemptGet makes one attempt, bounded by its own timeout so a single hung
// request can't block forever. status is 0 when the failure happened before
// a response arrived (a bad request, a network error). A 304 is a success: it
// only comes back to a request that asked for it (If-None-Match,
// If-Modified-Since).
func attemptGet(ctx context.Context, url string, headers map[string]string) (response, error) {
	ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response{}, &requestError{err: fmt.Errorf("could not build the request: %w", err)}
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return response{}, fmt.Errorf("could not retrieve the content from given url: %w", err)
	}
	defer resp.Body.Close()

	r := response{status: resp.StatusCode, header: resp.Header}
	if resp.StatusCode == http.StatusNotModified {
		return r, nil
	}
	if resp.StatusCode != http.StatusOK {
		return r, &StatusError{URL: url, StatusCode: resp.StatusCode}
	}

	r.body, err = io.ReadAll(resp.Body)
	if err != nil {
		return r, fmt.Errorf("could not read the response body: %w", err)
	}
	return r, nil
}
