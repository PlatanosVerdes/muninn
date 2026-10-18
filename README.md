# muninn

A page that watches the bots: who uses each one, how much traffic reaches it, and whether someone is
trying to flood it. Each bot reports its own figures; muninn finds the bots by a label and only reads.

```
 torre ──/bot-stats──┐
 wallabot ─/bot-stats─┼── muninn ── page (muninn.<domain>)
 pol bot ─/bot-stats──┘      └──── /metrics ── Prometheus ── Grafana history and alerts
```

## Why the name

Odin keeps two ravens, Huginn and Muninn, who fly over the world every day and come back to tell him
what they saw. Huginn is already the exit check next to [sleipnir](https://github.com/PlatanosVerdes/sleipnir);
muninn is the other one, and what it brings back is who has been talking to the bots.

## How a bot shows up

Its container carries one label, the URL of its `/bot-stats` document:

```yaml
labels:
  muninn.stats: "http://torre:9180/bot-stats"
```

muninn lists the containers with that label through
[docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy) (containers only, read-only)
and asks each running one for its document, every time the page refreshes. A stopped container still
shows, as stopped; one whose document does not answer shows why.

The document's format is in [gonana's README](https://github.com/PlatanosVerdes/gonana#botstats-the-bot-stats-contract),
and Go bots get it from `gonana/botstats`. Any bot that serves it shows up, Discord included: the
`platform` field says which.

**The document holds names**, so a bot serves it on a port of its own that is never published, and
muninn reaches it over the Docker network. muninn's own page shows them too: keep it behind the
tailnet.

## What it shows

Per bot: users (total, active today and this week, new today), messages per minute over the last
hour, messages and rejected requests over the last day, the bot's own figures, and everyone who
wrote, with how much and when they last did.

A rejected request is one the bot turned away, by its rate limit or as abuse: a climb in those, or
in messages per minute, is what someone flooding a bot looks like.

## Prometheus

`/metrics` has the same figures as gauges, labelled by container, platform and bot, never by person:
`muninn_bot_up`, `muninn_bot_users`, `muninn_bot_users_active_24h`, `muninn_bot_users_active_7d`,
`muninn_bot_users_new_24h`, `muninn_bot_messages_last_minute`, `muninn_bot_messages_24h`,
`muninn_bot_rejected_24h`.

## Settings

| Variable | Default | |
| :--- | :--- | :--- |
| `DOCKER_HOST_URL` | `http://docker-socket-proxy:2375` | The Docker API, read-only |
| `LISTEN_ADDR` | `:8095` | Page, API and metrics |
| `MUNINN_NAME` | empty | The machine, shown next to the title |

## What it does not do

It never talks to a bot platform, holds no token, and sends nothing. Traffic counts live in each bot
and start over when that bot restarts; the people are kept by the bot on disk.
