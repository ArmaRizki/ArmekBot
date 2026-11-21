package sink

import (
	"context"
	"fmt"
	"sync"
	"time"

	musicsink "github.com/ArmaRizki/ArmekBot/pkg/music/sink"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// SessionGetter returns the current Discord session (used so providers stay valid across reconnects).
type SessionGetter func() *discordgo.Session

// DiscordSinkProvider implements sink.Provider for a single guild. target is the voice channel ID.
type DiscordSinkProvider struct {
	getSession       SessionGetter
	guildID          string
	voiceReadyDelay  time.Duration
	log              zerolog.Logger
	mu               sync.Mutex
	vc               *discordgo.VoiceConnection
	currentChannelID string
	// onVoiceLost reports a connection that Discord dropped on its own (kicked,
	// moved, channel deleted). Guarded by mu.
	onVoiceLost func(channelID string)
}

// NewDiscordSinkProvider creates a sink provider for the given session getter and guild.
func NewDiscordSinkProvider(getSession SessionGetter, guildID string, voiceReadyDelay time.Duration, log zerolog.Logger) *DiscordSinkProvider {
	if voiceReadyDelay <= 0 {
		voiceReadyDelay = 500 * time.Millisecond
	}
	return &DiscordSinkProvider{
		getSession:      getSession,
		guildID:         guildID,
		voiceReadyDelay: voiceReadyDelay,
		log:             log.With().Str("component", "sink").Logger(),
	}
}

// SetOnVoiceLost registers the callback that reports a voice connection lost
// without the bot asking for it. Set it once, before the first Sink call.
func (p *DiscordSinkProvider) SetOnVoiceLost(fn func(channelID string)) {
	p.mu.Lock()
	p.onVoiceLost = fn
	p.mu.Unlock()
}

const (
	// VoiceJoinTimeout limits how long we wait for voice connection to become ready (e.g. no permission = no event).
	VoiceJoinTimeout = 15 * time.Second
	// voiceDisconnectTimeout bounds a voice disconnect: the fork waits for the
	// connection to actually die, which never finishes without a deadline.
	voiceDisconnectTimeout = 5 * time.Second
)

// alive reports whether a cached connection can still carry audio. Discord ends a
// voice session with a close code that the app only sees as a closed dead channel,
// so status alone would keep handing out a connection that is already gone.
func alive(vc *discordgo.VoiceConnection) bool {
	if vc == nil {
		return false
	}
	select {
	case <-vc.DeadChannel():
		return false
	default:
		return true
	}
}

// Sink joins the voice channel (or reuses existing) and returns an AudioSink. target must be non-empty.
func (p *DiscordSinkProvider) Sink(target string) (musicsink.AudioSink, error) {
	if target == "" {
		return nil, fmt.Errorf("voice channel ID is required")
	}

	p.mu.Lock()
	if p.currentChannelID == target && alive(p.vc) {
		vc := p.vc
		p.mu.Unlock()
		return &DiscordSink{vc: vc, log: p.log}, nil
	}
	old := p.vc
	p.vc = nil
	p.currentChannelID = ""
	p.mu.Unlock()

	// The join and the disconnect below take seconds: they must never run under
	// p.mu, or ReleaseSink/InvalidateSink stall behind them.
	if old != nil {
		p.disconnect(old, "rejoin")
	}

	dg := p.getSession()
	if dg == nil {
		return nil, fmt.Errorf("no Discord session")
	}
	joinCtx, cancel := context.WithTimeout(context.Background(), VoiceJoinTimeout)
	defer cancel()
	vc, err := dg.ChannelVoiceJoin(joinCtx, p.guildID, target, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel: %w", err)
	}

	// Re-read the shared state under the lock: another goroutine may have joined
	// while we were connecting. The guild keeps exactly one connection, so the
	// winner is reused and only a genuinely different connection is discarded.
	p.mu.Lock()
	if p.currentChannelID == target && alive(p.vc) {
		winner := p.vc
		p.mu.Unlock()
		if winner != vc {
			p.disconnect(vc, "join-race")
		}
		return &DiscordSink{vc: winner, log: p.log}, nil
	}
	p.vc = vc
	p.currentChannelID = target
	p.mu.Unlock()

	go p.watchDeath(vc, target)

	p.log.Info().Str("channel_id", target).Str("guild_id", p.guildID).Msg("voice_joined")

	// Give the connection time to settle — without holding p.mu.
	time.Sleep(p.voiceReadyDelay)

	return &DiscordSink{vc: vc, log: p.log}, nil
}

// watchDeath reports a connection the bot did not release. Nothing else in the
// app notices: a voice drop (close code 4014) never reaches the gateway.
func (p *DiscordSinkProvider) watchDeath(vc *discordgo.VoiceConnection, target string) {
	dead := vc.DeadChannel()
	if dead == nil {
		return
	}
	<-dead

	p.mu.Lock()
	current := p.vc == vc && p.currentChannelID == target
	onLost := p.onVoiceLost
	p.mu.Unlock()

	// A release or a newer join already replaced it: that was intentional.
	if !current {
		return
	}
	if onLost != nil {
		onLost(target)
	}
}

// ReleaseSink disconnects from the voice channel for the given target.
func (p *DiscordSinkProvider) ReleaseSink(target string) {
	p.mu.Lock()
	vc := p.vc
	if vc == nil {
		p.mu.Unlock()
		return
	}
	if target != "" && p.currentChannelID != target {
		p.mu.Unlock()
		return
	}
	p.vc = nil
	p.currentChannelID = ""
	p.mu.Unlock()

	p.disconnect(vc, "release")
}

// InvalidateSink clears the cached VoiceConnection without requiring a target match.
// The next Sink(target) will join again (e.g. after voice WebSocket loss while gateway reconnects).
func (p *DiscordSinkProvider) InvalidateSink() {
	p.mu.Lock()
	vc := p.vc
	p.vc = nil
	p.currentChannelID = ""
	p.mu.Unlock()

	p.disconnect(vc, "invalidate")
}

// disconnect tears down a voice connection with a bounded deadline and outside the lock.
func (p *DiscordSinkProvider) disconnect(vc *discordgo.VoiceConnection, phase string) {
	if vc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), voiceDisconnectTimeout)
	defer cancel()
	if err := vc.Disconnect(ctx); err != nil {
		p.log.Warn().Str("phase", phase).Err(err).Msg("voice_disconnect_failed")
	}
}
