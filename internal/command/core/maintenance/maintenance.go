package maintenance

import (
	"fmt"

	"github.com/ArmaRizki/ArmekBot/internal/discord/cmdadapter"
	"github.com/ArmaRizki/ArmekBot/internal/discord/reply"

	"github.com/bwmarrin/discordgo"
)

type Maintenance struct{}

func (c *Maintenance) Name() string        { return "maintenance" }
func (c *Maintenance) Description() string { return "Bot maintenance commands" }
func (c *Maintenance) Group() string       { return "core" }
func (c *Maintenance) Category() string    { return "⚙️ Settings" }
func (c *Maintenance) UserPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}

func (c *Maintenance) SlashDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name(),
		Description: c.Description(),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "ping",
				Description: "Check bot latency",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "download-db",
				Description: "Download the current server database as a JSON file",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "status",
				Description: "Retrieve statistics about the guild",
			},
		},
	}
}

func (c *Maintenance) Run(ctx interface{}) error {
	context, ok := ctx.(*cmdadapter.SlashInteractionContext)
	if !ok {
		return nil
	}

	s := context.Session
	e := context.Event
	stor := context.Storage

	options := e.ApplicationCommandData().Options

	if len(options) == 0 {
		return reply.RespondEmbedEphemeral(s, e, &discordgo.MessageEmbed{
			Description: "No subcommand provided.",
		})
	}

	sub := options[0]
	switch sub.Name {
	case "ping":
		return runPing(s, e)
	case "download-db":
		if stor == nil {
			return reply.RespondEmbedEphemeral(s, e, &discordgo.MessageEmbed{
				Description: "Storage is not available.",
			})
		}
		return runDownloadDB(s, e, stor)
	case "status":
		return runStatus(s, e)
	default:
		return reply.RespondEmbedEphemeral(s, e, &discordgo.MessageEmbed{
			Description: fmt.Sprintf("Unknown subcommand: %s", sub.Name),
		})
	}
}
