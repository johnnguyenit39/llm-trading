# .agent — Engineering Context Docs

Short, self-contained context files for priming future prompts. Each file
is meant to be dropped into a chat as-is so the agent has enough context
to modify the module without re-reading the whole codebase.

## Index

| File                                   | Scope                                                                                                                                                                         | When to include                                                                                                  |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| [advisor-module.md](advisor-module.md) | The whole app. A Telegram bot that fetches market data, hands it to DeepSeek, streams the reply, and persists any trade JSON the LLM emits. No HTTP, no cron, no user module. | Any task touching `modules/advisor/`, `modules/agent_decision/`, `telegram/`, DeepSeek client, Redis, or Postgres schema. |

## Global conventions (apply to every doc here)

- **Codebase**: single Go binary (`j_ai_trade`). Long-polls Telegram, talks
  to DeepSeek + Binance REST, writes `agent_decisions` to Postgres, uses
  Redis for chat sessions.
- **Hexagonal**: `biz/` defines interfaces, `storage/` / `transport/` /
  `provider/` host adapters. biz never imports a vendor.
- **Docs language**: English. Diagrams via Mermaid (no inline styling — it
  breaks the dark theme).
- **When a doc becomes stale**: update it in the same PR that changes the
  code. The doc IS the prompt — drift here costs LLM accuracy in every
  future session.
