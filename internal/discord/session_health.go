package discord

import (
	"context"
	"sync"
	"time"

	"github.com/ArmaRizki/ArmekBot/internal/discord/watchdog"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// makeSessionUnhealthyNotifier returns the callback the health watchers use to report an
// unhealthy session. It reports whether the session is going away (true only when the
// disconnected channel is closed); a false return means the session survives and the
// watcher must keep watching.
func (b *Bot) makeSessionUnhealthyNotifier(disconnected chan struct{}) func() bool {
	var restartOnce sync.Once
	var unhealthyMu sync.Mutex
	var unhealthyCount int
	var unhealthyWindowStart time.Time

	invalidateSinks := func() {
		if b.voice != nil {
			b.voice.InvalidateAllSinks()
		}
	}

	return func() bool {
		mode := b.cfg.DiscordUnhealthyMode
		switch mode {
		case "ignore":
			return false
		case "restart-voice":
			invalidateSinks()
			return false
		case "restart-session", "":
		default:
			b.log.Warn().Str("mode", mode).Msg("discord_unhealthy_mode_unknown")
		}

		grace := b.cfg.DiscordUnhealthyGrace
		if grace < 0 {
			grace = 0
		}
		window := b.cfg.DiscordUnhealthyWindow
		if window <= 0 {
			window = time.Minute
		}

		shouldRestart := true
		if grace > 0 {
			now := time.Now()
			unhealthyMu.Lock()
			if unhealthyWindowStart.IsZero() || now.Sub(unhealthyWindowStart) > window {
				unhealthyWindowStart = now
				unhealthyCount = 0
			}
			unhealthyCount++
			if unhealthyCount <= grace {
				shouldRestart = false
			}
			unhealthyMu.Unlock()
		}

		if !shouldRestart {
			invalidateSinks()
			return false
		}

		restartOnce.Do(func() {
			b.log.Warn().Msg("discord_session_unhealthy")
			invalidateSinks()
			close(disconnected)
		})
		// The session is going away (closed now or by an earlier signal).
		return true
	}
}

func (b *Bot) startSessionHealthWatchers(
	sessionCtx context.Context,
	dg *discordgo.Session,
	tracker *watchdog.Tracker,
	notifyUnhealthy func() bool,
) {
	// The silence watcher re-arms every tick while the gateway stays silent, so the
	// warning is logged once per silence episode: a new last-WS timestamp means the
	// gateway recovered and went silent again.
	var silenceMu sync.Mutex
	var lastWS time.Time
	firstInEpisode := func(meta watchdog.WSSilenceMeta) bool {
		silenceMu.Lock()
		defer silenceMu.Unlock()
		first := lastWS.IsZero() || !lastWS.Equal(meta.LastWS)
		lastWS = meta.LastWS
		return first
	}

	go watchdog.NewWSSilence(
		tracker,
		b.cfg.WSSilenceTimeout,
		dg.HeartbeatLatency,
		func(meta watchdog.WSSilenceMeta) bool {
			var ev *zerolog.Event
			if firstInEpisode(meta) {
				ev = b.log.Warn()
			} else {
				ev = b.log.Debug()
			}
			ev.Dur("since_last_ws", meta.SinceLastWS).
				Dur("timeout", meta.Timeout).
				Dur("heartbeat_latency", meta.HeartbeatLatency).
				Msg("gateway_silent")
			return notifyUnhealthy()
		},
		watchdog.WSSilenceOptions{SettleDelay: 15 * time.Second, Tick: 10 * time.Second},
	).Run(sessionCtx)

	go func() {
		select {
		case <-sessionCtx.Done():
			return
		case <-time.After(15 * time.Second):
		}

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		fails := 0

		for {
			select {
			case <-sessionCtx.Done():
				return
			case <-ticker.C:
				lat := dg.HeartbeatLatency()
				if lat < 0 {
					b.log.Debug().Dur("heartbeat_latency", lat).Msg("heartbeat_latency_skipped")
					continue
				}
				if _, err := dg.User("@me"); err != nil {
					fails++
					b.log.Warn().Int("fails", fails).Err(err).Msg("api_probe_failed")
					if fails >= 3 {
						b.log.Warn().Int("fails", fails).Msg("api_probe_threshold")
						notifyUnhealthy()
						return
					}
				} else {
					if fails > 0 {
						b.log.Info().Int("fails", fails).Msg("api_probe_recovered")
					}
					fails = 0
					b.log.Debug().Dur("heartbeat_latency", lat).Msg("heartbeat_latency")
				}
			}
		}
	}()
}
