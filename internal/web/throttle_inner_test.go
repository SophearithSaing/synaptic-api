package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// hitConfig is the single-window config used by the injected-clock
// tests.
var hitConfig = ThrottleConfig{
	Limit: 1, TTL: time.Minute, Block: 2 * time.Minute,
}

// TestThrottlerHitUsesInjectedClock pins the injected-clock behavior:
// the blocked countdown derives from the injected instant, and the
// window resets by its configured TTL, not a separate cleanup period.
func TestThrottlerHitUsesInjectedClock(t *testing.T) {
	throttler := NewThrottler(hitConfig, map[string]ThrottleConfig{}, nil, nil)
	ctx := context.Background()

	now := time.Now()
	allowed, _, err := throttler.hit(ctx, "ip|", now, hitConfig)
	if err != nil || !allowed {
		t.Fatalf("first request: allowed=%v err=%v", allowed, err)
	}

	over, retryAfter, err := throttler.hit(ctx, "ip|", now.Add(time.Second), hitConfig)
	if err != nil {
		t.Fatal(err)
	}
	if over {
		t.Fatal("second request exceeds the limit")
	}
	if retryAfter != 2*time.Minute {
		t.Fatalf("retryAfter %v, want the full block duration", retryAfter)
	}

	blocked, retryAfter, err := throttler.hit(
		ctx, "ip|", now.Add(30*time.Second), hitConfig,
	)
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("blocked request must be rejected")
	}
	// The countdown is the remaining block on the injected clock:
	// blockedUntil is now+1s+2m, requested 29 seconds later.
	if retryAfter != 91*time.Second {
		t.Fatalf("retryAfter %v, want the remaining injected block", retryAfter)
	}

	// At block expiry the counting window has also rolled over, so the
	// request is allowed again.
	reset, _, err := throttler.hit(
		ctx, "ip|", now.Add(2*time.Minute+time.Second), hitConfig,
	)
	if err != nil || !reset {
		t.Fatalf("post-block request: allowed=%v err=%v", reset, err)
	}
}

// TestThrottlerHitStopsOnStoreFailure pins that a store failure
// surfaces instead of the throttler inventing a decision.
func TestThrottlerHitStopsOnStoreFailure(t *testing.T) {
	store := &failingThrottleStore{}
	throttler := NewThrottler(
		hitConfig, map[string]ThrottleConfig{}, nil, store,
	)

	_, _, err := throttler.hit(
		context.Background(), "ip|", time.Now(), hitConfig,
	)
	if err != errStoreOutage {
		t.Fatalf("store failure must surface, got %v", err)
	}
}

// failingThrottleStore models an unreachable shared store.
type failingThrottleStore struct{}

// errStoreOutage models a store outage for the failure-path test.
var errStoreOutage = errors.New("store outage")

// Update implements ThrottleStore.
func (f *failingThrottleStore) Update(
	context.Context, string, func(ThrottleState) ThrottleState,
) (ThrottleState, error) {
	return ThrottleState{}, errStoreOutage
}

// Size implements ThrottleStore.
func (f *failingThrottleStore) Size(context.Context) (int, error) {
	return 0, errStoreOutage
}

// Prune implements ThrottleStore.
func (f *failingThrottleStore) Prune(
	context.Context, time.Time,
) (int64, error) {
	return 0, errStoreOutage
}

// TestThrottlerConcurrentHitsPinAtomicUpdates fires many concurrent
// hits at one key and pins that the atomic store update loses no
// increments: exactly Limit requests pass per window.
func TestThrottlerConcurrentHitsPinAtomicUpdates(t *testing.T) {
	limit := 100
	config := ThrottleConfig{
		Limit: limit, TTL: time.Minute, Block: 5 * time.Minute,
	}
	throttler := NewThrottler(config, map[string]ThrottleConfig{}, nil, nil)
	ctx := context.Background()

	attempts := 4 * limit
	results := make([]bool, attempts)
	var failures sync.Mutex
	var failed []error

	var waiting sync.WaitGroup
	waiting.Add(attempts)
	start := make(chan struct{})

	for slot := range attempts {
		go func(slot int) {
			defer waiting.Done()
			<-start

			permitted, _, err := throttler.hit(ctx, "ip|", time.Now(), config)
			if err != nil {
				failures.Lock()
				failed = append(failed, err)
				failures.Unlock()

				return
			}
			results[slot] = permitted
		}(slot)
	}
	close(start)
	waiting.Wait()

	if len(failed) != 0 {
		t.Fatalf("store errors under load: %v", failed[:min(len(failed), 3)])
	}

	passed := 0
	for _, permitted := range results {
		if permitted {
			passed++
		}
	}
	if passed != limit {
		t.Fatalf(
			"allowed %d hits, want exactly %d — lost updates",
			passed, limit,
		)
	}
}

