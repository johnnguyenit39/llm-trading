# Advisor Module — Engineering Context

> Drop this file into any prompt to give the agent full context on the advisor
> module. Keep it short, concrete, and up to date.

## 1. Purpose (one paragraph)

The **advisor** module is a conversational Telegram bot — a trader-buddy
chat companion — that lives inside the existing `j_ai_trade` Go binary.
Users DM a separate bot, the backend long-polls Telegram, streams replies
from **DeepSeek**, and edits the Telegram bubble progressively so the UX
feels like texting a person.

**Phase 1 (shipped):** chat-only; bot explicitly declines to invent
numbers.

**Phase 2 (shipped):** on-demand technical analysis. When the user asks
for a concrete signal ("XAU giờ buy hay sell?") the backend fetches
candles from Binance, runs the canonical 4-strategy ensemble + indicator
suite, renders a compact `[MARKET_DATA]` digest, and injects it as an
extra user-role turn before the LLM call. The bot explains the rule
engine's verdict in natural language. Same logic is triggered explicitly
by `/analyze SYMBOL [TF]`.

**Phase 3 (roadmap, §12):** proactive push, news digest, cost tracking.

The existing cron-based signal broadcaster runs in parallel under a
different bot token and now shares the ensemble/market-data factories
with the advisor (see `trading/ensembles` + `trading/marketdata`).

## 2. Who orchestrates what (IMPORTANT)

- **Backend (this repo) = the brain.** It holds the DeepSeek API key and
  the Telegram bot token, long-polls Telegram, calls DeepSeek, and edits
  Telegram messages to stream tokens. No external relay, no OpenClaw.
- **DeepSeek = stateless reasoning engine.** It streams tokens back over
  SSE; it does NOT call back into our backend.
- **Telegram Bot API = transport only.** Long-polling (`getUpdates`) for
  ingress, `sendMessage` / `editMessageText` / `sendChatAction` for egress.

If a future decision is to flip any of this (function-calling DeepSeek,
webhook mode instead of polling, a relay in front), update this doc first.

## 3. End-to-end data flow

```mermaid
flowchart LR
  U["User DM"] --> PLAT["Chat platform<br/>(Telegram today)"]
  PLAT <-->|"platform-specific<br/>(long-poll / webhook / ...)"| TX["transport/*<br/>adapter"]

  subgraph repo ["j_ai_trade (Go binary)"]
    TX -->|"biz.IncomingMessage"| CH["biz/chat_handler.go<br/>(depends only on interfaces)"]
    CH --> F["biz/user_filter.go<br/>(DM + allowlist)"]
    F --> G["biz/greeter.go<br/>(welcome once per chat)"]
    G --> S["biz.SessionStore<br/>(storage/redis today)"]
    S --> MA["biz.MarketAnalyzer<br/>(biz/market today, optional)"]
    MA -.->|"intent hit: fetch candles"| BN["marketdata.BinanceFetcher"]
    MA -.->|"run ensemble"| ENS["ensembles.DefaultEnsembleFor"]
    MA --> P["biz/prompt_builder.go<br/>system prompt + history +<br/>[MARKET_DATA] blob"]
    P --> LLM["biz.LLMProvider<br/>(provider/deepseek today)<br/>POST + SSE stream"]
    LLM -->|"chunk string channel"| BUB["biz.MessageBubble<br/>(throttled edit-in-place)"]
    BUB --> TX
    CH -.->|"KeepTyping via transport"| TX
    CH -.->|"Append Turn"| S
  end
```

Key invariants:

- One Telegram update -> one goroutine in `ChatHandler.handleMessage`. Slow
  DeepSeek replies for user A never block user B.
- `editMessageText` is throttled to one call per ~500ms per chat. The
  first `sendMessage` call is subject to the same window so the opening
  paint carries substantive text instead of a 1–2 character flash; the
  typing indicator covers the pre-paint gap.
- Session history is rolling — oldest turns are trimmed, TTL slides on
  every append. Restart-safe because state lives in Redis, not RAM.
- Non-fatal everywhere: missing `DEEP_SEEK_API_KEY`, missing bot token, or
  a Redis outage only disables the bot; cron + HTTP API keep running.

## 4. Module layout — hexagonal, three isolated seams

