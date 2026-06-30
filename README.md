# multiplat-playlist

A CLI music player written in Go for playing links from multiple music platforms in one queue.

Current status: early MVP. YouTube playback works through `yt-dlp` and `mpv`. Spotify track URLs can play through Spotify preview URLs when credentials are configured and a preview is available.

## Why?

Sometimes you want to listen to a song only available on YouTube, but you're vibing on Spotify. Or vice versa. This tool lets you queue up songs from any platform and play them in one unified flow.

## Features

- Play YouTube songs from the command line
- Play Spotify track previews when available
- Queue supported links and play them sequentially
- Control active MPV playback with pause, resume, stop, next, and status commands
- Persist config, queue state, and active playback session metadata
- Planned: full Spotify playback support

## Installation

### From source:

```bash
git clone https://github.com/yourusername/multiplat-playlist.git
cd multiplat-playlist
go build -o mplay ./cmd/mplay
sudo mv mplay /usr/local/bin/
```

### With Go install:

```bash
go install github.com/yourusername/multiplat-playlist/cmd/mplay@latest
```

## Prerequisites

- **mpv** installed for audio playback
- **yt-dlp** for YouTube stream extraction

```bash
# macOS
brew install mpv yt-dlp

# Linux (Debian/Ubuntu)
sudo apt install mpv yt-dlp

# Arch Linux
sudo pacman -S mpv yt-dlp
```

## Usage

Play a single song:
```bash
mplay https://www.youtube.com/watch?v=dQw4w9WgXcQ
mplay spotify:track:[track-id]
```

Queue multiple songs:
```bash
mplay queue add https://youtu.be/[video-id]
mplay queue play
```

Controls:
```bash
mplay pause
mplay resume
mplay next
mplay stop
mplay status
```

### Playback Control Model

The current queue model is foreground-only:

- `mplay queue play` must stay running while the queue is playing.
- The active MPV PID and IPC socket are saved in `~/.config/multiplat-playlist/session.json`.
- `pause`, `resume`, `stop`, `next`, and `status` can be run from another terminal while that session is active.
- `next` stops the active MPV process. The queue advances only if the original `mplay queue play` process is still running and observes MPV exit.
- `next` does not mutate persisted queue state by itself when no active playback session exists.

A daemon/session worker model is planned for future cross-invocation queue ownership.

## Supported Platforms

- **YouTube** - Implemented through `yt-dlp`
- **Spotify** - Track metadata lookup and preview URL playback fallback

### Supported URL Formats

**Spotify:**
- `https://open.spotify.com/track/[track-id]`
- `spotify:track:[track-id]`

**YouTube:**
- `https://www.youtube.com/watch?v=[video-id]`
- `https://youtu.be/[video-id]`

## Configuration

Spotify playback uses the client credentials flow for track metadata and preview URL lookup:
```bash
mplay auth
```

Or manually create `~/.config/multiplat-playlist/config.json`:
```json
{
  "spotify": {
    "client_id": "your_client_id",
    "client_secret": "your_client_secret"
  },
  "player": {
    "backend": "mpv",
    "volume": 100
  }
}
```

Get Spotify credentials from the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).

## Tech Stack

- **Go** - Fast, single-binary distribution, excellent concurrency
- Spotify Web API for track metadata and preview URLs
- yt-dlp for YouTube stream extraction
- mpv for audio playback

## How It Works

1. Parse the URL to detect the platform.
2. For YouTube links, resolve an audio stream with `yt-dlp`.
3. For Spotify links, fetch track metadata and use the preview URL when Spotify provides one.
4. Route the stream to `mpv`.
5. Persist queue state and active MPV session metadata.

## Project Structure

```
multiplat-playlist/
├── cmd/
│   └── mplay/           # Main CLI entry point
├── internal/
│   ├── player/          # Audio player wrapper (mpv)
│   ├── spotify/         # Spotify API client & logic
│   ├── youtube/         # YouTube stream extraction
│   ├── queue/           # Queue management
│   ├── config/          # Configuration handling
│   └── parser/          # URL parsing & detection
├── go.mod
├── go.sum
└── README.md
```

## Limitations

- Spotify playback is preview-only; many tracks do not expose a preview URL
- Full Spotify playback is not implemented yet
- YouTube playback quality depends on available streams
- Terminal-based, no GUI controls
- Requires external dependencies (mpv, yt-dlp)
- Queue playback is foreground-only; there is no daemon that owns queue advancement after `mplay queue play` exits

## Roadmap

- [x] Spotify metadata and preview playback fallback
- [ ] Session/daemon queue worker for cross-invocation queue ownership
- [ ] Playlist URL support
- [ ] Search functionality (play by song name)
- [ ] Interactive TUI mode with progress bar
- [ ] Cross-compilation releases (Linux, macOS, Windows)

## Contributing

Pull requests welcome. For major changes, open an issue first.

## License

MIT
