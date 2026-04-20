package engine

import (
	"context"
	"testing"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/models"
)

// trendUpMarket returns market data whose entry TF classifies as TrendUp.
func trendUpMarket() models.MarketData {
	return models.MarketData{
		Symbol: "BTCUSDT",
		Candles: map[models.Timeframe][]baseCandle.BaseCandle{
			models.TF_H1: uptrendCandles(250, 100, 0.5),
		},
	}
}

// buy/sell/none helpers with a sane RR so NetRR gate passes for buy/sell.
func buyVote(conf float64) models.StrategyVote {
	return models.StrategyVote{
		Direction: models.DirectionBuy, Confidence: conf,
		Entry: 100, StopLoss: 98, TakeProfit: 105,
	}
}
func sellVote(conf float64) models.StrategyVote {
	return models.StrategyVote{
		Direction: models.DirectionSell, Confidence: conf,
		Entry: 100, StopLoss: 102, TakeProfit: 95,
	}
}
func noneVote() models.StrategyVote {
	return models.StrategyVote{Direction: models.DirectionNone}
}

func runWith(strats []Strategy) *models.TradeDecision {
	e := NewEnsemble(NewDefaultRiskManager(), DefaultEnsembleConfig())
	for _, s := range strats {
		e.Register(s)
	}
	return e.Analyze(context.Background(), StrategyInput{
		Market:  trendUpMarket(),
		Equity:  1000,
		EntryTF: models.TF_H1,
	})
}

func TestEnsemble_FullConsensusBuys(t *testing.T) {
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionBuy {
		t.Fatalf("expected BUY, got %s (reason=%s)", dec.Direction, dec.Reason)
	}
	if dec.Tier != "full" {
		t.Errorf("expected full tier, got %q", dec.Tier)
	}
	if dec.SizeFactor != 1.0 {
		t.Errorf("expected SizeFactor=1.0, got %v", dec.SizeFactor)
	}
	if dec.Agreement != 3 || dec.EligibleCount != 3 {
		t.Errorf("expected 3/3 eligible, got %d/%d", dec.Agreement, dec.EligibleCount)
	}
}

func TestEnsemble_HalfConsensus(t *testing.T) {
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: noneVote()},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionBuy {
		t.Fatalf("expected BUY, got %s", dec.Direction)
	}
	if dec.Tier != "half" {
		t.Errorf("expected half tier, got %q", dec.Tier)
	}
	if dec.SizeFactor != 0.5 {
		t.Errorf("expected SizeFactor=0.5, got %v", dec.SizeFactor)
	}
}

func TestEnsemble_QuarterLoneStrong(t *testing.T) {
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(88)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: noneVote()},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: noneVote()},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionBuy {
		t.Fatalf("expected BUY, got %s (reason=%s)", dec.Direction, dec.Reason)
	}
	if dec.Tier != "quarter" {
		t.Errorf("expected quarter tier, got %q", dec.Tier)
	}
	if dec.SizeFactor != 0.25 {
		t.Errorf("expected SizeFactor=0.25, got %v", dec.SizeFactor)
	}
}

func TestEnsemble_DissentVetoKills(t *testing.T) {
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: sellVote(90)}, // ≥ DissentVetoConf
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE from dissent veto, got %s", dec.Direction)
	}
	if len(dec.VetoReasons) == 0 {
		t.Error("expected veto reasons populated")
	}
}

func TestEnsemble_SplitDecision(t *testing.T) {
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(75)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: sellVote(75)},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE on split decision, got %s", dec.Direction)
	}
}

func TestEnsemble_WeakConsensus(t *testing.T) {
	// Two voters below any tier's confidence/ratio requirement.
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(55)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(55)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: noneVote()},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE on weak consensus, got %s (reason=%s)", dec.Direction, dec.Reason)
	}
}

func TestEnsemble_NetRRGateRejects(t *testing.T) {
	// All three vote BUY with TP way too close → netRR < MinNetRR (1.3).
	tight := models.StrategyVote{
		Direction: models.DirectionBuy, Confidence: 80,
		Entry: 100, StopLoss: 98, TakeProfit: 100.5, // raw RR 0.25
	}
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: tight},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: tight},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: tight},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE on low NetRR, got %s (reason=%s)", dec.Direction, dec.Reason)
	}
}

func TestEnsemble_NoEligibleStrategies(t *testing.T) {
	// Only range-eligible strategies but regime is TrendUp → 0 eligible.
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeRange}, vote: buyVote(90)},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE with 0 eligible, got %s", dec.Direction)
	}
	if dec.EligibleCount != 0 {
		t.Errorf("expected 0 eligible, got %d", dec.EligibleCount)
	}
}

func TestEnsemble_CoherentEntrySLTP(t *testing.T) {
	// Invariant: if a BUY fires, SL < Entry < TP (anchor preserves triplet).
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(82)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(75)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(70)},
	}
	dec := runWith(strats)
	if dec.Direction != models.DirectionBuy {
		t.Fatalf("setup expected to fire BUY, got %s", dec.Direction)
	}
	if !(dec.StopLoss < dec.Entry && dec.Entry < dec.TakeProfit) {
		t.Errorf("incoherent BUY triplet: SL=%v Entry=%v TP=%v", dec.StopLoss, dec.Entry, dec.TakeProfit)
	}
	if dec.NetRR <= 0 {
		t.Errorf("expected positive NetRR, got %v", dec.NetRR)
	}
}

func TestEnsemble_ExposureCapVetoes(t *testing.T) {
	exposure := NewExposureTracker()
	// Pre-fill exposure to near cap so next signal will exceed.
	// equity=1000, MaxTotal=3.0 → cap=$3000. Pre-commit $2900.
	exposure.Commit("ETHUSDT", 2900, time.Hour)

	e := NewEnsemble(NewDefaultRiskManager(), DefaultEnsembleConfig()).WithExposureTracker(exposure)
	strats := []Strategy{
		&fakeStrategy{name: "a", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "b", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
		&fakeStrategy{name: "c", regimes: []models.Regime{models.RegimeTrendUp}, vote: buyVote(80)},
	}
	for _, s := range strats {
		e.Register(s)
	}
	dec := e.Analyze(context.Background(), StrategyInput{
		Market:  trendUpMarket(),
		Equity:  1000,
		EntryTF: models.TF_H1,
	})
	if dec.Direction != models.DirectionNone {
		t.Errorf("expected NONE from exposure cap, got %s (reason=%s)", dec.Direction, dec.Reason)
	}
}
