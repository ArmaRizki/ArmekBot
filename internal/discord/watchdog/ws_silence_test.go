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
