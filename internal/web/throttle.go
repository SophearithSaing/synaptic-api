package web

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ThrottleConfig describes a fixed-window rate limit.
type ThrottleConfig struct {
	// Limit is the maximum number of requests per window.
	Limit int
	// TTL is the window length.
	TTL time.Duration
	// Block is how long requests are rejected after the limit is
	// exceeded.
	Block time.Duration
}

// throttledMessage is the pinned 429 response message.
const throttledMessage = "ThrottlerException: Too Many Requests"

// unknownClientIP is the shared bucket for requests whose peer address
// cannot be resolved.
const unknownClientIP = "unknown"

// ThrottleState is the persisted fixed-window state of one rate limit
// key.
type ThrottleState struct {
	// Epoch is the start of the counting window.
	Epoch time.Time
	// Expire is the horizon of the counting window: the first instant
	// after which the window rolls over.
	Expire time.Time
	// Count is the number of hits recorded inside the window.
	Count int
	// BlockedUntil is a future instant while rejected requests share
	// the pinned 429 with a Retry-After countdown.
	BlockedUntil time.Time
}

// ThrottleStore persists one ThrottleState per rate limit key. The
// Throttler owns the fixed-window rules and applies them as one atomic
// state update; a store must serialize concurrent updates of the same
// key so no hit is lost between the read and the write. An in-process
// store locks; a shared store needs the equivalent guarantee, for
// example by performing the pair as one transaction.
type ThrottleStore interface {
	// Update atomically applies mutate to the key's stored state —
	// seeding a zero state on first use — and returns the stored
	// result. Concurrent updates of one key take effect one after
	// another; no increment may be lost.
	Update(
		key string,
		mutate func(state ThrottleState) ThrottleState,
	) (ThrottleState, error)
	// Size reports the number of stored keys so the Throttler can
	// trigger pruning.
	Size() int
	// Prune deletes every entry whose key and state fail the keeper.
	Prune(keep func(key string, state ThrottleState) bool)
}

// NewThrottler builds a Throttler with global limits, per-route
// overrides keyed by "METHOD /path", the client resolver, and the
// throttle state store. A nil store falls back to in-process memory.
func NewThrottler(
	global ThrottleConfig,
	overrides map[string]ThrottleConfig,
	clientIP ClientIPFunc,
	store ThrottleStore,
) *Throttler {
	if store == nil {
		store = NewMemoryThrottleStore()
	}

	return &Throttler{
		global:    global,
		overrides: overrides,
		clientIP:  clientIP,
		store:     store,
	}
}

// ClientIPFunc resolves the client address a request is throttled on.
// Resolution must be explicit: forwarding headers are spoofable and may
// only be honored when the peer is a trusted interpreter of them.
type ClientIPFunc func(*http.Request) string

// Throttler applies fixed-window rate limits per client address, with
// per-route overrides, over a pluggable ThrottleStore.
//
// The wired deployment keeps its state in one process: limits reset on
// restart, replicas count independently, and every client behind a
// proxy shares the proxy's bucket when the client resolver reads the
// transport peer. Multi-replica deployments need a shared ThrottleStore
// implementation before the limits are meaningful at scale.
type Throttler struct {
	global    ThrottleConfig
	overrides map[string]ThrottleConfig
	clientIP  ClientIPFunc
	store     ThrottleStore
}

// NewMemoryThrottleStore builds the in-process ThrottleStore.
func NewMemoryThrottleStore() ThrottleStore {
	return newMemoryThrottleStore()
}

// memoryThrottleStore keeps throttle windows in process memory under
// one lock, applying every update atomically under it.
type memoryThrottleStore struct {
	mu      sync.Mutex
	buckets map[string]ThrottleState
}

func newMemoryThrottleStore() *memoryThrottleStore {
	return &memoryThrottleStore{
		buckets: make(map[string]ThrottleState),
	}
}

// Update implements ThrottleStore. The whole read-mutate-write happens
// under one lock, so concurrent hits of one key never lose increments.
func (m *memoryThrottleStore) Update(
	key string,
	mutate func(state ThrottleState) ThrottleState,
) (ThrottleState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := mutate(m.buckets[key])
	m.buckets[key] = updated

	return updated, nil
}

