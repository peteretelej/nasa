# CLI Installation

## Install

```bash
go install github.com/peteretelej/nasa/cmd/nasa@latest
```

## Verify

```bash
nasa --help
```

Expected output:

```text
Usage: nasa <command> [flags]

Commands:
  apod       Astronomy Picture of the Day
  neo        Near-Earth Object Web Service
  donki      Space Weather (DONKI)
  images     NASA Image and Video Library
  ssd        Small-Body Database (SSD/CNEOS)
  eonet      Earth Observatory Natural Event Tracker

Flags:
  --version  Print version and exit
  --help     Show this help message
```

## API Key

Set your API key via environment variable:

```bash
export NASA_API_KEY="your-registered-key"
```

The CLI also accepts `NASAKEY` as a fallback. Without either, it uses NASA's `DEMO_KEY` (30 requests/hour).

Most commands also accept `--api-key` to override per-invocation:

```bash
nasa apod --api-key="your-key"
```

## Output Formats

All commands default to formatted table output. Add `--json` for machine-readable JSON:

```bash
nasa apod --json
nasa neo feed --json
```