The advisor is structured as a pure domain core (`biz/`) surrounded by
three interchangeable adapters. **Adding a new LLM vendor, chat platform,
or session backend never touches `biz/`** — you drop a sibling package
into `provider/`, `transport/`, or `storage/` that satisfies the matching
interface, then change one line in `advisor_init.go`.

```
modules/advisor/
  advisor_init.go                 wires the three adapters into ChatHandler

  biz/                            DOMAIN CORE — interfaces + pure logic only
    llm_provider.go               interface LLMProvider (Stream, Name)
    chat_transport.go             interface ChatTransport + MessageBubble +
                                      IncomingMessage DTO
    session_store.go              interface SessionStore
    market_analyzer.go            interface MarketAnalyzer (MaybeEnrich)
                                      + EnrichmentHints / EnrichmentResult — Phase 2
    chat_handler.go               orchestrator: depends ONLY on the 4 above
    user_filter.go                platform-neutral DM/allowlist rule
    prompt_builder.go             SystemPrompt + BuildMessagesWithMarket()
    greeter.go                    WelcomeMessage constant

    market/                       Phase-2 concrete MarketAnalyzer impl
      symbol_resolver.go          VI/EN alias map -> Binance ticker
      intent.go                   keyword heuristic + /analyze command parser
      market_clock.go             next H1/H4/D1 close helpers (UTC-aligned)
      digest.go                   PairSnapshot + Build + Render (hybrid prose+JSON)
      analyzer.go                 top-level MaybeEnrich (intent -> fetch -> digest)

  model/
    turn.go                       Turn{Role,Content,Time} — OpenAI-shape DTO

  provider/                       LLM VENDOR ADAPTERS
    deepseek/
      client.go                   biz.LLMProvider impl (SSE streaming)
    (openai/, anthropic/, ...)    future siblings; no changes elsewhere

  storage/                        SESSION BACKEND ADAPTERS
    redis/
      session_store.go            biz.SessionStore impl (LIST + TTL)
    (postgres/, memory/, ...)     future siblings

  transport/                      CHAT PLATFORM ADAPTERS
    telegram/
      transport.go                biz.ChatTransport impl (wraps telegram/)
    (zalo/, discord/, slack/, ...)  future siblings

telegram/                         LOW-LEVEL TELEGRAM PRIMITIVES
  telegram_service.go             EXISTING — cron broadcaster (untouched)
  advisor_bot.go                  AdvisorBot: raw getUpdates / sendMessage /
                                      editMessageText / sendChatAction
  advisor_types.go                Update/Message/Chat/User DTOs
  listener.go                     long-poll loop -> chan Update
  typing.go                       KeepTyping helper (tick 4s)
  stream_editor.go                ProgressiveMessage (throttled edit)
```

Key dependency rules enforced by the layout:

- `biz/` imports only `model/` and stdlib/log. Grep proof:
  `grep -r "j_ai_trade/" modules/advisor/biz` returns only `.../model`.
- `telegram/` imports nothing from `modules/advisor/*`. It's a reusable
  low-level package; the adapter in `transport/telegram/` is the only
  bridge.
- Adapter packages never import each other — they all depend inward on
  `biz/` and `model/`.

No `transport/gin/` yet — Phase 1 has zero HTTP surface. The advisor is
100% push-driven by the chat transport's Updates channel.

## 5. Runtime control flow (per user message)

Every step below references ONLY interface types — concrete vendor/
platform names are resolved at construction time in `advisor_init.go`.

