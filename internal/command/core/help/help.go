package help

import (
	"github.com/ArmaRizki/ArmekBot/internal/discord/cmdadapter"
	"github.com/ArmaRizki/ArmekBot/internal/discord/reply"
	"github.com/keshon/buildinfo"
	"github.com/keshon/command"

	"github.com/bwmarrin/discordgo"
)

// formatCommand renders a command with its subcommands. Registry entries are
// middleware-wrapped, so the root must be unwrapped before the slash
// definition can be read.
func formatCommand(c command.Command) string {
	return cmdadapter.FormatCommandWithSubcommands(command.Root(c))
}

type Help struct{}

func (c *Help) Name() string        { return "help" }
func (c *Help) Description() string { return "Get a list of available commands" }
func (c *Help) Group() string       { return "core" }
func (c *Help) Category() string    { return "🕯️ Information" }
func (c *Help) UserPermissions() []int64 {
	return []int64{}
}

func (c *Help) SlashDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name(),
		Description: c.Description(),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "category",
				Description: "View commands grouped by category",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "group",
				Description: "View commands grouped by group",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "flat",
				Description: "View all commands as a flat list",
			},
		},
	}
}

func (c *Help) Run(ctx interface{}) error {
	context, ok := ctx.(*cmdadapter.SlashInteractionContext)
	if !ok {
		return nil
	}

	session := context.Session
	event := context.Event

	if err := reply.RespondDeferredEphemeral(session, event); err != nil {
		context.AppLog.Error().Err(err).Msg("help_defer_failed")
		return err
	}

	data := event.ApplicationCommandData()
	if len(data.Options) == 0 {
		return reply.FollowupEmbedEphemeral(session, event, &discordgo.MessageEmbed{
			Description: "No subcommand provided. Use `category`, `group`, or `flat`.",
		})
	}

	var output string
	switch data.Options[0].Name {
	case "group":
		output = runHelpByGroup()
	case "flat":
		output = runHelpFlat()
	default:
		output = runHelpByCategory()
	}

	info := buildinfo.Get()
	embed := &discordgo.MessageEmbed{
		Title:       info.Project + " Help",
		Description: output,
		Color:       reply.EmbedColor,
	}

	return reply.FollowupEmbedEphemeral(session, event, embed)
}
