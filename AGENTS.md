# Agent Guidelines

These guidelines are for any future coding agent working on this repository, regardless of tool: Codex, Claude, OpenCode, Cursor, or another agent. Read `README.md`, `MEMORY.md`, and this file before making changes.

## Current Baseline

This project is an early MVP Go CLI for unified Spotify and YouTube playback. The current codebase has local scaffolding for Cobra commands, queue persistence, config persistence, YouTube stream extraction through `yt-dlp`, and MPV playback. Spotify playback is not implemented yet, and playback controls do not work across separate CLI invocations.

Treat `MEMORY.md` as the current status snapshot. Treat `README.md` as product intent, not guaranteed implementation truth.

Important: `.gitignore` anchors the local binaries as `/mplay` and `/multiplat-playlist`, so `cmd/mplay/` is intended to be visible to normal git status/staging.

## Working Principles

- Preserve user changes. The worktree may be dirty; inspect before editing and never revert unrelated changes.
- Keep changes narrow and cohesive. Prefer one feature or bug fix per change set.
- Favor explicit, boring Go over clever abstractions.
- Prefer standard library APIs unless a dependency clearly pays for itself.
- Keep CLI commands thin. Business logic belongs below `cmd/`, primarily in `internal/app` or package-specific internals.
- Keep external process boundaries explicit. Calls to `mpv`, `yt-dlp`, or future `librespot` should be wrapped behind small interfaces that can be tested without running those binaries.
- Return wrapped errors with context using `fmt.Errorf("operation: %w", err)`.
- Avoid logging secrets. Spotify client secrets, passwords, access tokens, refresh tokens, cookies, and command lines containing secrets must not be printed.
- Use context for long-running work, process execution, network requests, and playback loops.

## Architecture Boundaries

### `cmd/mplay`

Responsibilities:
- Define Cobra commands, flags, argument validation, and user-facing command descriptions.
- Create the application object or call focused constructors.
- Pass `cmd.Context()` into long-running operations.
- Print only command-level output that truly belongs in the CLI surface.

Avoid:
- URL parsing logic.
- Queue mutation details.
- Spotify, YouTube, or MPV implementation details.
- Direct calls to `exec.Command` for domain behavior.

### `internal/app`

Responsibilities:
- Coordinate config, queue, platform clients, and player behavior.
- Contain workflow-level behavior such as `PlayURL`, `QueueAdd`, `QueuePlay`, `Pause`, `Resume`, `Stop`, and `Status`.
- Own user-facing workflow errors when multiple packages interact.

Guidance:
- Split construction by need. Non-playback commands such as `auth`, `queue add`, `queue list`, and `queue clear` should not require `mpv`.
- Use dependency injection for tests. Prefer constructors that can accept interfaces for player, queue store, platform clients, and streams.
- Keep `App` from becoming a catch-all. If logic becomes platform-specific, move it to the platform package.

### `internal/player`

Responsibilities:
- Define player interfaces and MPV-backed implementation.
- Manage MPV subprocess lifecycle and IPC communication.
- Provide clear status/control behavior.

Guidance:
- Do not hold locks while waiting for playback to finish if that blocks pause/resume/stop/status.
- Cross-process controls need a deliberate session model. Persisting an MPV socket path, PID, and queue/session metadata is preferable to relying on in-memory fields.
- Cleanup must be idempotent.
- Tests should use a fake player for app-level behavior and small unit tests for command encoding/session metadata.

### `internal/queue`

Responsibilities:
- Manage queue state, current index, and persistence.
- Keep serialization stable and backward-compatible.

Guidance:
- Validate loaded queue state, especially index bounds.
- Prefer explicit methods over direct field exposure.
- Avoid mixing playback control with queue storage. The queue knows order and current item; the app/session layer decides what to play.
- Consider using a store abstraction before adding broad tests, so test files can use temp dirs instead of real home config paths.

### `internal/config`

Responsibilities:
- Load/save application configuration.
- Provide defaults.

Guidance:
- Do not silently discard malformed config.
- Validate values that affect behavior, such as player backend and volume range.
- Keep secrets out of errors and logs.
- Avoid hardcoding `os.UserHomeDir` in code that needs tests; use a path resolver or injected config path when practical.

### `internal/parser`

Responsibilities:
- Detect supported URL types and extract stable IDs.

Guidance:
- Keep parser behavior deterministic and well-tested.
- Add table-driven tests for every supported URL shape before expanding platform support.
- Be conservative: reject ambiguous inputs until a feature explicitly supports them.

### `internal/youtube`

Responsibilities:
- Wrap `yt-dlp` stream URL extraction and, later, metadata extraction.

Guidance:
- Use `exec.CommandContext`.
- Check dependency availability separately from extraction.
- Capture stderr for useful errors, but sanitize if future commands include secrets.
- Stream URLs expire; if caching is added, store expiry metadata and refresh on failure.

### `internal/spotify`

Responsibilities:
- Implement Spotify API integration and future Spotify playback support.

