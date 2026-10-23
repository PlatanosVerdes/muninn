# What it shows

Per bot: its profile photo, users (total, active today and this week, new today), actions per minute over the last
hour, actions and rejected requests over the last day, the bot's own figures, and everyone who
used it, with how much and when they last did. An action is a written message or a button tap.

A rejected request is one the bot turned away, by its rate limit or as abuse: a climb in those, or
in actions per minute, is what someone flooding a bot looks like.

A Telegram bot's photo comes from its public `t.me/<bot>` page, cached for 12 hours and served by
muninn at `/api/avatar/telegram/<bot>`: the Bot API only hands out a photo behind a URL that carries
the token. A bot without a photo shows its initial.
