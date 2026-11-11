package discord

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/ArmaRizki/ArmekBot/internal/config"
	"github.com/ArmaRizki/ArmekBot/internal/discord/cmdlogger"
	"github.com/ArmaRizki/ArmekBot/internal/discord/cmdsync"
	"github.com/ArmaRizki/ArmekBot/internal/discord/execguard"
	"github.com/ArmaRizki/ArmekBot/internal/discord/voice"
	"github.com/ArmaRizki/ArmekBot/internal/storage"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// Bot is the Discord bot. Lifecycle is managed by Run/run; handlers are wired in run.
type Bot struct {
	dg        *discordgo.Session
	storage   *storage.Storage
	slashCmds map[string][]*discordgo.ApplicationCommand
	cfg       *config.Config
	mu        sync.RWMutex
	voice     *voice.Service
	log       zerolog.Logger

	cmdSyncer *cmdsync.Syncer
	cmdLogger *cmdlogger.Logger

	sessionCtx atomic.Value // *sessionCtxHolder
	cmdGuard   atomic.Value // *cmdGuardHolder
}

type sessionCtxHolder struct {
	ctx context.Context
}

type cmdGuardHolder struct {
	g *execguard.Guard
}

var disabledGuard = execguard.New(0, 0)

func (b *Bot) baseSessionContext() context.Context {
	if v := b.sessionCtx.Load(); v != nil {
		if holder, ok := v.(*sessionCtxHolder); ok && holder != nil && holder.ctx != nil {
			return holder.ctx
		}
	}
	return context.Background()
}

func (b *Bot) guard() *execguard.Guard {
	if v := b.cmdGuard.Load(); v != nil {
		if holder, ok := v.(*cmdGuardHolder); ok && holder != nil && holder.g != nil {
			return holder.g
		}
	}
	return disabledGuard
}

func (b *Bot) commandContext() (context.Context, context.CancelFunc) {
	base := b.baseSessionContext()
	return b.guard().Context(base)
}

func (b *Bot) acquireCommandSlot(ctx context.Context) error {
	return b.guard().Acquire(ctx)
}

func (b *Bot) releaseCommandSlot() {
	b.guard().Release()
}

func (b *Bot) isGuildBlacklisted(guildID string) bool {
	return slices.Contains(b.cfg.DiscordGuildBlacklist, guildID)
}

// session returns the current Discord session, or nil before the first session opens.
// It reads b.dg under the lock: RunSession swaps it on every session restart.
func (b *Bot) session() *discordgo.Session {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.dg
}

// syncer returns the current slash-command syncer, or nil before the first session opens.
// It reads b.cmdSyncer under the lock: RunSession swaps it on every session restart.
func (b *Bot) syncer() *cmdsync.Syncer {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cmdSyncer
}
