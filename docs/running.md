
# Running ArmekBot

This guide covers both modes:
- Discord bot
- CLI player

---

## Requirements

None beyond the bot itself. Two external binaries are **optional**:

- **FFmpeg** — only the transcode parsers need it (SoundCloud, radio, and the
  `kkdai-link`/`ytdlp-*` fallbacks). YouTube plays by Opus passthrough, so a
  YouTube-first bot needs no FFmpeg at all.
- **yt-dlp** — optional last-resort parser.

Both default to `PATH` and can be pointed elsewhere via `ffmpeg.FFmpegPath` /
`ytdlp.YtdlpPath`.

---

## Discord bot

### Step 1: Create bot

1. Open https://discord.com/developers/applications
2. Create application
3. Go to "Bot"
4. Create bot and copy token

Enable intents:
- Presence
- Server Members
- Message Content

---

### Step 2: Invite bot

Replace YOUR_APPLICATION_ID:

https://discord.com/oauth2/authorize?client_id=YOUR_APPLICATION_ID&scope=bot&permissions=3238912

---

### Step 3: Configure

Create `.env` or export variables:

```env
DISCORD_TOKEN=your-token
````

Optional variables:

| Variable                  | Description                                                | Default                 |
| ------------------------- | ---------------------------------------------------------- | ----------------------- |
| `STORAGE_PATH`            | Directory the datastore owns (write-ahead log + snapshots). Locked to one process. | `./data/store` |
| `INIT_SLASH_COMMANDS`     | Set to `true` to register slash commands on every startup. | `false`                 |
| `DEVELOPER_ID`            | Your Discord user ID for developer-only commands.          | (none)                  |
| `DISCORD_GUILD_BLACKLIST` | Comma-separated guild IDs the bot will leave.              | (none)                  |
| `VOICE_READY_DELAY_MS`    | Delay after joining VC before sending Opus (avoids OP4 race). | `500`                 |
| `WS_SILENCE_TIMEOUT`      | Treat gateway as unhealthy after this long without messages. | `2m`                  |
| `DISCORD_UNHEALTHY_MODE`  | Action on unhealthy: `restart-session`, `restart-voice`, `ignore`. | `restart-session` |
| `DISCORD_UNHEALTHY_GRACE` | In `restart-session`: ignore first N unhealthy signals within window (still invalidates sinks). | `0` |
| `DISCORD_UNHEALTHY_WINDOW`| Window for `DISCORD_UNHEALTHY_GRACE` counting.             | `1m`                    |
| `PLAYER_TRANSPORT_RECOVERY_MODE` | On voice transport failure: `hard` (rejoin VC) or `soft` (reopen stream first, then hard fallback). | `hard` |
| `PLAYER_TRANSPORT_SOFT_ATTEMPTS` | In `soft` mode: how many soft retries before hard fallback. | `1` |
| `CACHE_ENABLED`           | Cache played tracks to disk; later plays (any guild, or `/play <id>`) serve instantly with no extraction. | `false` |
| `CACHE_DIR`               | Directory for cache blobs (wiped on boot when not persistent).           | `./data/cache`          |
| `CACHE_MAX_BYTES`         | Global cache size cap; least-recently-used tracks are evicted past it.   | `2147483648` (2 GiB)    |
| `CACHE_PERSISTENT`        | Keep the cache across restarts (`false` = transient, wiped on boot).     | `true`                  |
| `BUFFER_AHEAD_MS`         | Anti-skip read-ahead depth (ms) masking short source stalls; `0` disables. | `10000`               |
| `DEFAULT_STAY_CONNECTED`  | Keep the bot in the voice channel after the queue ends (24/7 mode). | `false`               |
| `COMMAND_TIMEOUT`         | Hard timeout for a single command execution.               | `30s`                   |
| `COMMAND_PARALLELISM`     | Max number of concurrently running command handlers.       | `16`                    |
| `LOG_LEVEL`               | Log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`. | `info`         |
| `LOG_FILE`                | Optional JSON log file with rotation; empty = stderr only (pretty console). | (none)      |
| `LOG_MAX_SIZE_MB`         | Max size of the log file before it is rotated (MB).        | `10`                    |
| `LOG_MAX_BACKUPS`         | Number of rotated log files to keep.                       | `3`                     |
| `LOG_MAX_AGE_DAYS`        | Max age in days for rotated files (`0` = no age-based deletion). | `0`                |
| `LOG_COMPRESS`            | Gzip the rotated log files.                                | `false`                 |


---

### Step 4: Run

```bash
go build -o armekbot-discord ./cmd/discord
./armekbot-discord
```

---

## CLI player

No Discord setup required.

```bash
go build -o armekbot-cli ./cmd/cli
./armekbot-cli
```

Commands:

* play <url or query>
* next
* stop
* queue
* status
* quit

---

## Docker

See `docker/README.md`