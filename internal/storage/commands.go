package storage

import (
	"slices"

	"github.com/keshon/datastore"
)

// DisableGroup marks a command group disabled for the guild (idempotent).
func (s *Storage) DisableGroup(guildID, group string) error {
	return s.db.Update(func(tx *datastore.Tx) error {
		col := datastore.In(tx, s.settings)
		g := guildSettingsIn(col, guildID)
		if slices.Contains(g.CommandsDisabled, group) {
			return nil
		}
		g.CommandsDisabled = append(g.CommandsDisabled, group)
		return col.Put(g)
	})
}

// EnableGroup re-enables a command group for the guild (idempotent).
func (s *Storage) EnableGroup(guildID, group string) error {
	return s.db.Update(func(tx *datastore.Tx) error {
		col := datastore.In(tx, s.settings)
		g := guildSettingsIn(col, guildID)
		updated := make([]string, 0, len(g.CommandsDisabled))
		for _, existing := range g.CommandsDisabled {
			if existing != group {
				updated = append(updated, existing)
			}
		}
		if len(updated) == len(g.CommandsDisabled) {
			return nil
		}
		g.CommandsDisabled = updated
		return col.Put(g)
	})
}

// guildSettingsIn returns the guild's settings as seen inside a transaction,
// or a blank set for an unknown guild. The returned value is a private copy:
// mutate it and Put it back.
func guildSettingsIn(col *datastore.TxCollection[*GuildSettings], guildID string) *GuildSettings {
	if g, ok := col.Get(guildID); ok {
		return g
	}
	return &GuildSettings{GuildID: guildID}
}

// IsGroupDisabled reports whether a command group is disabled for the guild.
func (s *Storage) IsGroupDisabled(guildID, group string) (bool, error) {
	return slices.Contains(s.guildSettings(guildID).CommandsDisabled, group), nil
}

// DisabledGroups lists the guild's disabled command groups.
func (s *Storage) DisabledGroups(guildID string) ([]string, error) {
	return s.guildSettings(guildID).CommandsDisabled, nil
}
