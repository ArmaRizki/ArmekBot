package cmdadapter

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/keshon/command"
)

type Adapter struct {
	Cmd Handler
}

func (a *Adapter) Name() string             { return a.Cmd.Name() }
func (a *Adapter) Description() string      { return a.Cmd.Description() }
func (a *Adapter) Group() string            { return a.Cmd.Group() }
func (a *Adapter) Category() string         { return a.Cmd.Category() }
func (a *Adapter) UserPermissions() []int64 { return a.Cmd.UserPermissions() }

func (a *Adapter) Run(ctx context.Context, inv *command.Invocation) error {
	// The command body does not take a context, so the guard's deadline can only be
	// honored here: refuse to start once the slot's context is already done.
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return a.Cmd.Run(inv.Data)
}

func (a *Adapter) SlashDefinition() *discordgo.ApplicationCommand {
	if sp, ok := a.Cmd.(SlashProvider); ok {
		return sp.SlashDefinition()
	}
	return nil
}

func (a *Adapter) ContextDefinition() *discordgo.ApplicationCommand {
	if cp, ok := a.Cmd.(ContextMenuProvider); ok {
		return cp.ContextDefinition()
	}
	return nil
}

func (a *Adapter) ReactionDefinition() string {
	if rp, ok := a.Cmd.(ReactionProvider); ok {
		return rp.ReactionDefinition()
	}
	return ""
}

func (a *Adapter) Component(ctx *ComponentInteractionContext) error {
	if ch, ok := a.Cmd.(ComponentInteractionHandler); ok {
		return ch.Component(ctx)
	}
	return nil
}
