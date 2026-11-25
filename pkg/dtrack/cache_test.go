package dtrack

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func countingHandler(requests *atomic.Int32, delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(delay)
		_, _ = w.Write([]byte(`{"version":"5.2.2"}`))
	}
}

func withCache(ttl time.Duration) func(*Options) {
	return func(o *Options) { o.CacheTTL = ttl }
}

func TestCacheServesRepeatedRequests(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, countingHandler(&requests, 0), withCache(time.Minute))
	for range 3 {
		v, err := c.Version(context.Background())
		must(t, err)
		if v.Version != "5.2.2" {
			t.Fatalf("version %q", v.Version)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("got %d requests, want 1", requests.Load())
	}
}

func TestCacheExpires(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, countingHandler(&requests, 0), withCache(time.Minute))
	now := time.Now()
	c.cache.now = func() time.Time { return now }
	_, err := c.Version(context.Background())
	must(t, err)
	now = now.Add(2 * time.Minute)
	_, err = c.Version(context.Background())
	must(t, err)
	if requests.Load() != 2 {
		t.Fatalf("got %d requests, want 2", requests.Load())
	}
}

func TestWithoutCacheBypassesCache(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, countingHandler(&requests, 0), withCache(time.Minute))
	ctx := WithoutCache(context.Background())
	for range 2 {
		_, err := c.Version(ctx)
		must(t, err)
	}
	if requests.Load() != 2 {
		t.Fatalf("got %d requests, want 2", requests.Load())
	}
}

func TestErrorsAreNotCached(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "nope", http.StatusForbidden)
	}), withCache(time.Minute))
	for range 2 {
		if _, err := c.Version(context.Background()); !IsStatus(err, http.StatusForbidden) {
			t.Fatalf("err = %v", err)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("got %d requests, want 2", requests.Load())
	}
}

func TestConcurrentIdenticalRequestsShareOneRoundTrip(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, countingHandler(&requests, 50*time.Millisecond))
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := c.Version(context.Background())
			must(t, err)
		})
	}
	wg.Wait()
	if requests.Load() != 1 {
		t.Fatalf("got %d requests, want 1", requests.Load())
	}
}

func TestCancelledWaiterDoesNotFailOthers(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, countingHandler(&requests, 100*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		_, err := c.Version(ctx)
		errs <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if _, err := c.Version(context.Background()); err != nil {
		t.Fatalf("second caller failed: %v", err)
	}
	if err := <-errs; err != context.Canceled {
		t.Fatalf("cancelled caller got %v", err)
	}
}

func TestMaxConcurrentLimitsInFlightRequests(t *testing.T) {
	var current, peak atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := current.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)
		_, _ = w.Write([]byte(`{}`))
	}), func(o *Options) { o.MaxConcurrent = 2 })

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			_, err := c.Project(context.Background(), string(rune('a'+i)))
			must(t, err)
		})
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatalf("peak concurrency %d, want <= 2", peak.Load())
	}
}

func TestCacheEvictsByBytes(t *testing.T) {
	cache := newResponseCache(time.Minute, 10)
	cache.set("a", []byte("12345"), nil)
	cache.set("b", []byte("12345"), nil)
	if _, ok := cache.get("a"); !ok {
		t.Fatal("a evicted too early")
	}
	cache.set("c", []byte("12345"), nil)
	if _, ok := cache.get("b"); ok {
		t.Fatal("least recently used entry b should be evicted")
	}
	if _, ok := cache.get("a"); !ok {
		t.Fatal("recently used entry a should stay")
	}
	cache.set("huge", make([]byte, 11), nil)
	if _, ok := cache.get("huge"); ok {
		t.Fatal("entries larger than the cache must not be stored")
	}
}

func TestEndpointLabel(t *testing.T) {
	for in, want := range map[string]string{
		"/version":                                    "/version",
		"/v1/project/tag/team-payments":               "/v1/project/tag/:param",
		"/v1/metrics/project/1f0e/since/20261101":     "/v1/metrics/project/:param/since/:param",
		"/v1/finding/project/193995f2-2205-4b1b-ac0b": "/v1/finding/project/:param",
		"/v1/metrics/portfolio/current":               "/v1/metrics/portfolio/current",
	} {
		if got := endpoint(in); got != want {
			t.Errorf("endpoint(%q) = %q, want %q", in, got, want)
		}
	}
}
