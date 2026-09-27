package web

import (
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

// Throttler is an in-memory fixed-window rate limiter tracked per client
// IP. Routes may override the global limits with their own windows.
type Throttler struct {
	global    ThrottleConfig
	overrides map[string]ThrottleConfig

	mu      sync.Mutex
	buckets map[string]*throttleWindow
}

// NewThrottler builds a Throttler with global limits and per-route
// overrides keyed by "METHOD /path".
func NewThrottler(
	global ThrottleConfig,
	overrides map[string]ThrottleConfig,
) *Throttler {
	return &Throttler{
		global:    global,
		overrides: overrides,
		buckets:   make(map[string]*throttleWindow),
	}
}

// throttleWindow tracks one fixed window and any active block.
type throttleWindow struct {
	epoch        time.Time
	count        int
	blockedUntil time.Time
}

// hit records one request. It reports whether the request is allowed
// and, when rejected, how long remains until the block or window
// expires.
func (th *Throttler) hit(
	key string,
	now time.Time,
	config ThrottleConfig,
) (allowed bool, retryAfter time.Duration) {
	th.mu.Lock()
	defer th.mu.Unlock()

	window := th.buckets[key]
	if window == nil {
		window = &throttleWindow{}
		th.buckets[key] = window
	}
	if len(th.buckets) > 65536 {
		th.prune(now)
	}

	if now.Before(window.blockedUntil) {
		return false, time.Until(window.blockedUntil)
	}
	if window.epoch.Add(config.TTL).Before(now) {
		window.epoch = now
		window.count = 0
	}

	window.count++
	if window.count > config.Limit {
		window.blockedUntil = now.Add(config.Block)
		return false, config.Block
	}

	return true, 0
}

// prune drops windows that are neither blocked nor inside an open
// counting window. Callers must hold mu.
func (th *Throttler) prune(now time.Time) {
	for key, window := range th.buckets {
		if now.Before(window.blockedUntil) {
			continue
		}
		if now.After(window.epoch) && window.epoch.Add(time.Hour).Before(now) {
			delete(th.buckets, key)
		}
	}
}

// reject writes the pinned 429 body with the Retry-After header.
func rejectThrottled(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set(
		"Retry-After",
		strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))),
	)
	WriteJSON(w, http.StatusTooManyRequests, &Error{
		StatusCode: http.StatusTooManyRequests,
		Message:    throttledMessage,
	})
}

// Middleware returns rate-limiting middleware applying the global
// limits and any route override for the request.
func (th *Throttler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := hostOnly(r.RemoteAddr)

		allowed, retryAfter := th.hit(ip+"|", time.Now(), th.global)
		if !allowed {
			rejectThrottled(w, retryAfter)
			return
		}

		if override, ok := th.overrides[r.Method+" "+r.URL.Path]; ok {
			allowed, retryAfter = th.hit(
				ip+"|"+r.Method+" "+r.URL.Path,
				time.Now(),
				override,
			)
			if !allowed {
				rejectThrottled(w, retryAfter)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// hostOnly returns the host part of an address, dropping the port.
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}
