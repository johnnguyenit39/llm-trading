// cmd/backtest replays historical candles to evaluate the bot's
// signal quality without paying spread / slippage / live latency. The
// loop is intentionally simple:
//
//  1. Sample N timestamps uniformly in [now − weeks, now − walkBuffer].
//  2. For each timestamp `t`, pull multi-TF candles ending at t and
//     build the SAME digest production sees (so prompt parity is exact).
//  3. Feed [system_prompt + market_blob + user_question] to DeepSeek
//     with temperature=0 + fixed seed for reproducibility. Local cache
//     means re-runs that don't change the prompt are free.
//  4. Parse any ```json``` decision the LLM emits; walk forward 4h of
//     M1 candles to see whether TP or SL hit first.
//  5. Aggregate: hit rate, breakdown by confidence, MFE/MAE.
//
// Look-ahead bias note: DeepSeek's training cutoff (~mid-2024) means
// post-cutoff samples (the 6-week default) are clean — the model has
// no weights-encoded knowledge of those moments. If you stretch the
// window past 12 months the bias risk grows; consider price-shifting
// or using deepseek-reasoner with a more recent cutoff.
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/logger"
	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/biz/market"
	"j_ai_trade/modules/advisor/model"
	"j_ai_trade/modules/advisor/provider/deepseek"
	"j_ai_trade/trading/models"
)

func main() {
	logger.InitializeLogger()

	if err := godotenv.Load(); err != nil {
		log.Debug().Err(err).Msg("backtest: no .env loaded; using process env")
	}

	cfg := parseFlags()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bs := binance.NewBinanceService(repository.NewBinanceRepository())
	hist := newHistoricalFetcher(bs)

	client, err := deepseek.New()
	if err != nil {
		log.Fatal().Err(err).Msg("backtest: DeepSeek client init failed")
	}
	client = client.WithTemperature(cfg.temperature).WithSeed(cfg.seed)

	cache, err := newFileCache(cfg.cacheDir)
	if err != nil {
		log.Fatal().Err(err).Msg("backtest: cache init failed")
	}
	runner := newLLMRunner(client, cache, deepseekModelFromEnv(), cfg.temperature, cfg.seed)

	timestamps := sampleTimestamps(cfg)
	log.Info().
		Int("samples", len(timestamps)).
		Time("first", timestamps[0]).
		Time("last", timestamps[len(timestamps)-1]).
		Msg("backtest: sample window")

	report := &runReport{
		Symbol:      cfg.symbol,
		Samples:     len(timestamps),
		Weeks:       cfg.weeks,
		Model:       deepseekModelFromEnv(),
		Temperature: cfg.temperature,
		Seed:        cfg.seed,
		GeneratedAt: time.Now().UTC(),
	}

	for i, t := range timestamps {
		select {
		case <-ctx.Done():
			log.Warn().Int("done", i).Msg("backtest: interrupted")
			break
		default:
		}
		res := runOneSample(ctx, runner, hist, cfg, i, t)
		if res.Cached {
			report.CacheHits++
		} else if res.Error == "" {
			report.APICallsMade++
		}
		report.Results = append(report.Results, res)
		fmt.Fprintf(os.Stderr, "  [%d/%d] %s  %s\n", i+1, len(timestamps),
			t.Format("01-02 15:04"), summariseSample(res))
	}

	if err := writeReport(cfg.outFile, report); err != nil {
		log.Warn().Err(err).Str("path", cfg.outFile).Msg("backtest: report write failed")
	}
	summarise(os.Stdout, report)
}

// runOneSample executes the per-sample pipeline. Errors are recorded
// in the result rather than aborting the run — a single bad sample
// shouldn't kill a 200-sample budget you've already paid for.
func runOneSample(ctx context.Context, runner *llmRunner, hist *historicalFetcher, cfg *config, idx int, t time.Time) sampleResult {
	res := sampleResult{Index: idx, SampledAt: t.UTC()}

	required := map[models.Timeframe]int{
		models.TF_M1: market.CandleBudget,
		models.TF_M5: market.CandleBudget,
		models.TF_H1: market.CandleBudget,
		models.TF_H4: market.CandleBudget,
	}
	mkt, err := hist.FetchSnapshotAt(ctx, cfg.symbol, t, required)
	if err != nil {
		res.Error = "fetch_snapshot: " + err.Error()
		return res
	}
	snap, err := market.Build(mkt, models.TF_M5, t)
	if err != nil {
		res.Error = "digest_build: " + err.Error()
		return res
	}
	digest := market.Render(snap)
	if digest == "" {
		res.Error = "empty_digest"
		return res
	}

	turns := biz.BuildMessagesWithMarket(nil, cfg.userPrompt, digest)
	// Strip Time fields from the cache key path: BuildMessagesWithMarket
	// stamps time.Now() into Turn.Time. That field never reaches the
	// wire (only Role/Content do) so we just hash on what DeepSeek sees.
	for i := range turns {
		turns[i].Time = time.Time{}
	}

	reply, cached, err := runner.Run(ctx, turns)
	if err != nil {
		res.Error = "llm: " + err.Error()
		return res
	}
	res.Cached = cached
	if len(reply) > 200 {
		res.ReplyExcerpt = reply[:200]
	} else {
		res.ReplyExcerpt = reply
	}

	d := biz.ExtractDecision(reply)
	if d == nil {
		// LLM declined to fire — that IS a result. Track it as no-trade.
		res.NoTrade = true
		return res
	}
	res.Decision = d

	forward, err := hist.FetchM1Forward(ctx, cfg.symbol, t, cfg.walkBars)
	if err != nil {
		res.Error = "forward_fetch: " + err.Error()
		return res
	}
	if len(forward) == 0 {
		res.Error = "no_forward_candles"
		return res
	}
	out := resolveOutcome(d, forward)
	res.Outcome = &out
	return res
}

