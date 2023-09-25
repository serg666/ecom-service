package httputil

import "time"

type Limiter struct {
	lastTime *time.Time
	counter  uint
	limit    uint
	per      time.Duration
}

func (l *Limiter) Allow() bool {
	now := time.Now()
	if l.lastTime != nil {
		if now.Sub(*l.lastTime) < l.per {
			l.counter++
			if l.counter > l.limit {
				return false
			} else {
				return true
			}
		}
	}

	l.lastTime = &now
	l.counter = 1

	return true
}

func NewLimiter(limit uint, per time.Duration) *Limiter {
	return &Limiter{
		counter: 0,
		limit:   limit,
		per:     per,
	}
}
