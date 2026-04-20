package indicators

import (
	"math"
	"testing"
	"time"

	baseCandle "j_ai_trade/common"
)

func TestSMA_Basic(t *testing.T) {
	if got := SMA([]float64{1, 2, 3, 4, 5}, 5); got != 3 {
		t.Errorf("SMA = %v, want 3", got)
	}
	if got := SMA([]float64{1, 2, 3, 4, 5}, 3); got != 4 {
		t.Errorf("SMA of last 3 = %v, want 4", got)
	}
}

func TestSMA_InsufficientData(t *testing.T) {
	if got := SMA([]float64{1, 2}, 5); got != 0 {
		t.Errorf("SMA with insufficient data should return 0, got %v", got)
	}
}

func TestEMA_ConstantSeries(t *testing.T) {
	vals := make([]float64, 30)
	for i := range vals {
		vals[i] = 100
	}
	if got := EMA(vals, 10); math.Abs(got-100) > 1e-9 {
		t.Errorf("EMA of constant = %v, want 100", got)
	}
}

func TestRSI_AllGains(t *testing.T) {
	vals := make([]float64, 30)
	for i := range vals {
		vals[i] = 100 + float64(i)
	}
	if got := RSI(vals, 14); got != 100 {
		t.Errorf("RSI of strictly increasing = %v, want 100", got)
	}
}

func TestRSI_AllLosses(t *testing.T) {
	vals := make([]float64, 30)
	for i := range vals {
		vals[i] = 100 - float64(i)
	}
	got := RSI(vals, 14)
	if got > 5 {
		t.Errorf("RSI of strictly decreasing = %v, want near 0", got)
	}
}

func TestATR_PositiveOnMovingCandles(t *testing.T) {
	bars := make([]baseCandle.BaseCandle, 20)
	for i := range bars {
		bars[i] = baseCandle.BaseCandle{
			High:  100 + float64(i) + 1,
			Low:   100 + float64(i) - 1,
			Close: 100 + float64(i),
		}
	}
	if got := ATR(bars, 14); got <= 0 {
		t.Errorf("ATR should be positive, got %v", got)
	}
}

func TestDonchianChannel(t *testing.T) {
	bars := make([]baseCandle.BaseCandle, 20)
	for i := range bars {
		bars[i] = baseCandle.BaseCandle{
			High: 100 + float64(i), // 100..119
			Low:  80 - float64(i),  // 80..61
		}
	}
	hi, lo := DonchianChannel(bars, 20)
	if hi != 119 {
		t.Errorf("Donchian high = %v, want 119", hi)
	}
	if lo != 61 {
		t.Errorf("Donchian low = %v, want 61", lo)
	}
}

func TestBollingerBands_Centering(t *testing.T) {
	// Constant series → zero stddev → bands collapse to middle.
	vals := make([]float64, 30)
	for i := range vals {
		vals[i] = 100
	}
	upper, mid, lower := BollingerBands(vals, 20, 2.0)
	if mid != 100 {
		t.Errorf("middle = %v, want 100", mid)
	}
	if math.Abs(upper-100) > 1e-9 || math.Abs(lower-100) > 1e-9 {
		t.Errorf("bands should collapse on constant series: upper=%v lower=%v", upper, lower)
	}
}

func TestClosedCandles_DropsLast(t *testing.T) {
	bars := []baseCandle.BaseCandle{
		{Close: 1}, {Close: 2}, {Close: 3},
	}
	got := ClosedCandles(bars)
	if len(got) != 2 {
		t.Errorf("expected 2 closed, got %d", len(got))
	}
	if got[len(got)-1].Close != 2 {
		t.Errorf("last closed = %v, want 2", got[len(got)-1].Close)
	}
}

func TestSwingHighLow_DetectsPivots(t *testing.T) {
	// For leftRight=3 with n=10, the scan window covers indices [3,6].
	// Highs peak at index 5 (value 10), lows trough at index 4 (value 1).
	highs := []float64{5, 6, 7, 8, 9, 10, 7, 6, 5, 4}
	lows := []float64{5, 6, 7, 8, 1, 2, 3, 4, 5, 4}
	bars := make([]baseCandle.BaseCandle, 10)
	for i := range bars {
		bars[i] = baseCandle.BaseCandle{
			High:     highs[i],
			Low:      lows[i],
			Close:    highs[i],
			OpenTime: time.Unix(int64(i), 0),
		}
	}
	sh, sl := SwingHighLow(bars, 3)
	if sh != 10 {
		t.Errorf("swing high = %v, want 10", sh)
	}
	if sl != 1 {
		t.Errorf("swing low = %v, want 1", sl)
	}
}
