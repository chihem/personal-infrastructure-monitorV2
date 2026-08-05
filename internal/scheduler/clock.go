package scheduler

import "time"

type clock interface {
	Now() time.Time
	NewTimer(time.Duration) timer
}

type timer interface {
	C() <-chan time.Time
	Stop() bool
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) NewTimer(duration time.Duration) timer {
	return realTimer{Timer: time.NewTimer(duration)}
}

type realTimer struct {
	*time.Timer
}

func (timer realTimer) C() <-chan time.Time {
	return timer.Timer.C
}
