# convertthis

A small, modern web app that converts audio files (**WAV ↔ MP3**) — built in Go,
shipped as a **single binary** with the UI embedded. Conversions run through
`ffmpeg`, so adding more formats later is just a table entry.

```
┌─ Browser (Tailwind UI) ─┐      ┌──────── Go binary ────────┐
│  drag file → choose fmt │ ───▶ │ HTTP → Job → Converter    │ ──▶ ffmpeg
│  download result        │ ◀─── │        ↘ Storage (local)  │
└─────────────────────────┘      └───────────────────────────┘
```

## Requirements

- **Go 1.25+**
- **ffmpeg** on your PATH:
  - Windows: `winget install Gyan.FFmpeg`
  - macOS: `brew install ffmpeg`
  - Linux: `sudo apt install ffmpeg`

## Run

```bash
go run ./cmd/server
# then open http://localhost:8080
```

Build a standalone binary:

```bash
go build -o convertthis ./cmd/server
./convertthis
```

### Config (env vars)

| Var        | Default              | Description                |
| ---------- | -------------------- | -------------------------- |
| `ADDR`     | `:8080`              | Listen address             |
| `WORK_DIR` | `<temp>/convertthis` | Where converted files land |

## Project layout

```
cmd/server/main.go      wiring: choose implementations, start HTTP
internal/convert/       Converter interface + ffmpeg engine + format table
internal/storage/       Storage interface + local filesystem impl
internal/job/           Job model + synchronous in-memory queue
internal/httpapi/       handlers + routing (stdlib ServeMux, no deps)
web/                    embedded frontend (index.html, static/app.js)
```

The three interfaces — `Converter`, `Storage`, `Queue` — are the seams that let
this scale without a rewrite (see **Scaling** below).

## Styling: Tailwind

For zero-friction dev, `index.html` loads Tailwind from the Play CDN. For
production, generate a static stylesheet with the **standalone Tailwind CLI**
(no Node required):

```bash
# download the CLI once (see https://github.com/tailwindlabs/tailwindcss/releases)
./tailwindcss -i web/src/input.css -o web/static/output.css --minify
```

Then in `index.html`: remove the two `cdn.tailwindcss.com` script lines and
uncomment `<link rel="stylesheet" href="/static/output.css" />`. The CSS is
embedded into the binary automatically.

## Docker

```bash
docker build -t convertthis .
docker run -p 8080:8080 convertthis
```

## Scaling path

The MVP is deliberately synchronous and local. Each step below swaps one
implementation behind an existing interface — no rewrite:

1. **Robustness** — file validation, conversion timeouts, temp-file cleanup, rate limiting.
2. **Async** — replace `SyncQueue` with a Redis/asynq queue + worker pool; the UI polls `GET /api/jobs/{id}`.
3. **Distributed** — replace `storage.Local` with S3/GCS; run multiple stateless replicas behind a load balancer.
4. **Product** — accounts, history, batch uploads, more formats (FLAC/OGG/M4A — one registry entry each).
