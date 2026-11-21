package discordgo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// newTestSession returns a session that can run ChannelVoiceJoin without a
// gateway: any attempt to send the voice state update fails fast.
func newTestSession() *Session {
	return &Session{VoiceConnections: map[string]*VoiceConnection{}}
}

// A live connection to the requested channel must be returned as is. Discord
// sends no VOICE_SERVER_UPDATE for a repeated join, so waiting for Ready would
// block until the caller's deadline and report a join failure for a connection
// that is already playing audio.
func TestChannelVoiceJoinReusesLiveConnection(t *testing.T) {
	s := newTestSession()
	live := &VoiceConnection{
		Cond:      sync.NewCond(&sync.Mutex{}),
		dead:      make(chan struct{}),
		Status:    VoiceConnectionStatusReady,
		ChannelID: "chan-a",
		GuildID:   "guild",
	}
	s.VoiceConnections["guild"] = live

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := s.ChannelVoiceJoin(ctx, "guild", "chan-a", false, true)
	if err != nil {
		t.Fatalf("join on a live connection failed: %v", err)
	}
	if got != live {
		t.Fatalf("expected the live connection back, got %p", got)
	}
	if live.Status != VoiceConnectionStatusReady {
		t.Fatalf("live connection status was clobbered: %v", live.Status)
	}
}

// A connection that lives in another channel is not a match: the join must go
// through the gateway (here it fails at the missing websocket, which proves the
// short-circuit did not fire) and must not leave the connection half-reset.
func TestChannelVoiceJoinDoesNotReuseOtherChannel(t *testing.T) {
	s := newTestSession()
	live := &VoiceConnection{
		Cond:      sync.NewCond(&sync.Mutex{}),
		dead:      make(chan struct{}),
		Status:    VoiceConnectionStatusReady,
		ChannelID: "chan-a",
		GuildID:   "guild",
	}
	s.VoiceConnections["guild"] = live

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := s.ChannelVoiceJoin(ctx, "guild", "chan-b", false, true); !errors.Is(err, ErrWSNotFound) {
		t.Fatalf("expected ErrWSNotFound, got %v", err)
	}
	if live.Status != VoiceConnectionStatusReady {
		t.Fatalf("status of a live connection must survive a failed join, got %v", live.Status)
	}
}

// A fresh connection has to wait for the voice handshake: returning early would
// hand out a connection that cannot send audio yet.
func TestChannelVoiceJoinWaitsForFreshConnection(t *testing.T) {
	s := newTestSession()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// No gateway: the update fails before any wait, so the call must not hang.
	if _, err := s.ChannelVoiceJoin(ctx, "guild", "chan-a", false, true); !errors.Is(err, ErrWSNotFound) {
		t.Fatalf("expected ErrWSNotFound, got %v", err)
	}
}

// A dead connection is not reused even when it still names the requested channel:
// Discord ended that session, so the join has to start a new one.
func TestChannelVoiceJoinRejoinsDeadConnection(t *testing.T) {
	s := newTestSession()
	dead := make(chan struct{})
	close(dead)
	s.VoiceConnections["guild"] = &VoiceConnection{
		Cond:      sync.NewCond(&sync.Mutex{}),
		dead:      dead,
		Dead:      dead,
		Status:    VoiceConnectionStatusDead,
		ChannelID: "chan-a",
		GuildID:   "guild",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if _, err := s.ChannelVoiceJoin(ctx, "guild", "chan-a", false, true); !errors.Is(err, ErrWSNotFound) {
		t.Fatalf("expected a fresh join attempt (ErrWSNotFound), got %v", err)
	}
}