```
1. ChatTransport.Updates()        normalized biz.IncomingMessage on channel
2. ChatHandler.Run                receives msg, fans out to handleMessage goroutine
3. UserFilter.ShouldHandle        msg.IsDM && (allowlist empty || userID allowed)
4. handleCommand                  /start /reset /help /analyze short-circuit the LLM
5. maybeGreet                     TryGreet (atomic SETNX) -> SendMessage(WelcomeMessage)
6. ChatTransport.KeepTyping       spawn ticker sending "typing" every 4s
7. SessionStore.Load              LRANGE advisor:session:<chat_id>
8. SessionStore.GetLastSymbol     GET advisor:lastsym:<chat_id> (may be "")
9. MarketAnalyzer.MaybeEnrich     intent detect (+ lastSymbol fallback for
                                    follow-ups like "bây giờ bao nhiêu") ->
                                    fetch candles -> ensemble -> digest.
                                    Returns EnrichmentResult{Digest,Ack,Symbol}.
                                    Any failure -> fall through.
10. (optional) SendMessage ack    "Đang kiểm tra XAUUSDT H4..." to signal progress
11. SessionStore.SetLastSymbol    pin result.Symbol (TTL = SessionTTL) so the
                                    NEXT turn can resolve a bare "giờ sao"
                                    back to the same pair
12. BuildMessagesWithMarket       [system, ...history, [MARKET_DATA]? , {user,text}]
13. LLMProvider.Stream            returns <-chan string, <-chan error
14. ChatTransport.NewBubble       biz.MessageBubble backed by the platform
15. bubble.Start("") -> first Append sends lazily -> Finish flushes last edit
16. SessionStore.Append           RPUSH user turn, RPUSH assistant turn, LTRIM, EXPIRE
                                    (market blob is NOT persisted — goes stale fast)
```

Per-message budget: 90s (context.WithTimeout). Even if the LLM hangs the
bubble won't stay stuck forever.

## 6. Prompt contract

The system prompt is in `biz/prompt_builder.go` as `SystemPrompt`. Key
rules pinned there (edit ONLY via that constant — single source of truth):

- Respond in user's language (VI or EN auto-detected by DeepSeek itself).
- Short replies (3–6 sentences) for chat vibe; no heavy markdown.
- **Never fabricate market data.** The bot may cite numbers ONLY when
  they appear inside a `[MARKET_DATA]...[/MARKET_DATA]` block. Outside
  that block — including when Phase 2 fails to fetch — it must say so
  and refuse to invent figures.
- **Never recycle stale prices from prior replies.** When the current
  turn has `[MARKET_DATA]`, the bot must quote from the NEW block even
  if the number looks identical to a previous reply (crypto/gold moves
  every second). When the current turn has NO `[MARKET_DATA]`, the bot
  must refuse to quote past numbers as "current" — say "chưa có data
  mới" and suggest `/analyze`. This is the invariant that makes
  follow-ups like "bây giờ bao nhiêu" actually refresh.
- Default stance: explain + endorse the rule engine's verdict. Gentle
  dissent is allowed but must cite a specific fact from the digest.
- Structured footer (emoji + Entry/SL/TP/RR/Conf/Tier lines) is required
  ONLY when the bot actually proposes a setup; casual questions stay
  free-form prose.

Each DeepSeek call receives:
`[system, ...history_from_redis, maybe [MARKET_DATA] user-turn, user_message]`.

The `[MARKET_DATA]` blob is built by `biz/market/digest.go#Render`. Shape:

1. Header line with symbol, UTC timestamp, entry TF.
2. "Next closes" for each TF in the snapshot.
3. One prose block per TF: regime, ADX, price, EMA20/50/200, RSI14, ATR,
   Bollinger bands, Donchian, swing high/low.
4. Rule engine verdict block: Direction, tier, conf, netRR, Entry/SL/TP,
   agreement ratio, per-strategy votes, veto reasons.
5. Compact JSON footer with the exact numbers the bot may echo verbatim.
6. Closing `[/MARKET_DATA]` tag.

The blob is injected as an **extra user-role turn**, not baked into the
system prompt, so (a) prompt-caching still kicks in and (b) the blob
doesn't leak into persisted history (stale data risk).

## 7. Reuse of existing code

- **`telegram/telegram_service.go`** — NOT used by advisor. That's the cron
  bot (`J_AI_TRADE_BOT_V1`) pushing signals to a fixed channel.
- **`telegram/advisor_*.go` + `telegram/stream_editor.go` + `telegram/typing.go`** —
  low-level Telegram Bot API primitives. Re-usable outside the advisor if
  any other module ever needs conversational Telegram I/O. The
  advisor-specific wiring lives in `modules/advisor/transport/telegram/`.
- **`config/redis/redis.go`** — reused. `RedisClient.GetClient()` is passed
  into `advisor.Init`.
- **`logger`** (zerolog) — standard `log.Info()/Warn()/Error()` used
  throughout the module.

