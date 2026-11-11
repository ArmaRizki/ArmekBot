package discord

import (
	"github.com/ArmaRizki/ArmekBot/internal/discord/watchdog"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) wireSessionHandlers(dg *discordgo.Session, tracker *watchdog.Tracker) {
	b.configureIntents(dg)
	dg.AddHandler(func(s *discordgo.Session, e *discordgo.Event) {
		_ = s
		_ = e
		tracker.MarkWSNow()
	})
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		tracker.MarkReadyNow()
		b.onReady(s, r)
	})
	dg.AddHandler(b.onGuildCreate)
	dg.AddHandler(b.onGuildDelete)
	dg.AddHandler(b.onInteractionCreate)
}
