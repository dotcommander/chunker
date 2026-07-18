# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/).

## [v1.2.0] - 2026-07-18

First public release under `github.com/dotcommander/chunker`. Chunking is
delegated to [`github.com/dotcommander/reliquary`](https://github.com/dotcommander/reliquary).

### Changed
- Module path renamed from `chunker` to `github.com/dotcommander/chunker` so the
  tool installs via `go install github.com/dotcommander/chunker/cmd/chunker@latest`.
- Unknown `strategy` and `token_encoding` values are now rejected with a clear
  error instead of being silently remapped to defaults. **Behavior change** for
  callers that relied on the silent-default leniency.
- `total_tokens` metadata is computed from the original request text (consistent
  with `total_chars`); previously it summed per-chunk counts and double-counted
  the overlap region for `token_based` chunking.
- `start_char` / `end_char` offsets are derived from the library's authoritative
  byte spans rather than re-derived by substring search.

### Fixed
- `http.Server` now sets `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` /
  `IdleTimeout` (Slowloris defense); previously only the handler-level timeout was set.
- The `files` subcommand now bounds per-file reads (100 MiB cap, matching stdin),
  preventing memory exhaustion on large glob matches. The HTTP `/chunk` body is
  bounded at 10 MiB.
- Corrected the `cl100k_base` encoding comment (GPT-3.5/GPT-4, not gpt-5).
- Stopped git-ignoring the `cmd/chunker/` source directory — a bare `chunker`
  ignore pattern had hidden 9 production files (including the embedded config)
  from version control; a fresh clone previously would not build.

### Prior versions
v1.0.0 and v1.1.0 predate this changelog; see `git log` for their history.
