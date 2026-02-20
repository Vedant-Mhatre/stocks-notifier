<img width="250" src="https://user-images.githubusercontent.com/52707230/214772056-ef465e1e-4d71-47ec-9fb8-24a441e74e51.png" />

[![Release](https://img.shields.io/github/v/release/Vedant-Mhatre/stocks-notifier)](https://github.com/Vedant-Mhatre/stocks-notifier/releases)
[![Release Workflow](https://github.com/Vedant-Mhatre/stocks-notifier/actions/workflows/release.yml/badge.svg)](https://github.com/Vedant-Mhatre/stocks-notifier/actions/workflows/release.yml)

Stocks Notifier is a privacy-first alert tool that tracks real-time stock prices for a list of stocks and sends notifications when your alert condition is met.

No signup. No API key setup. Zero tracking. Data stays local.

If this saves you setup time, consider starring the repo.

For docs, visit [blog.vmhatre.com/stocks-notifier/](https://blog.vmhatre.com/stocks-notifier/).  
Fallback docs: [GitHub repository](https://github.com/Vedant-Mhatre/stocks-notifier).

### Status

This repo is active and working for real-time US tickers via `stockprices.dev`, with an optional delayed fallback for non‑US symbols using Stooq.

Made by [Vedant Mhatre](https://vmhatre.com/).

### Why this

* Privacy-first: runs locally and does not collect your data.
* No API-key friction for normal usage.
* Lightweight: CLI-first with optional local web UI.
* Practical alerts: supports both `below` and `above` targets.

### Tradeoffs

* Real-time feed is US-focused.
* Non-US symbols require delayed fallback (`STOCKS_NOTIFIER_ALLOW_DELAYED=1`).

### Roadmap

* Better UI polish and usability for non-technical users.
* More resilient provider fallback strategy.
* Smarter error handling and retry/backoff tuning.
* Broader test coverage for runtime behavior.

### Data sources

* Real-time (US only): `stockprices.dev` (no signup, public endpoint)
* Delayed fallback: Stooq daily close when `STOCKS_NOTIFIER_ALLOW_DELAYED=1`

### Quick start

* Clone the repo.
* Copy `stocks.sample.json` to `stocks.json`.
* Run from source: `go run . .`
* The `.` argument is the directory containing `stocks.json`.

### Rule format

* Legacy format: `"AAPL": 180` (alerts when price is `below` 180).
* Directional format: `"TSLA": {"threshold": 250, "direction": "above"}`.
* Supported directions: `below`, `above` (default: `below`).

### Data behavior

* Real-time source (US tickers): `stockprices.dev`.
* Non-US or suffixed symbols (for example `.NS`) require delayed fallback.
* Enable delayed fallback: `STOCKS_NOTIFIER_ALLOW_DELAYED=1` (Stooq daily close).
* Alert state is persisted, so repeated alerts are suppressed while condition stays true.
* Optional reminder interval while condition stays true: `STOCKS_NOTIFIER_REMINDER_INTERVAL=2h`.

### Polling controls

* `STOCKS_NOTIFIER_POLL_INTERVAL` (default `10m`)
* `STOCKS_NOTIFIER_POLL_NEAR_INTERVAL` (default `2m`)
* `STOCKS_NOTIFIER_NEAR_THRESHOLD_PERCENT` (default `2`)

### Optional local UI

* Start UI: `go run . . --web`
* Open `http://127.0.0.1:8080`
* Optional bind address: `go run . . --web --addr=0.0.0.0:8080`

### Background run

* `nohup go run . . &` (logs go to `nohup.out`)

### Testing

```bash
go test ./...
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup and contribution guidelines.
