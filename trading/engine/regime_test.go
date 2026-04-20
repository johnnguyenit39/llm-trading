package engine

import (
	"testing"

	"j_ai_trade/trading/models"
)

func TestDetectRegime_TrendUp(t *testing.T) {
	candles := uptrendCandles(250, 100, 0.5)
	got := DetectRegime(candles, DefaultRegimeThresholds())
	if got != models.RegimeTrendUp {
		t.Errorf("expected TrendUp, got %s", got)
	}
}

func TestDetectRegime_TrendDown(t *testing.T) {
	candles := downtrendCandles(250, 300, 0.5)
	got := DetectRegime(candles, DefaultRegimeThresholds())
	if got != models.RegimeTrendDown {
		t.Errorf("expected TrendDown, got %s", got)
	}
}

func TestDetectRegime_Range(t *testing.T) {
	candles := rangeCandles(250, 100, 2)
	got := DetectRegime(candles, DefaultRegimeThresholds())
	if got != models.RegimeRange {
		t.Errorf("expected Range, got %s", got)
	}
}

func TestDetectRegime_InsufficientCandles(t *testing.T) {
	candles := uptrendCandles(100, 100, 0.5)
	got := DetectRegime(candles, DefaultRegimeThresholds())
	if got != models.RegimeChoppy {
		t.Errorf("expected Choppy on insufficient data, got %s", got)
	}
}

func TestDetectRegimeMulti_ContradictingTFs(t *testing.T) {
	entry := uptrendCandles(250, 100, 0.5)
	htf := downtrendCandles(250, 300, 0.5)
	got := DetectRegimeMulti(entry, htf, DefaultRegimeThresholds())
	if got != models.RegimeChoppy {
		t.Errorf("expected Choppy when entry up vs HTF down, got %s", got)
	}
}

func TestDetectRegimeMulti_RangeVsStrongHTFTrend(t *testing.T) {
	entry := rangeCandles(250, 100, 2)
	htf := uptrendCandles(250, 100, 0.5) // strong up
	got := DetectRegimeMulti(entry, htf, DefaultRegimeThresholds())
	if got != models.RegimeChoppy {
		t.Errorf("expected Choppy when range conflicts with strong HTF trend, got %s", got)
	}
}

func TestDetectRegimeMulti_AlignedKeepsEntry(t *testing.T) {
	entry := uptrendCandles(250, 100, 0.5)
	htf := uptrendCandles(250, 100, 0.5)
	got := DetectRegimeMulti(entry, htf, DefaultRegimeThresholds())
	if got != models.RegimeTrendUp {
		t.Errorf("expected TrendUp when both TFs agree, got %s", got)
	}
}