Phase 2 reuse (shipped):

- **`trading/ensembles`** — shared `DefaultEnsembleFor(tf)` + `DefaultSymbols`.
  Both the cron broadcaster and the advisor market analyzer import from
  here. Adding/removing a pair or tweaking a tier wiring happens once.
- **`trading/marketdata`** — `CandleFetcher` interface + `BinanceFetcher`
  impl. Advisor calls it on every user query with analysis intent; cron
  calls it on every tier tick. Same code, same semantics.
- **`trading/indicators`** — EMA/RSI/ATR/ADX/BB/Donchian/SwingHighLow
  invoked by `biz/market/digest.go` to build the per-TF prose blocks.
- **`trading/engine`** — `DetectRegime` for the per-TF regime column in
  the digest; `StrategyInput` / `Ensemble.Analyze` for the rule-engine
  verdict embedded in the digest.
- **`brokers/binance`** — REST client wrapped by `marketdata.BinanceFetcher`.

Not yet touched: `notifier`, `modules/order`. Phase 3 proactive push
will use `notifier`.

## 7a. How to add a new adapter

### New LLM vendor (e.g. OpenAI)

1. `modules/advisor/provider/openai/client.go` — struct implementing
   `biz.LLMProvider` (methods: `Stream(ctx, []model.Turn) (<-chan string, <-chan error)`
   and `Name() string`).
2. Add env vars (`OPENAI_API_KEY`, `OPENAI_MODEL`, ...).
3. In `advisor_init.go` swap the one line:
   `llm, err := openaiProvider.New()` (or gate on an env var to choose).
4. No change in `biz/`, `transport/`, `storage/`.

### New chat platform (e.g. Zalo, Discord)

1. `modules/advisor/transport/zalo/transport.go` — struct implementing
   `biz.ChatTransport` (methods: `Updates`, `SendMessage`, `NewBubble`,
   `KeepTyping`, `Name`). Whatever streaming-edit primitives the platform
   exposes (Discord allows message edits too; Zalo may need
   "delete + resend" semantics) are hidden inside a private
   `biz.MessageBubble` impl in the adapter package.
2. Add the bot-token / credentials env var.
3. In `advisor_init.go`:
   `transport, err := zaloTransport.NewTransport(ctx)`.
4. No change in `biz/`, `provider/`, `storage/`.

### New session backend (e.g. Postgres for durable audit)

1. `modules/advisor/storage/postgres/session_store.go` — implements
   `biz.SessionStore` (5 methods).
2. Migration: add `advisor_sessions` table via
   `config/postgres/AutoMigrate`.
3. In `advisor_init.go`:
   `store := postgresStorage.NewSessionStore(db)`.

The compile-time `var _ biz.LLMProvider = (*Client)(nil)` guard in every
adapter file ensures mismatches surface at build time, not at runtime.

## 8. Environment variables

| Env var                       | Required | Purpose                                                      |
| ----------------------------- | -------- | ------------------------------------------------------------ |
| `J_AI_TRADE_ADVISOR`          | yes      | Telegram bot token for the advisor bot (separate from cron). |
| `DEEP_SEEK_API_KEY`           | yes      | DeepSeek API key. Held ONLY by backend. Never logged.        |
| `DEEP_SEEK_BASE_URL`          | no       | Default `https://api.deepseek.com`.                          |
| `DEEP_SEEK_MODEL`             | no       | Default `deepseek-chat`. `deepseek-reasoner` also supported. |
| `ADVISOR_ALLOWED_USER_IDS`    | no       | Comma-separated Telegram user IDs. Unset = public.           |
| `REDIS_HOST`, `REDIS_PORT`    | yes      | Already used by the rest of the app.                         |

JWT `AuthMiddleware` is not involved — advisor has no HTTP routes.

## 9. Session persistence (Redis layout)

```
advisor:session:<chat_id>    LIST<json(Turn)>   trimmed to last 12, TTL 30m slide
advisor:greeted:<chat_id>    STRING "1"         TTL 30 days
```

Why Redis over Postgres for Phase 1:

- Built-in TTL keeps state bounded without a cron job.
- Atomic RPUSH+LTRIM+EXPIRE via pipeline.
- Sub-ms latency on every message.
- Already configured in the repo.

