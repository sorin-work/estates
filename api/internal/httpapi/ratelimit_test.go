package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time               { return c.t }
func (c *fakeClock) add(d time.Duration)          { c.t = c.t.Add(d) }
func newFakeClock() *fakeClock                    { return &fakeClock{t: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)} }
func withClock(l *limiter, c *fakeClock) *limiter { l.now = c.now; return l }

func TestLimiterBurstAndRefill(t *testing.T) {
	clock := newFakeClock()
	l := withClock(newLimiter(5, 15*time.Minute), clock)

	for i := range 5 {
		if ok, _ := l.take("a"); !ok {
			t.Fatalf("attempt %d refused", i+1)
		}
	}
	ok, wait := l.take("a")
	if ok || wait != 3*time.Minute {
		t.Fatalf("sixth attempt = %v, %v; want refused for 3m", ok, wait)
	}
	if ok, _ := l.take("b"); !ok {
		t.Error("another key shares the bucket")
	}

	clock.add(2 * time.Minute)
	if ok, wait := l.take("a"); ok || wait != time.Minute {
		t.Fatalf("after 2m = %v, %v; want refused for 1m", ok, wait)
	}
	clock.add(time.Minute)
	if ok, _ := l.take("a"); !ok {
		t.Error("one attempt should be back after 3m")
	}
	if ok, _ := l.take("a"); ok {
		t.Error("only one attempt should be back")
	}

	l.reset("a")
	for i := range 5 {
		if ok, _ := l.take("a"); !ok {
			t.Fatalf("attempt %d after reset refused", i+1)
		}
	}
}

func TestLimiterForgetsIdleKeys(t *testing.T) {
	clock := newFakeClock()
	l := withClock(newLimiter(2, time.Minute), clock)
	l.take("a")
	l.take("b")
	clock.add(30 * time.Second)
	l.take("c")
	l.take("c")
	if n := l.size(); n != 3 {
		t.Fatalf("size = %d, want 3", n)
	}
	// After a full period a and b are full again and swept; c was used
	// up later and is still short of half an attempt.
	clock.add(45 * time.Second)
	l.take("d")
	if n := l.size(); n != 2 {
		t.Errorf("size after sweep = %d, want 2 (c and d)", n)
	}
}

func TestClientAddr(t *testing.T) {
	proxies := []netip.Prefix{netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("10.0.0.1/32")}
	cases := []struct {
		name, remote, forwarded string
		trusted                 []netip.Prefix
		want                    string
	}{
		{"no proxies", "203.0.113.5:4000", "198.51.100.1", nil, "203.0.113.5"},
		{"untrusted peer", "203.0.113.5:4000", "198.51.100.1", proxies, "203.0.113.5"},
		{"trusted peer", "172.18.0.3:4000", "198.51.100.1", proxies, "198.51.100.1"},
		{"spoofed left part", "172.18.0.3:4000", "1.2.3.4, 198.51.100.1", proxies, "198.51.100.1"},
		{"two proxies", "172.18.0.3:4000", "198.51.100.1, 10.0.0.1", proxies, "198.51.100.1"},
		{"no header", "172.18.0.3:4000", "", proxies, "172.18.0.3"},
		{"garbage header", "172.18.0.3:4000", "not-an-ip", proxies, "172.18.0.3"},
		{"ipv6 peer", "[2001:db8::1]:4000", "", proxies, "2001:db8::1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.RemoteAddr = c.remote
			if c.forwarded != "" {
				r.Header.Set("X-Forwarded-For", c.forwarded)
			}
			if got := clientAddr(r, c.trusted); got != c.want {
				t.Errorf("clientAddr = %q, want %q", got, c.want)
			}
		})
	}
}

func TestWaitText(t *testing.T) {
	for secs, want := range map[int]string{1: "1 second", 45: "45 seconds", 60: "1 minute", 61: "2 minutes", 180: "3 minutes"} {
		if got := waitText(secs); got != want {
			t.Errorf("waitText(%d) = %q, want %q", secs, got, want)
		}
	}
	if !strings.Contains(waitText(900), "15 minutes") {
		t.Error("15 minutes")
	}
}

func TestLimiterKeepsAtMostMaxKeys(t *testing.T) {
	clock := newFakeClock()
	l := withClock(newLimiter(5, time.Minute), clock)
	l.maxKeys = 3
	for _, k := range []string{"a", "b", "c"} {
		l.take(k)
		clock.add(time.Second)
	}
	l.take("d")
	if n := l.size(); n != 3 {
		t.Fatalf("size = %d, want 3", n)
	}
	l.mu.Lock()
	_, hasA := l.buckets["a"]
	_, hasD := l.buckets["d"]
	l.mu.Unlock()
	if hasA || !hasD {
		t.Errorf("the oldest key should go: a kept %v, d kept %v", hasA, hasD)
	}
}

func TestLimiterStartAndHas(t *testing.T) {
	clock := newFakeClock()
	l := withClock(newLimiter(5, 15*time.Minute), clock)
	if !l.has("new") {
		t.Error("an unknown key has attempts")
	}
	if ok, _ := l.takeFrom("x", 1); !ok {
		t.Fatal("first attempt refused")
	}
	if ok, _ := l.takeFrom("x", 1); ok {
		t.Error("a key started with one attempt allowed two")
	}
	if l.has("x") {
		t.Error("has reports an attempt left")
	}
}

func TestLimitKey(t *testing.T) {
	if limitKey("ab", "c") == limitKey("a", "bc") {
		t.Error("parts run together")
	}
	if len(limitKey(strings.Repeat("x", 10000))) != 16 {
		t.Error("key is not fixed size")
	}
}
