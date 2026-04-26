# j_ai_trade — Telegram Trading Advisor Bot

Conversational trading advisor for **gold scalping (XAUUSDT)** delivered over Telegram. Each user message triggers a fresh fetch of multi-timeframe market data + economic-calendar context, which is fed to DeepSeek; the LLM decides whether to emit a `BUY`/`SELL` card with `entry / SL / TP / lot` or recommend waiting.

## What it does

- **Reactive advice** — natural-language chat. M1 entry, M5 confirm, H1/H4 trend.
- **News awareness** — fetches the ForexFactory weekly calendar; blocks signal generation in the T-15…T+30 window around CPI/FOMC/NFP and warns during pre-event / recovery zones.
- **Proactive alerts** — push notifications at T-30 / T-15 / T-5 before high-impact USD events. Per-chat opt-in (`/alerts on|off`, default on).
- **Trade card persistence** — decisions emitted as a `agent_decision` row in Firestore (or in-memory if Firebase isn't configured).
- **Risk-sized lot** — automatic position sizing so hitting SL costs `ADVISOR_RISK_PCT %` of `ADVISOR_ACCOUNT_USDT`. Disable by setting either to `0`.

## Architecture

```
Telegram  ──►  ChatHandler  ──►  DeepSeek (streamed reply)
                   │
                   ├──►  MarketAnalyzer ──►  Binance REST (M1/M5/H1/H4 klines)
                   ├──►  News Calendar ───►  ForexFactory XML feed
                   ├──►  AlertWorker (1-min ticker → proactive pushes)
                   ├──►  SessionStore  (in-memory; 7-day TTL)
                   └──►  DecisionStore (Firestore | in-memory fallback)
```

Clean Architecture: every dependency is interface-shaped under `modules/advisor/biz/`. Concrete adapters live alongside (`provider/deepseek`, `transport/telegram`, `storage/memory`, `brokers/binance`).

Key packages:

| Path | Role |
| --- | --- |
| `main.go` | Composition root + health server. |
| `modules/advisor/biz/` | Domain logic (chat handler, prompt builder, decision parser). |
| `modules/advisor/biz/market/` | Symbol/TF resolution, indicator pipeline, digest renderer. |
| `modules/advisor/biz/market/news/` | Calendar feed, gate (reactive blackout), alert worker (proactive pushes). |
| `modules/advisor/provider/deepseek/` | Streaming LLM client. |
| `modules/advisor/transport/telegram/` | Long-poll adapter + edit-in-place message bubble. |
| `modules/agent_decision/` | Decision storage interface + Firestore/memory backends. |
| `brokers/binance/` | REST kline fetcher. |
| `trading/indicators/` | EMA / RSI / ATR / ADX implementations used by the digest. |

## Quickstart

Requires Go ≥ 1.25.

```bash
git clone https://github.com/<you>/j_ai_trade.git
cd j_ai_trade
cp .env.example .env
# Fill J_AI_TRADE_ADVISOR + DEEP_SEEK_API_KEY at minimum, then:
go run .
```

The process boots the Telegram long-poll loop and a Gin health server on `$PORT` (default `80`) for platform readiness probes.

## Environment variables

| Var | Required | Purpose |
| --- | --- | --- |
| `J_AI_TRADE_ADVISOR` | ✅ | Telegram bot token (separate from the cron signal bot). |
| `DEEP_SEEK_API_KEY` | ✅ | DeepSeek API key. |
| `ADVISOR_ALLOWED_USER_IDS` | recommended in prod | Comma-separated Telegram user IDs allowlisted. Empty ⇒ allow everyone (dev only). |
| `SERVICE_ACCOUNT_FIREBASE_BASE_64` | optional | Base64'd service-account JSON. Unset ⇒ decisions persist in-memory. |
| `DEEP_SEEK_BASE_URL` | optional | Override LLM endpoint (proxies / compat hosts). |
| `DEEP_SEEK_MODEL` | optional | `deepseek-chat` (default, fast) or `deepseek-reasoner` (slower, deeper). |
| `ADVISOR_ACCOUNT_USDT` | optional | Notional account size for risk sizing (default 1000). `0` disables sizing. |
| `ADVISOR_RISK_PCT` | optional | % of account to lose if SL hit (default 0.5). `0` disables sizing. |
| `ENV` | optional | `PROD` toggles release-mode Gin and skips `.env` loading. |
| `PORT` | optional | Health server port (default `80`). |

## Bot commands

| Command | Behaviour |
| --- | --- |
| `/start` | Welcome message (also auto-fires once per chat). |
| `/reset` | Wipes session memory for the current chat. |
| `/help` | Shows the command list. |
| `/analyze [SYMBOL] [TF]` | Force a fresh analysis (defaults: `XAUUSDT`, M5/H1 confluence). |
| `/alerts on \| off` | Toggle proactive news pushes for the chat. Default on. |

Bare messages route through the same analyzer; the bot decides whether market data is needed based on the text and pinned `LastSymbol`.

## Tests

```bash
go test ./...
```

Coverage focuses on the riskiest layers: news gate window logic, alert worker tier/dedupe, calendar refresh + backoff, decision parser & risk sizing, and chat-handler critical paths (commands, panic recovery, decision extraction, stream errors).

## Design notes

- **Graceful degradation everywhere.** Binance down ⇒ chat-only fallback. ForexFactory feed down ⇒ no news line, no alerts, but the bot keeps replying. Firestore down ⇒ in-memory decisions.
- **Per-message context budget**: 90 s. The LLM stream and Binance fetch share it via `context.WithTimeout`.
- **News gate** ranks states `active > pre > recovery > none` and is rendered as a one-line directive in the `[MARKET_DATA]` block; the LLM applies the blackout reactively.
- **Alert worker** uses single-flight refresh (no pile-on) and per-(eventID, tier, chatID) dedupe so a 1-minute scan tick can't double-push.
- **Risk sizing** owns lot size only — entry/SL/TP come entirely from the LLM's structural read of the chart.