Phase 2 can add a durable `advisor_sessions` table in Postgres for audit
logs without changing the `SessionStore` interface.

## 10. Failure modes & fallbacks

| Failure                              | Behavior                                                        |
| ------------------------------------ | --------------------------------------------------------------- |
| `J_AI_TRADE_ADVISOR` missing         | Log warn, `advisor.Init` returns early. Cron + HTTP keep going. |
| `DEEP_SEEK_API_KEY` missing          | Same — advisor disabled, rest of app fine.                      |
| Redis unreachable at startup         | `advisor.Init` is skipped in `main.go`.                         |
| Redis hiccup mid-session             | Load returns empty -> bot answers without history. Append logged. |
| DeepSeek non-2xx / timeout           | Stream error drained; if no tokens arrived user sees a polite apology message; existing bubble stays if partial. |
| Telegram 429 / edit-rate-limit       | `editMessageText` error is logged at debug; next flush retries. |
| `getUpdates` transient error         | Linear backoff 1s -> 2s -> ... 30s, retries forever.            |
| Handler panic                        | `defer recover()` in `handleMessage`; logs + returns.           |

## 11. Testing plan (not yet implemented)

Hexagonal layout makes test writing easy — `biz/` is tested with **fakes
of the three interfaces**, no real vendor calls:

- `storage/redis/session_store_test.go` — miniredis; assert trim + TTL.
- `provider/deepseek/client_test.go` — `httptest.Server` returning canned
  SSE events; verify chunk channel, graceful `[DONE]`, error path.
- `transport/telegram/transport_test.go` — intercept HTTP with
  `httptest.Server`; verify `IncomingMessage` normalization + bubble edits.
- `biz/prompt_builder_test.go` — system prompt is first message + history
  order preserved.
- `biz/user_filter_test.go` — table-driven: DM/group, allowlist on/off.
- `biz/chat_handler_test.go` — fake `ChatTransport` + fake `LLMProvider`
  (in-memory channel) + miniredis-backed store. End-to-end: greeting
  only once, reset clears, bubble sees >=1 append.

## 12. Phase 2/3 roadmap

### Phase 2 — on-demand market analysis (SHIPPED)

- [x] Extract `DefaultEnsembleFor(tf)` into `trading/ensembles`
      (shared by cron + advisor).
- [x] Extract candle-fetch into `trading/marketdata` (`CandleFetcher`
      interface + `BinanceFetcher` impl).
- [x] `biz/market/symbol_resolver.go` — VI/EN aliases, universe bounded
      by `ensembles.DefaultSymbols`.
- [x] `biz/market/intent.go` — keyword heuristic + `/analyze SYMBOL [TF]`
      command parser.
- [x] `biz/market/market_clock.go` — next H1/H4/D1 close helpers.
- [x] `biz/market/digest.go` — `PairSnapshot` + `Render` hybrid
      prose+JSON blob.
- [x] `biz/market/analyzer.go` — `MaybeEnrich` top-level wiring,
      non-fatal on fetch errors.
- [x] `biz.MarketAnalyzer` interface + optional hook in `ChatHandler`.
- [x] `SystemPrompt` updated: bot cites numbers only inside
      `[MARKET_DATA]`, defaults to explaining rule engine verdict, uses
      structured footer only when proposing a setup.
- [x] Graceful degradation: Binance outage -> chat-only fallback.

### Phase 2.5 — deferred

- [ ] Audit table `advisor_sessions` in Postgres for conversation logs
      + cost tracking.
- [ ] Digest unit tests (fake candles -> golden prompt).
- [ ] Per-user session quota / rate limit.

### Phase 3 — proactive + multi-modal

- Bot DMs the user when the cron's ensemble fires a high-confidence signal
  matching their watchlist ("Hey, XAU just printed a bullish BOS on H1...").
- Switch DeepSeek model to `deepseek-reasoner` for signal-generation
  requests (keep `deepseek-chat` for small talk to save tokens).
- News digest (fundamentals, macro calendar) as a second MarketAnalyzer-
  shaped enrichment hook.
- Optional webhook mode (`setWebhook`) for lower first-token latency in
  prod (polling is fine for local dev).
