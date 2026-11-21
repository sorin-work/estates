package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// limiter is a token bucket per key, kept in memory. Each key may use up to
// burst attempts at once, and gets one attempt back every burst-th part of
// per. Buckets that have filled up again are forgotten, so the map only
// holds keys that were busy recently.
//
// It is per process: running several API instances multiplies the limits.
type limiter struct {
	burst int
	per   time.Duration
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

func newLimiter(burst int, per time.Duration) *limiter {
	return &limiter{burst: burst, per: per, now: time.Now, buckets: map[string]*bucket{}}
}

// refill brings b up to date at now and reports whether it is full.
func (l *limiter) refill(b *bucket, now time.Time) bool {
	rate := float64(l.burst) / float64(l.per)
	b.tokens = min(float64(l.burst), b.tokens+float64(now.Sub(b.at))*rate)
	b.at = now
	return b.tokens >= float64(l.burst)
}

// take uses one attempt for key. When none is left it uses nothing and
// returns how long until the next one.
func (l *limiter) take(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.burst), at: now}
		l.buckets[key] = b
	}
	l.refill(b, now)
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	perToken := l.per / time.Duration(l.burst)
	return false, time.Duration((1 - b.tokens) * float64(perToken))
}

// reset gives key all its attempts back.
func (l *limiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// sweep drops full buckets, at most once per period. The caller holds mu.
func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.per {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if l.refill(b, now) {
			delete(l.buckets, k)
		}
	}
}

func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// Limits on signing in and signing up.
const (
	// loginsPerAddress is how many login attempts one address may make a
	// minute, right or wrong.
	loginsPerAddress = 20
	// failuresPerEmail is how many wrong passwords an email may get in
	// emailWindow. A successful login gives them back.
	failuresPerEmail = 5
	emailWindow      = 15 * time.Minute
	// signupsPerAddress is how many accounts one address may create an hour.
	signupsPerAddress = 10
)

type authLimits struct {
	loginByAddress  *limiter
	loginByEmail    *limiter
	signupByAddress *limiter
}

func newAuthLimits() authLimits {
	return authLimits{
		loginByAddress:  newLimiter(loginsPerAddress, time.Minute),
		loginByEmail:    newLimiter(failuresPerEmail, emailWindow),
		signupByAddress: newLimiter(signupsPerAddress, time.Hour),
	}
}

// errTooManyAttempts answers 429 with a Retry-After header in whole seconds.
func errTooManyAttempts(w http.ResponseWriter, wait time.Duration) error {
	secs := int((wait + time.Second - 1) / time.Second)
	secs = max(secs, 1)
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	return &apiError{Status: http.StatusTooManyRequests, Code: "too_many_attempts",
		Message: "too many attempts, try again in " + waitText(secs)}
}

func waitText(secs int) string {
	if secs < 60 {
		return plural(secs, "second", "seconds")
	}
	return plural((secs+59)/60, "minute", "minutes")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// clientAddr is the address a request came from. X-Forwarded-For is only
// believed when the direct peer is a trusted proxy; then the header is read
// from the right, skipping further trusted proxies, and the first address
// that is not one is the client. Entries further left were written by the
// client itself and could say anything.
func clientAddr(r *http.Request, trusted []netip.Prefix) string {
	peer := parseAddr(r.RemoteAddr)
	if !peer.IsValid() {
		return r.RemoteAddr
	}
	if !isTrusted(peer, trusted) {
		return peer.String()
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		a := parseAddr(strings.TrimSpace(hops[i]))
		if !a.IsValid() {
			break
		}
		if !isTrusted(a, trusted) {
			return a.String()
		}
		peer = a
	}
	return peer.String()
}

// parseAddr reads "ip" or "ip:port".
func parseAddr(s string) netip.Addr {
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap()
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
