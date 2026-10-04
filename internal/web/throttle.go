package web

import (
	"context"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ThrottleConfig describes a fixed-window rate limit.
type ThrottleConfig struct {
	Limit int
	TTL   time.Duration
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
	Epoch        time.Time
	Expire       time.Time
	Count        int
	BlockedUntil time.Time
}

// ThrottleStore persists one ThrottleState per rate limit key. The
// Throttler owns the fixed-window rules and applies them as one atomic
// state update; Update must serialize concurrent mutations of one key
// against each other so no hit is lost between the read and the
// write. An in-process store locks; a shared store upholds the same
// guarantee across processes through its own atomic update or
// compare-and-set loop.
type ThrottleStore interface {
	// Update atomically applies mutate to the key's stored state —
	// seeding a zero state on first use — and returns the stored
	// result. Concurrent updates of one key take effect one after
	// another; no increment may be lost.
	Update(
		ctx context.Context,
		key string,
		mutate func(state ThrottleState) ThrottleState,
	) (ThrottleState, error)
	// Size reports the number of stored keys so the Throttler can
	// trigger pruning.
	Size(ctx context.Context) (int, error)
	// Prune deletes every window that is neither blocked at now nor
	// inside a counting window still open at now, returning the
	// number of deleted entries.
	Prune(ctx context.Context, now time.Time) (int64, error)
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

// Throttler applies fixed-window rate limits per resolved client
// address, with per-route overrides, over a pluggable ThrottleStore.
// The wired deployment persists state in the shared throttle store, so
// limits also hold across restarts and replicas; unit tests and
// isolated local constructions may fall back to the memory store.
type Throttler struct {
	global    ThrottleConfig
	overrides map[string]ThrottleConfig
	clientIP  ClientIPFunc
	store     ThrottleStore
}

// NewMemoryThrottleStore builds the in-process ThrottleStore for unit
// tests and locally isolated constructions.
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
	_ context.Context,
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
func (m *memoryThrottleStore) Size(context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.buckets), nil
}

// Prune implements ThrottleStore.
func (m *memoryThrottleStore) Prune(
	_ context.Context,
	now time.Time,
) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for key, state := range m.buckets {
		if now.Before(state.BlockedUntil) {
			continue
		}
		if now.After(state.Expire) {
			delete(m.buckets, key)
			deleted++
		}
	}

	return deleted, nil
}

// throttleBucketLimit bounds the stored keys before stale windows
// become reclaim candidates.
const throttleBucketLimit = 65536

// hit records one request. It reports whether the request is allowed
// and, when rejected, how long remains until the block or window
// expires. All instants derive from the injected now; update failures
// surface as errors so the caller decides the failure policy.
func (th *Throttler) hit(
	ctx context.Context,
	key string,
	now time.Time,
	config ThrottleConfig,
) (allowed bool, retryAfter time.Duration, err error) {
	state, err := th.store.Update(
		ctx, key,
		func(current ThrottleState) ThrottleState {
			current, allowed, retryAfter = hitWindow(current, now, config)

			return current
		},
	)
	if err != nil {
		return false, 0, err
	}

	if state.Epoch.Equal(now) && th.overCap(ctx) {
		// Reclaim is best effort: a failure only delays cleanup.
		_, _ = th.store.Prune(ctx, now)
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

// overCap reports, best effort, whether stored keys exceed the
// reclaim threshold. A sizing failure skips the reclaim hint.
func (th *Throttler) overCap(ctx context.Context) bool {
	size, err := th.store.Size(ctx)
	if err != nil {
		return false
	}

	return size > throttleBucketLimit
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
			r.Context(), clientIP+"|", time.Now(), th.global,
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
				r.Context(),
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

// ipRange parses one trusted proxy as an exact address or CIDR.
func ipRange(value string) (*net.IPNet, error) {
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}

		return network, nil
	}

	address := net.ParseIP(value)
	if address == nil {
		return nil, &net.AddrError{Err: "invalid IP address", Addr: value}
	}
	if raw := address.To4(); raw != nil {
		return &net.IPNet{IP: raw, Mask: net.CIDRMask(32, 32)}, nil
	}

	return &net.IPNet{IP: address.To16(), Mask: net.CIDRMask(128, 128)}, nil
}

// ForwardedForClientIP builds a Throttler address resolver that honors
// the X-Forwarded-For chain only while the transport peer and every
// intermediate hop between the client and this process fall inside the
// trusted proxy ranges. Proxies append the address they serve, so the
// resolver walks the chain right-to-left past trusted hops and keys on
// the first untrusted address. Requests from any other peer ignore the
// header completely, so spoofed values cannot rotate buckets; with no
// trusted proxy configured, only direct peers resolve.
func ForwardedForClientIP(
	trustedProxies []string,
) (ClientIPFunc, error) {
	ranges := make([]*net.IPNet, 0, len(trustedProxies))
	for _, value := range trustedProxies {
		network, err := ipRange(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, network)
	}

	trusted := func(ip net.IP) bool {
		for _, network := range ranges {
			if network.Contains(ip) {
				return true
			}
		}

		return false
	}

	return func(r *http.Request) string {
		peer := net.ParseIP(hostOnly(r.RemoteAddr))
		if peer == nil {
			return unknownClientIP
		}

		if !trusted(peer) {
			return peer.String()
		}

		return forwardedClientIP(r, trusted)
	}, nil
}

// forwardedClientIP walks a trusted peer's forwarding chain from
// rightmost to leftmost and returns the untrusted client address.
func forwardedClientIP(r *http.Request, trusted func(net.IP) bool) string {
	forwarded := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if forwarded == "" {
		return hostOnly(r.RemoteAddr)
	}

	hops := strings.Split(forwarded, ",")
	for position := len(hops) - 1; position >= 0; position-- {
		hop := strings.TrimSpace(hops[position])
		address := net.ParseIP(hop)
		if address == nil {
			// A trusted proxy wrote this value; key on it as-is so the
			// garbage stays bounded to its source.
			return hop
		}
		if trusted(address) {
			continue
		}

		return address.String()
	}

	// Every reported hop is itself trusted: the real client cannot be
	// derived, so the transport peer stays the key.
	return hostOnly(r.RemoteAddr)
}

// hostOnly returns the host part of an address, dropping the port.
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}
