package discord

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// attachDiscordgoLogger routes discordgo internal logs through zerolog (global hook in discordgo).
func attachDiscordgoLogger(log zerolog.Logger) {
	discordgo.Logger = func(msgL, caller int, format string, a ...interface{}) {
		raw := fmt.Sprintf(format, a...)

		var (
			ev *zerolog.Event
		)
		switch msgL {
		case discordgo.LogError:
			ev = log.Error()
		case discordgo.LogWarning:
			ev = log.Warn()
		case discordgo.LogInformational:
			ev = log.Info()
		case discordgo.LogDebug:
			ev = log.Debug()
		default:
			ev = log.Info()
		}

		// "src" (never "at"): the root logger's Caller() hook owns "at" and writes
		// it at Msg() time, so reusing the name produced a duplicate key.
		// caller+1: discordgo resolves the caller depth inside its msglog, so from
		// inside this closure the real callsite is one frame further up.
		if _, file, line, ok := runtime.Caller(caller + 1); ok {
			ev.Str("src", file+":"+strconv.Itoa(line))
		}

		ev.Msg(raw)
	}
}