// TestMemoryStorePruneAndSize pin the retained store operations on the
// axis the shared store must mirror: pruning only expired unblocked
// windows and sizing the entries.
func TestMemoryStorePruneAndSize(t *testing.T) {
	store := newMemoryThrottleStore()
	ctx := context.Background()
	now := time.Now()

	store.buckets["expired"] = ThrottleState{
		Epoch: now.Add(-5 * time.Minute), Expire: now.Add(-time.Minute),
	}
	store.buckets["open"] = ThrottleState{
		Epoch: now, Expire: now.Add(time.Minute), Count: 2,
	}
	store.buckets["blocked"] = ThrottleState{
		Epoch:        now.Add(-10 * time.Minute),
		Expire:       now.Add(-time.Minute),
		BlockedUntil: now.Add(time.Minute),
	}

	if size, err := store.Size(ctx); err != nil || size != 3 {
		t.Fatalf("size %v err %v, want 3", size, err)
	}

	deleted, err := store.Prune(ctx, now)
	if err != nil || deleted != 1 {
		t.Fatalf("pruned %d err %v, want 1 expired unblocked window", deleted, err)
	}
	if _, kept := store.buckets["expired"]; kept {
		t.Fatal("expired unblocked window must be pruned")
	}
	if _, kept := store.buckets["open"]; !kept {
		t.Fatal("open window must stay")
	}
	if _, kept := store.buckets["blocked"]; !kept {
		t.Fatal("blocked window must stay until its block ends")
	}
}

// TestThrottlerMiddlewareResolvesClientIP pins the explicit resolver:
// resolver output keys the bucket and an empty resolution falls back
// to the transport peer.
func TestThrottlerMiddlewareResolvesClientIP(t *testing.T) {
	throttler := NewThrottler(
		ThrottleConfig{Limit: 1, TTL: time.Minute, Block: time.Minute},
		map[string]ThrottleConfig{},
		func(*http.Request) string { return "shared" },
		nil,
	)
	handler := throttler.Middleware(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)

	first := httptest.NewRequest(http.MethodGet, "/anything", nil)
	first.RemoteAddr = "10.0.0.1:1234"
	one := httptest.NewRecorder()
	handler.ServeHTTP(one, first)
	if one.Code != 200 {
		t.Fatalf("first request got %d", one.Code)
	}

	// A different transport peer sharing the resolved identity also
	// counts against the same bucket.
	second := httptest.NewRequest(http.MethodGet, "/anything", nil)
	second.RemoteAddr = "10.0.0.2:1234"
	two := httptest.NewRecorder()
	handler.ServeHTTP(two, second)
	if two.Code != 429 {
		t.Fatalf("shared resolver must bucket together, got %d", two.Code)
	}
}

// TestThrottlerPrunesThroughStore pins the over-cap pruning trigger on
// a fresh bucket.
func TestThrottlerPrunesThroughStore(t *testing.T) {
	store := newMemoryThrottleStore()
	now := time.Now()
	for index := range throttleBucketLimit + 1 {
		key := "k" + string(rune('A'+index%26)) +
			time.UnixMilli(int64(index)).String()
		store.buckets[key] = ThrottleState{
			Epoch:  now.Add(-2 * time.Minute),
			Expire: now.Add(-time.Minute),
		}
	}
	if size, err := store.Size(context.Background()); err != nil ||
		size <= throttleBucketLimit {
		t.Fatal("test preconditions require an over-cap store")
	}

	throttler := NewThrottler(
		hitConfig, map[string]ThrottleConfig{}, nil, store,
	)
	allowed, _, err := throttler.hit(
		context.Background(), "ip|", now, hitConfig,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("fresh bucket request must be allowed")
	}
	if size, err := store.Size(context.Background()); err == nil &&
		size >= throttleBucketLimit {
		t.Fatalf("hit must have pruned stale windows, kept %d", size)
	}
}

// TestThrottlerDirectClientIPUnparseable pins the safe fallback for
// peers without a parseable IP address.
func TestThrottlerDirectClientIPUnparseable(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "not-an-address"

	if got := DirectClientIP(request); got != unknownClientIP {
		t.Fatalf("unparseable peer resolved %q", got)
	}

	request.RemoteAddr = net.JoinHostPort("10.1.2.3", "55")
	if got := DirectClientIP(request); got != "10.1.2.3" {
		t.Fatalf("peer with port resolved %q", got)
	}
}

