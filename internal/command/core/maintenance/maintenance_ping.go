package maintenance

import (
	"fmt"

	"github.com/ArmaRizki/ArmekBot/internal/discord/reply"
	"github.com/bwmarrin/discordgo"
)

func runPing(s *discordgo.Session, e *discordgo.InteractionCreate) error {

	latency := s.HeartbeatLatency().Milliseconds()
	return reply.RespondEmbedEphemeral(s, e, &discordgo.MessageEmbed{
		Title:       "Pong! 🏓",
		Description: fmt.Sprintf("Latency: %dms", latency),
		Color:       reply.EmbedColor,
	})
}
