package main

import (
	"fmt"
	"io"
	"strings"
)

// writeMetrics is the same figures in Prometheus' text format, so Grafana keeps their history and
// can alert on a bot whose traffic jumps. Labelled by bot, not by person: names stay on the page.
func writeMetrics(w io.Writer, bots []bot) {
	type metric struct {
		name, help string
		value      func(b bot) float64
	}
	gauges := []metric{
		{"muninn_bot_up", "1 when the bot answered its /bot-stats", func(b bot) float64 {
			if b.Stats != nil {
				return 1
			}
			return 0
		}},
		{"muninn_bot_users", "People who ever wrote to the bot", func(b bot) float64 { return float64(b.Stats.Users.Total) }},
		{"muninn_bot_users_active_24h", "People who wrote in the last 24 hours", func(b bot) float64 { return float64(b.Stats.Users.Active24h) }},
		{"muninn_bot_users_active_7d", "People who wrote in the last 7 days", func(b bot) float64 { return float64(b.Stats.Users.Active7d) }},
		{"muninn_bot_users_new_24h", "People who wrote for the first time in the last 24 hours", func(b bot) float64 { return float64(b.Stats.Users.New24h) }},
		{"muninn_bot_messages_last_minute", "Messages in the last full minute", func(b bot) float64 {
			pm := b.Stats.Traffic.PerMinute
			if len(pm) < 2 {
				return 0
			}
			return float64(pm[len(pm)-2])
		}},
		{"muninn_bot_messages_24h", "Messages in the last 24 hours", func(b bot) float64 { return float64(b.Stats.Traffic.Messages24h) }},
		{"muninn_bot_rejected_24h", "Requests turned away in the last 24 hours", func(b bot) float64 { return float64(b.Stats.Traffic.Rejected24h) }},
	}
	for i, g := range gauges {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", g.name, g.help, g.name)
		for _, b := range bots {
			if i > 0 && b.Stats == nil {
				continue
			}
			fmt.Fprintf(w, "%s{%s} %g\n", g.name, labels(b), g.value(b))
		}
	}
}

func labels(b bot) string {
	l := []string{`container="` + esc(b.Container) + `"`}
	if b.Stats != nil {
		l = append(l, `platform="`+esc(b.Stats.Platform)+`"`, `bot="`+esc(b.Stats.Bot)+`"`)
	}
	return strings.Join(l, ",")
}

func esc(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}