// forwardedCase pairs one request configuration with the resolved
// throttled address.
type forwardedCase struct {
	name         string
	remoteAddr   string
	forwardedFor string
	trusted      []string
	want         string
	wantError    bool
}

// TestForwardedForClientIP pins the trusted-proxy resolution rules.
func TestForwardedForClientIP(t *testing.T) {
	cases := []forwardedCase{
		{
			name:         "directPeerIgnoresHeaders",
			remoteAddr:   "1.2.3.4:9999",
			forwardedFor: "203.0.113.9",
			want:         "1.2.3.4",
		},
		{
			name:         "untrustedPeerSpoofIgnored",
			remoteAddr:   "1.2.3.4:9999",
			forwardedFor: "203.0.113.9",
			trusted:      []string{"10.0.0.0/8"},
			want:         "1.2.3.4",
		},
		{
			name:         "trustedProxySingleHop",
			remoteAddr:   "10.0.0.1:9999",
			forwardedFor: "203.0.113.9",
			trusted:      []string{"10.0.0.0/8"},
			want:         "203.0.113.9",
		},
		{
			name:         "trustedChainSkipsTrustedHops",
			remoteAddr:   "10.0.0.1:9999",
			forwardedFor: "203.0.113.9, 10.0.0.9",
			trusted:      []string{"10.0.0.0/8"},
			want:         "203.0.113.9",
		},
		{
			name:         "allTrustedChainFallsBackToPeer",
			remoteAddr:   "10.0.0.1:9999",
			forwardedFor: "10.0.0.8, 10.0.0.9",
			trusted:      []string{"10.0.0.0/8"},
			want:         "10.0.0.1",
		},
		{
			name:         "trustedPeerWithoutHeader",
			remoteAddr:   "10.0.0.1:9999",
			forwardedFor: "",
			trusted:      []string{"10.0.0.0/8"},
			want:         "10.0.0.1",
		},
		{
			name:         "exactAddressTrustApplies",
			remoteAddr:   "192.0.2.1:9999",
			forwardedFor: "203.0.113.9",
			trusted:      []string{"192.0.2.1"},
			want:         "203.0.113.9",
		},
		{
			name:       "unparseablePeerFallsBack",
			remoteAddr: "not-an-address",
			trusted:    []string{"10.0.0.0/8"},
			want:       unknownClientIP,
		},
		{
			name:       "invalidTrustedValueFails",
			remoteAddr: "10.0.0.1:9999",
			trusted:    []string{"nope"},
			wantError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			resolve, err := ForwardedForClientIP(test.trusted)
			if test.wantError {
				if err == nil {
					t.Fatal("invalid trusted proxy must fail construction")
				}

				return
			}
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodGet, "/x", nil)
			request.RemoteAddr = test.remoteAddr
			if test.forwardedFor != "" {
				request.Header.Set("X-Forwarded-For", test.forwardedFor)
			}

			if got := resolve(request); got != test.want {
				t.Fatalf("resolved %q, want %q", got, test.want)
			}
		})
	}
}

// TestForwardedForClientIPWithoutTrustPreservesDirectBehavior pins
// that an empty trusted-proxy list ignores forwarding headers even
// from otherwise trusted-looking peers.
func TestForwardedForClientIPWithoutTrustPreservesDirectBehavior(t *testing.T) {
	resolve, err := ForwardedForClientIP(nil)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/x", nil)
	request.RemoteAddr = "10.0.0.1:9999"
	request.Header.Set("X-Forwarded-For", "203.0.113.9")

	if got := resolve(request); got != "10.0.0.1" {
		t.Fatalf("resolver keyed %q, want the direct peer", got)
	}
}

// TestForwardedForClientIPMultipleHeadersPinsChain pins chained
// forwarding headers, which Go accumulates as separate header values.
func TestForwardedForClientIPMultipleHeadersPinsChain(t *testing.T) {
	resolve, err := ForwardedForClientIP([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/x", nil)
	request.RemoteAddr = "10.0.0.1:9999"
	request.Header.Add("X-Forwarded-For", "203.0.113.9")
	request.Header.Add("X-Forwarded-For", "10.0.0.9")

	if got := resolve(request); got != "203.0.113.9" {
		t.Fatalf("chained headers resolved %q", got)
	}
}
