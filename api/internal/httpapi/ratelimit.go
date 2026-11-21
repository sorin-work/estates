package httpapi

import (
	"crypto/sha256"
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
// At most maxKeys buckets are kept; past that the one idle the longest is
// dropped, so a flood of new keys costs a bounded amount of memory.
//
// It is per process: running several API instances multiplies the limits.
type limiter struct {
	burst   int
	per     time.Duration
	now     func() time.Time
	maxKeys int

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

func newLimiter(burst int, per time.Duration) *limiter {
	return &limiter{burst: burst, per: per, now: time.Now, maxKeys: defaultMaxKeys, buckets: map[string]*bucket{}}
}

// defaultMaxKeys bounds each limiter's map. A bucket is a few dozen bytes
// plus its key, so this is a few megabytes at most.
const defaultMaxKeys = 50_000

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
	return l.takeFrom(key, l.burst)
}

// takeFrom is take for a key that, if new, starts with only start attempts
// instead of the full burst.
func (l *limiter) takeFrom(key string, start int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		l.makeRoom()
		b = &bucket{tokens: float64(min(start, l.burst)), at: now}
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

// has reports whether key has an attempt left, without using it.
func (l *limiter) has(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		return true
	}
	l.refill(b, l.now())
	return b.tokens >= 1
}

// makeRoom drops the bucket used longest ago when the map is full. The
// caller holds mu.
func (l *limiter) makeRoom() {
	if len(l.buckets) < l.maxKeys {
		return
	}
	var oldest string
	var at time.Time
	for k, b := range l.buckets {
		if oldest == "" || b.at.Before(at) {
			oldest, at = k, b.at
		}
	}
	delete(l.buckets, oldest)
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
	// failuresPerPair is how many wrong passwords one address may try for
	// one email in failureWindow. A right password gives them back.
	failuresPerPair = 5
	// failuresPerEmail is how many wrong passwords an email may get from
	// all addresses together in failureWindow. Past it, an address trying
	// that email for the first time gets one attempt instead of
	// failuresPerPair. It never refuses anyone by itself.
	failuresPerEmail = 20
	failureWindow    = 15 * time.Minute
	// signupsPerAddress is how many accounts one address may create an hour.
	signupsPerAddress = 10

	// Password reset: requests an hour per address and per email and
	// address, reset emails an hour per account whoever asks (more are
	// dropped quietly, so nobody can flood an inbox), and attempts to use
	// a link per address in failureWindow.
	forgotsPerAddress  = 10
	forgotsPerPair     = 3
	resetMailsPerEmail = 3
	resetsPerAddress   = 10
)

type authLimits struct {
	loginByAddress  *limiter
	loginByPair     *limiter
	failuresByEmail *limiter
	signupByAddress *limiter

	forgotByAddress   *limiter
	forgotByPair      *limiter
	resetMailsByEmail *limiter
	resetByAddress    *limiter
}

func newAuthLimits() authLimits {
	return authLimits{
		loginByAddress:  newLimiter(loginsPerAddress, time.Minute),
		loginByPair:     newLimiter(failuresPerPair, failureWindow),
		failuresByEmail: newLimiter(failuresPerEmail, failureWindow),
		signupByAddress: newLimiter(signupsPerAddress, time.Hour),

		forgotByAddress:   newLimiter(forgotsPerAddress, time.Hour),
		forgotByPair:      newLimiter(forgotsPerPair, time.Hour),
		resetMailsByEmail: newLimiter(resetMailsPerEmail, time.Hour),
		resetByAddress:    newLimiter(resetsPerAddress, failureWindow),
	}
}

// limitKey turns parts of a key into a short fixed-size string, so the
// limiters never hold what a client typed.
func limitKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return string(h.Sum(nil)[:16])
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