func summariseSample(r sampleResult) string {
	cached := ""
	if r.Cached {
		cached = " (cached)"
	}
	if r.Error != "" {
		return "ERR: " + r.Error + cached
	}
	if r.NoTrade {
		return "no-trade" + cached
	}
	if r.Decision == nil || r.Outcome == nil {
		return "??" + cached
	}
	return fmt.Sprintf("%s %s -> %s%s", r.Decision.Action, r.Decision.Symbol, r.Outcome.Kind, cached)
}

// ---- config / flag plumbing ----

type config struct {
	symbol      string
	samples     int
	weeks       int
	temperature float64
	seed        int
	walkBars    int
	cacheDir    string
	outFile     string
	userPrompt  string
	rng         *rand.Rand
}

func parseFlags() *config {
	cfg := &config{}
	flag.StringVar(&cfg.symbol, "symbol", "XAUUSDT", "symbol to backtest")
	flag.IntVar(&cfg.samples, "samples", 50, "number of timestamps to sample")
	flag.IntVar(&cfg.weeks, "weeks", 6, "history window in weeks (must be > warm-up of ~33 days for H4 budget)")
	flag.Float64Var(&cfg.temperature, "temp", 0, "LLM sampling temperature (0 for deterministic)")
	flag.IntVar(&cfg.seed, "seed", 42, "LLM seed for reproducibility")
	flag.IntVar(&cfg.walkBars, "walk-bars", 240, "M1 bars to walk forward when checking outcome (240 = 4h)")
	flag.StringVar(&cfg.cacheDir, "cache", "./backtest_cache", "directory for response cache (skip API on hit)")
	flag.StringVar(&cfg.outFile, "out", "./backtest_report.json", "JSON report path (empty to skip)")
	flag.StringVar(&cfg.userPrompt, "prompt", "Phân tích XAUUSDT giúp t.", "user message paired with the [MARKET_DATA] digest")
	rngSeed := flag.Int64("rng-seed", 1, "seed for the timestamp sampler (independent of LLM seed)")
	flag.Parse()
	cfg.rng = rand.New(rand.NewSource(*rngSeed))
	if cfg.outFile != "" {
		if abs, err := filepath.Abs(cfg.outFile); err == nil {
			cfg.outFile = abs
		}
	}
	return cfg
}

// sampleTimestamps picks `samples` uniform random points in
// [now − weeks·7d + warmup, now − walkBuffer]. Warmup leaves enough
// history for the H4 indicator pipeline (200 bars × 4h = 33 days);
// the walk buffer gives us 4h of forward M1 to resolve outcomes.
func sampleTimestamps(cfg *config) []time.Time {
	now := time.Now().UTC()
	warmup := time.Duration(market.CandleBudget) * 4 * time.Hour // ≈33 days for H4 = 200
	walkBuffer := time.Duration(cfg.walkBars+5) * time.Minute
	end := now.Add(-walkBuffer)
	start := now.Add(-time.Duration(cfg.weeks) * 7 * 24 * time.Hour).Add(warmup)
	if !start.Before(end) {
		log.Fatal().
			Time("start", start).
			Time("end", end).
			Msg("backtest: window too tight — increase -weeks or relax warmup")
	}
	span := end.Sub(start)
	out := make([]time.Time, cfg.samples)
	for i := 0; i < cfg.samples; i++ {
		raw := start.Add(time.Duration(cfg.rng.Int63n(int64(span))))
		// Truncate to M5 boundary. Two reasons: (a) cache key stability —
		// sub-minute jitter would re-hash the digest header
		// (`GeneratedAt.Format("2006-01-02 15:04")`) and miss cache on
		// the next run for no signal-quality reason. (b) realism: live
		// users typically ask AFTER an entry-TF bar closed, not in the
		// middle of one.
		out[i] = raw.Truncate(5 * time.Minute)
	}
	// Sort so the run-time progress bar reads chronologically; the
	// statistical properties are unaffected (samples are still iid).
	sortTimes(out)
	return out
}

func sortTimes(ts []time.Time) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j].Before(ts[j-1]); j-- {
			ts[j], ts[j-1] = ts[j-1], ts[j]
		}
	}
}

func deepseekModelFromEnv() string {
	if v := os.Getenv("DEEP_SEEK_MODEL"); v != "" {
		return v
	}
	return "deepseek-chat"
}

// Compile-time check that we depend on model.Turn / Role consts so a
// schema rename surfaces here too. Keeps grep-discoverability.
var _ = model.RoleUser
