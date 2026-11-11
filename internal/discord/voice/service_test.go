package voice

import (
	"testing"
	"time"

	"github.com/ArmaRizki/ArmekBot/internal/config"
	"github.com/ArmaRizki/ArmekBot/pkg/music/player"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// The status watcher owns a goroutine that only ends when its stop channel is
// closed: the player's status channel is never closed.
func TestWatchPlayerStatusExitsOnStop(t *testing.T) {
	s := &Service{
		getSession: func() *discordgo.Session { return nil },
		log:        zerolog.Nop(),
	}
	p := player.New(nil, nil)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		s.watchPlayerStatus("guild1", p, stop)
		close(done)
	}()
	close(stop)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watchPlayerStatus did not exit after its stop channel was closed")
	}
}

func TestHandleGuildDeleteClosesStatusWatcher(t *testing.T) {
	s := NewVoiceService(func() *discordgo.Session { return nil }, &config.Config{}, nil, zerolog.Nop())
	if s.GetOrCreatePlayer("guild1") == nil {
		t.Fatal("expected a player for the guild")
	}

	s.mu.RLock()
	stop := s.statusWatchers["guild1"]
	s.mu.RUnlock()
	if stop == nil {
		t.Fatal("expected a status watcher for the guild")
	}

	s.HandleGuildDelete("guild1")

	select {
	case <-stop:
	default:
		t.Fatal("expected the status watcher stop channel to be closed")
	}
}