// Size implements ThrottleStore.
func (m *memoryThrottleStore) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.buckets)
}

// Prune implements ThrottleStore.
func (m *memoryThrottleStore) Prune(
	keep func(key string, state ThrottleState) bool,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, state := range m.buckets {
		if !keep(key, state) {
			delete(m.buckets, key)
		}
	}
}

// throttleBucketLimit bounds the stored keys before stale windows
// become reclaim candidates.
const throttleBucketLimit = 65536

// hit records one request. It reports whether the request is allowed
// and, when rejected, how long remains until the block or window
// expires. All instants derive from the injected now. Store failures
// surface as errors so the caller can decide the failure policy; the
// stored state stays untouched then.
func (th *Throttler) hit(
	key string,
	now time.Time,
	config ThrottleConfig,
) (allowed bool, retryAfter time.Duration, err error) {
	state, err := th.store.Update(key, func(current ThrottleState) ThrottleState {
		current, allowed, retryAfter = hitWindow(current, now, config)

		return current
	})
	if err != nil {
		return false, 0, err
	}

	if state.Epoch.Equal(now) && th.store.Size() > throttleBucketLimit {
		th.prune(now)
	}

	return allowed, retryAfter, nil
}

// hitWindow applies one hit to the state and reports its effect: the
// updated state, whether the request is allowed, and the retry
// countdown when rejected. A zero state behaves like a fresh window,
// so updates need no separate existence signal.
func hitWindow(
	state ThrottleState,
	now time.Time,
	config ThrottleConfig,
) (ThrottleState, bool, time.Duration) {
	if now.Before(state.BlockedUntil) {
		return state, false, state.BlockedUntil.Sub(now)
	}
	if now.After(state.Expire) {
		state = ThrottleState{
			Epoch:  now,
			Expire: now.Add(config.TTL),
		}
	}

	state.Count++
	if state.Count > config.Limit {
		state.BlockedUntil = now.Add(config.Block)
		return state, false, config.Block
	}

	return state, true, 0
}

// prune drops windows past their counting horizon that are not
// blocked, through the store.
func (th *Throttler) prune(now time.Time) {
	th.store.Prune(func(_ string, state ThrottleState) bool {
		if now.Before(state.BlockedUntil) {
			return true
		}

		return !now.After(state.Expire)
	})
}

// reject writes the pinned 429 body with the Retry-After header.
func rejectThrottled(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set(
		"Retry-After",
		strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))),
	)
	WriteJSON(w, http.StatusTooManyRequests, throttledBody{
		StatusCode: http.StatusTooManyRequests,
		Message:    throttledMessage,
	})
}

// throttledBody is the pinned 429 body with statusCode first.
type throttledBody struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// Middleware returns rate-limiting middleware applying the global
// limits and any route override for the request. A throttle store
// failure fails open: the outage is logged and the request proceeds
// rather than blackholing traffic.
func (th *Throttler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := ""
		if th.clientIP != nil {
			clientIP = th.clientIP(r)
		}
		if clientIP == "" {
			clientIP = hostOnly(r.RemoteAddr)
		}

		allowed, retryAfter, err := th.hit(
			clientIP+"|", time.Now(), th.global,
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "throttle store failed; failing open",
				"error", err)
			allowed = true
		}
		if !allowed {
			rejectThrottled(w, retryAfter)
			return
		}

		if override, ok := th.overrides[r.Method+" "+r.URL.Path]; ok {
			allowed, retryAfter, err = th.hit(
				clientIP+"|"+r.Method+" "+r.URL.Path,
				time.Now(),
				override,
			)
			if err != nil {
				slog.ErrorContext(
					r.Context(), "throttle store failed; failing open",
					"error", err,
				)
				allowed = true
			}
			if !allowed {
				rejectThrottled(w, retryAfter)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// DirectClientIP resolves the transport peer address of a request,
// dropping any port. Peers without a parseable IP address share the
// conservative "unknown" bucket instead of a spoofable key.
func DirectClientIP(r *http.Request) string {
	host := hostOnly(r.RemoteAddr)
	if net.ParseIP(host) == nil {
		return unknownClientIP
	}

	return host
}

// hostOnly returns the host part of an address, dropping the port.
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}
