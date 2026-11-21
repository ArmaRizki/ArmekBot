package watchdog

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// The watcher must keep watching while the session survives (the callback returns
// false) and stop once the callback reports that the session is going away.
func TestWSSilenceKeepsWatchingUntilSessionGoesAway(t *testing.T) {
	tracker := NewTracker()
	tracker.MarkReadyNow()

	var calls int32
	w := NewWSSilence(
		tracker,
		time.Millisecond,
		nil,
		func(WSSilenceMeta) bool {
			return atomic.AddInt32(&calls, 1) >= 3
		},
		WSSilenceOptions{SettleDelay: time.Millisecond, Tick: 2 * time.Millisecond},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not stop after the session went away")
	}

	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 unhealthy signals, got %d", got)
	}
}

// An idle gateway is not a dead one: no events arrive for hours while the
// heartbeat keeps being acknowledged. Reporting that as silence restarted the
// session (and disconnected voice) on every idle minute.
func TestWSSilenceIgnoresIdleGatewayWithAcknowledgedHeartbeat(t *testing.T) {
	tracker := NewTracker()
	tracker.MarkReadyNow()

	var calls int32
	w := NewWSSilence(
		tracker,
		time.Millisecond,
		func() time.Duration { return 250 * time.Millisecond },
		func(WSSilenceMeta) bool {
			atomic.AddInt32(&calls, 1)
			return false
		},
		WSSilenceOptions{SettleDelay: time.Millisecond, Tick: 2 * time.Millisecond},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	w.Run(ctx)

	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected no unhealthy signal while heartbeats are acknowledged, got %d", got)
	}
}

// The silence check must still fire when the heartbeat is not acknowledged: a
// negative latency means our last heartbeat is newer than the last ACK.
func TestWSSilenceReportsSilenceWithoutHeartbeatAck(t *testing.T) {
	tracker := NewTracker()
	tracker.MarkReadyNow()

	var calls int32
	w := NewWSSilence(
		tracker,
		time.Millisecond,
		func() time.Duration { return -2 * time.Minute },
		func(WSSilenceMeta) bool {
			return atomic.AddInt32(&calls, 1) >= 1
		},
		WSSilenceOptions{SettleDelay: time.Millisecond, Tick: 2 * time.Millisecond},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w.Run(ctx)

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected the silence signal, got %d", got)
	}
}