Guidance:
- Start with minimal client credentials flow and track metadata lookup.
- Treat preview URL playback as a fallback path, not full Spotify playback.
- If full playback uses `librespot`, isolate it from Web API concerns.
- Handle rate limits and token expiry explicitly.
- Keep auth/token storage separate from track fetching and playback.

## Expansion Guidelines

### Adding a New Platform

1. Add parser support and table-driven parser tests.
2. Add a platform package or client with a small interface for metadata and stream resolution.
3. Wire the platform through `internal/app`.
4. Add app tests using fake platform clients.
5. Update `README.md` and `MEMORY.md` with the actual status.

Do not add platform-specific logic directly to Cobra commands.

### Adding Playback Controls

Decide and document the control model first:

- Foreground-only model: playback blocks the command and controls are limited to signals/keyboard handling in the same process.
- Session/daemon model: playback persists enough state for later CLI invocations to send IPC commands.

For the intended CLI UX, the session/daemon model is likely the correct direction. It should persist enough state to find the active MPV socket, verify the process is alive, and coordinate queue advancement.

### Adding Configuration

- Add fields with JSON tags.
- Provide defaults for missing fields.
- Validate values on load or app construction.
- Preserve backward compatibility with older config files.
- Update config reference docs when behavior is real.

### Adding Dependencies

- Prefer no new dependency unless it avoids substantial complexity.
- Keep dependencies direct in `go.mod` when imported by repo code.
- After adding a dependency, run `go mod tidy`.
- Do not add large frameworks for small CLI workflows.

## Maintainability Practices

- Use table-driven tests for parsing, config defaults, queue transitions, and command behavior.
- Keep package APIs small. Export only what another package actually needs.
- Use interfaces at process/network boundaries, not everywhere.
- Keep data models stable and document changes when persistence format changes.
- Avoid global mutable state. If unavoidable for CLI wiring, keep it at the edge.
- Keep user-facing errors concise; keep internal wrapped errors useful for debugging.
- Separate dependency availability checks from command execution where possible.
- Prefer deterministic tests over tests that depend on real `mpv`, `yt-dlp`, Spotify, or YouTube.
- Update `MEMORY.md` whenever implementation status materially changes.
- Update `README.md` only for behavior that is implemented or clearly labeled as planned.

## Testing Strategy

### Minimum Before Any Functional Change

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./...
GOCACHE=$PWD/.cache/go-build go build ./...
```

Using a workspace-local `GOCACHE` avoids sandbox failures from Go trying to write to the user cache.

### Unit Tests To Add First

- `internal/parser`: Spotify track URLs, Spotify URI form, YouTube watch URLs, youtu.be URLs, invalid URLs.
- `internal/queue`: add, next, current, clear, save/load, corrupted JSON, out-of-range saved index.
- `internal/config`: missing file defaults, malformed JSON error, save/load round trip, validation when added.
- `internal/app`: command workflows using fake queue/player/platform dependencies.

### Integration Tests

Use integration tests sparingly and gate them behind environment variables. They should not run by default in `go test ./...`.

Suggested gates:

- `MPLAY_INTEGRATION_MPV=1` for real MPV tests.
- `MPLAY_INTEGRATION_YTDLP=1` for real `yt-dlp` tests.
- `MPLAY_INTEGRATION_SPOTIFY=1` for Spotify Web API tests.

Integration tests must skip cleanly when dependencies or credentials are missing.

### External Process Tests

Avoid requiring real external binaries in unit tests. Prefer fake command runners or interfaces. When testing process wrappers:

- Use context cancellation tests.
- Test command construction separately from process execution.
- Capture stdout/stderr behavior deterministically.
- Use temp dirs for socket/session files.

### Filesystem Tests

- Use `t.TempDir()`.
- Do not write to the real user home directory.
- Do not depend on existing user config or queue files.
- If code currently hardcodes `os.UserHomeDir`, refactor toward injectable paths before writing broad tests.

## Documentation Expectations

After a meaningful change:

- Update `MEMORY.md` with the real implementation status.
- Update `README.md` if user-visible behavior changed.
- Mention new environment variables, config fields, or external dependency requirements.

Documentation should distinguish clearly between:

- implemented behavior,
- known limitations,
- planned work.

## Review Checklist

Before handing work back:

1. Inspect `git status --short` and verify only intended files changed.
2. Run `gofmt` on edited Go files.
3. Run `GOCACHE=$PWD/.cache/go-build go test ./...`.
4. Run `GOCACHE=$PWD/.cache/go-build go build ./...`.
5. For CLI changes, run the relevant command help or dry-run path when possible.
6. Check that no secrets, local absolute user config files, generated binaries, or cache files are included.
7. Update `MEMORY.md` if the current status changed.

## Known Current Priorities

1. Fix MPV locking so in-memory controls cannot block behind `Play` waiting for process exit.
2. Tighten queue/session behavior for `next`, stale sessions, and queue advancement semantics.
3. Implement minimal Spotify metadata and preview playback.
4. Add focused unit tests around parser, queue, config, and app workflows.
