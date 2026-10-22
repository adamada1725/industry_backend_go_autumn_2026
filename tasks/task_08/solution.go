package main

import (
	"math"
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	l := &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}

	if clock != nil {
		l.last = clock.Now()
	}

	return l
}
func (l *Limiter) AllowN(n int) bool {
	if n <= 0 || n > l.burst || l.clock == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	elapsed := now.Sub(l.last).Seconds()

	if elapsed > 0 && l.rate > 0 {
		l.tokens = math.Min(
			float64(l.burst),
			l.tokens+elapsed*l.rate,
		)
		l.last = now
	}

	if float64(n) > l.tokens {
		return false
	}

	l.tokens -= float64(n)
	return true
}
