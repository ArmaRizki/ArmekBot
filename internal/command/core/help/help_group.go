package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ArmaRizki/ArmekBot/internal/discord/cmdadapter"
	"github.com/keshon/command"
)

func runHelpByGroup() string {
	all := command.DefaultRegistry.GetAll()

	groupMap := make(map[string][]command.Command)
	for _, c := range all {
		meta, _ := command.Root(c).(cmdadapter.Meta)
		group := ""
		if meta != nil {
			group = meta.Group()
		}
		groupMap[group] = append(groupMap[group], c)
	}

	var sortedGroups []string
	for group := range groupMap {
		sortedGroups = append(sortedGroups, group)
	}
	sort.Strings(sortedGroups)

	var sb strings.Builder
	for _, group := range sortedGroups {
		fmt.Fprintf(&sb, "**%s**\n", group)
		cmds := groupMap[group]
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name() < cmds[j].Name() })
		for _, c := range cmds {
			sb.WriteString(formatCommand(c))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
