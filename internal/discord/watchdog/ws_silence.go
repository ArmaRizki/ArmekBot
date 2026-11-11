package watchdog

import (
	"context"
	"time"
)

type WSSilenceMeta struct {
	SinceLastWS      time.Duration
	HeartbeatLatency time.Duration
	Timeout          time.Duration
	// LastWS is the time of the last gateway frame: it changes when the gateway
	// recovers, so callers can tell a new silence episode from a continuing one.
	LastWS time.Time
}

// WSSilence restarts a session when the gateway receive loop appears silent.
//
// Behavior is intentionally simple and mirrors the old inline logic:
// - waits settleDelay before starting checks
// - ticks every tick interval
// - does nothing until tracker reports ready
// - triggers unhealthy when time since last WS exceeds timeout
//
// onUnhealthy reports whether the session is going away: when it returns true the
// watcher stops, otherwise it keeps watching and re-arms on the next tick (the
// session survived, e.g. a grace period or a voice-only restart).
type WSSilence struct {
	tracker     *Tracker
	timeout     time.Duration
	settleDelay time.Duration
	tick        time.Duration

	heartbeatLatency func() time.Duration
	onUnhealthy      func(meta WSSilenceMeta) bool
}

type WSSilenceOptions struct {
	SettleDelay time.Duration
	Tick        time.Duration
}

func NewWSSilence(tracker *Tracker, timeout time.Duration, heartbeatLatency func() time.Duration, onUnhealthy func(meta WSSilenceMeta) bool, opts WSSilenceOptions) *WSSilence {
	if opts.SettleDelay <= 0 {
		opts.SettleDelay = 15 * time.Second
	}
	if opts.Tick <= 0 {
		opts.Tick = 10 * time.Second
	}
	return &WSSilence{
		tracker:          tracker,
		timeout:          timeout,
		settleDelay:      opts.SettleDelay,
		tick:             opts.Tick,
		heartbeatLatency: heartbeatLatency,
		onUnhealthy:      onUnhealthy,
	}
}

func (w *WSSilence) Run(ctx context.Context) {
	if w == nil || w.tracker == nil || w.timeout <= 0 || w.onUnhealthy == nil {
		return
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(w.settleDelay):
	}

	ticker := time.NewTicker(w.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if !w.tracker.IsReady() {
				continue
			}
			since := w.tracker.SinceLastWS(now)
			if since <= w.timeout {
				continue
			}
			lat := time.Duration(0)
			if w.heartbeatLatency != nil {
				lat = w.heartbeatLatency()
			}
			if w.onUnhealthy(WSSilenceMeta{
				SinceLastWS:      since,
				HeartbeatLatency: lat,
				Timeout:          w.timeout,
				LastWS:           w.tracker.LastWSTime(),
			}) {
				return
			}
			// The session survives (grace period, voice-only restart): keep watching
			// and re-arm on the next tick. The silence duration only resets when the
			// gateway delivers a frame again, so this fires once per tick at most.
		}
	}
}
