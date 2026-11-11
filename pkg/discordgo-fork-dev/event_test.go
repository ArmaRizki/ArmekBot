package discordgo

import (
	"testing"
	"time"
)

// Open() holds the session lock for the whole gateway handshake, and the
// READY event is dispatched inside that handshake. onReady must therefore never
// take the session lock itself, or the gateway deadlocks before it is ready.
func TestOnReadyDoesNotBlockOnSessionLock(t *testing.T) {
	s := &Session{}

	s.Lock() // Open() holds this while the handshake dispatches READY
	defer s.Unlock()

	done := make(chan struct{})
	go func() {
		s.onReady(&Ready{SessionID: "sid", ResumeGatewayURL: "wss://resume.example"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("onReady blocked on the session lock: the gateway handshake would deadlock")
	}

	sessionID, resumeURL := s.sessionState()
	if sessionID != "sid" || resumeURL != "wss://resume.example" {
		t.Fatalf("session state not stored: %q %q", sessionID, resumeURL)
	}
}

func TestClearSessionState(t *testing.T) {
	s := &Session{}
	s.sessionMu.Lock()
	s.sessionID = "sid"
	s.resumeGatewayURL = "wss://resume.example"
	s.sessionMu.Unlock()

	s.clearSessionState()

	sessionID, resumeURL := s.sessionState()
	if sessionID != "" || resumeURL != "" {
		t.Fatalf("session state not cleared: %q %q", sessionID, resumeURL)
	}
}
