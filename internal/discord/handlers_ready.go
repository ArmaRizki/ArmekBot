package discord

import (
	"github.com/bwmarrin/discordgo"
)

// onReady fires on every successful connect/reconnect.
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	botInfo, err := s.User("@me")
	if err != nil {
		b.log.Warn().Err(err).Msg("bot_user_fetch_failed")
		return
	}

	// Rejoin the 24/7 voice channels first: the per-guild command sync below
	// makes blocking REST calls and must not delay the audio.
	if b.voice != nil {
		b.voice.RejoinStayConnectedTargets(s)
	}

	for _, g := range r.Guilds {
		if b.isGuildBlacklisted(g.ID) {
			b.log.Info().Str("guild_id", g.ID).Msg("guild_blacklisted_leaving")
			if err := s.GuildLeave(g.ID); err != nil {
				b.log.Error().Str("guild_id", g.ID).Err(err).Msg("guild_leave_failed")
			}
			continue
		}
		if b.cfg.InitSlashCommands {
			if syncer := b.syncer(); syncer != nil {
				if err := syncer.SyncGuildCommands(g.ID); err != nil {
					b.log.Error().Str("guild_id", g.ID).Err(err).Msg("commands_sync_failed")
				}
			}
		}
	}

	b.log.Info().Str("username", botInfo.Username).Msg("discord_ready")
}

// onGuildCreate fires when the bot joins a new guild.
func (b *Bot) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	b.log.Info().Str("guild_id", g.Guild.ID).Str("guild_name", g.Guild.Name).Msg("guild_added")
	if b.isGuildBlacklisted(g.ID) {
		b.log.Info().Str("guild_id", g.ID).Msg("guild_blacklisted_leaving")
		if err := s.GuildLeave(g.ID); err != nil {
			b.log.Error().Str("guild_id", g.ID).Err(err).Msg("guild_leave_failed")
		}
		return
	}
	if b.cfg.InitSlashCommands {
		if syncer := b.syncer(); syncer != nil {
			if err := syncer.SyncGuildCommands(g.ID); err != nil {
				b.log.Error().Str("guild_id", g.ID).Err(err).Msg("commands_sync_failed")
			}
		}
	}
}

// onGuildDelete fires when the bot leaves a guild or is removed.
func (b *Bot) onGuildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	if b.voice != nil {
		b.voice.HandleGuildDelete(g.ID)
	}
	b.log.Info().Str("guild_id", g.ID).Msg("guild_removed")
}
