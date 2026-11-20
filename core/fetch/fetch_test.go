package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// server answers with the given statuses in order (the last one repeats) and
// counts how many requests it saw.
func server(t *testing.T, statuses ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(hits.Add(1))
		status := statuses[min(n, len(statuses))-1]
		w.WriteHeader(status)
		if status == http.StatusOK {
			w.Write([]byte("spec-bytes"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func fastBackoff(t *testing.T) {
	t.Helper()
	old := baseBackoff
	baseBackoff = time.Millisecond
	t.Cleanup(func() { baseBackoff = old })
}

func TestRetriesTransientServerErrors(t *testing.T) {
	fastBackoff(t)
	srv, hits := server(t, 500, 503, 200)
	body, err := Get(context.Background(), srv.URL, nil)
	if err != nil || string(body) != "spec-bytes" {
		t.Fatalf("got %q, %v; want the body after two retries", body, err)
	}
	if hits.Load() != 3 {
		t.Errorf("%d attempts, want 3", hits.Load())
	}
}

func TestGivesUpAfterThreeAttempts(t *testing.T) {
	fastBackoff(t)
	srv, hits := server(t, 500)
	_, err := Get(context.Background(), srv.URL, nil)
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != 500 {
		t.Fatalf("got %v, want a wrapped StatusError 500", err)
	}
	if hits.Load() != maxAttempts {
		t.Errorf("%d attempts, want %d", hits.Load(), maxAttempts)
	}
}

// A 4xx means the request itself is wrong: retrying never helps, and the
// Stripe adapter relies on a fast, typed 404 to fall back to the legacy path.
func TestClientErrorsFailFastWithTheirStatus(t *testing.T) {
	fastBackoff(t)
	srv, hits := server(t, 404)
	_, err := Get(context.Background(), srv.URL, nil)
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != http.StatusNotFound {
		t.Fatalf("got %v, want StatusError 404", err)
	}
	if hits.Load() != 1 {
		t.Errorf("%d attempts on a 404, want 1", hits.Load())
	}
}

func TestMalformedURLFailsFast(t *testing.T) {
	fastBackoff(t)
	start := time.Now()
	if _, err := Get(context.Background(), "http://bad host/", nil); err == nil {
		t.Fatal("malformed URL: want an error")
	}
	if time.Since(start) > time.Second {
		t.Error("a malformed URL was retried")
	}
}

func TestSendsHeadersOnEveryAttempt(t *testing.T) {
	fastBackoff(t)
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		if len(seen) < 2 {
			w.WriteHeader(502)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	if _, err := Get(context.Background(), srv.URL, map[string]string{"Authorization": "Bearer t"}); err != nil {
		t.Fatal(err)
	}
	for i, h := range seen {
		if h != "Bearer t" {
			t.Errorf("attempt %d sent Authorization %q", i+1, h)
		}
	}
}

func TestCancelledContextStopsRetrying(t *testing.T) {
	old := baseBackoff
	baseBackoff = time.Hour // would hang forever without the context
	t.Cleanup(func() { baseBackoff = old })

	srv, _ := server(t, 500)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := Get(ctx, srv.URL, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want context.DeadlineExceeded", err)
	}
}
