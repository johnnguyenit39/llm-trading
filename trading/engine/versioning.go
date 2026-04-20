package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"j_ai_trade/trading/models"
)

// ConfigSnapshot captures every parameter that influences signal generation.
// Its canonical JSON is hashed to produce a StrategyVersion fingerprint.
//
// CRITICAL: whenever a strategy file gains/changes a tunable magic number, the
// developer must either (a) mirror that number into Strategies here, or (b)
// create a new strategy file (e.g. trend_follow_v2.go) and add its entry to
// Strategies. Both approaches flip the fingerprint, forcing a new version
// row — which is exactly the intent.
type ConfigSnapshot struct {
	Ensemble   EnsembleSnapshot            `json:"ensemble"`
	Risk       RiskSnapshot                `json:"risk"`
	PairCfgs   map[string]PairConfig       `json:"pair_configs"`
	Pairs      []string                    `json:"pairs"`
	Strategies map[string]map[string]any   `json:"strategies"`
	Tiers      map[string]TierSnapshot     `json:"tiers"`
}

// EnsembleSnapshot mirrors the subset of EnsembleConfig that is actually
// hashable / meaningful for backtests. Runtime-only fields (e.g. TTLs that
// come from cron cadence) are excluded on purpose so identical strategies
// running on different schedules still share a fingerprint.
type EnsembleSnapshot struct {
	FullAgreement   int     `json:"full_agreement"`
	FullRatio       float64 `json:"full_ratio"`
	FullAvgConf     float64 `json:"full_avg_conf"`
	HalfAgreement   int     `json:"half_agreement"`
	HalfRatio       float64 `json:"half_ratio"`
	HalfAvgConf     float64 `json:"half_avg_conf"`
	QuarterMinConf  float64 `json:"quarter_min_conf"`
	DissentVetoConf float64 `json:"dissent_veto_conf"`
	MinNetRR        float64 `json:"min_net_rr"`
	Regime          RegimeThresholds `json:"regime"`
}

// RiskSnapshot mirrors RiskManager's tunables. PairConfigs are captured
// separately because they're keyed by symbol.
type RiskSnapshot struct {
	RiskPerTradePct    float64 `json:"risk_per_trade_pct"`
	MarginRatio        float64 `json:"margin_ratio"`
	MaxTotalNotional   float64 `json:"max_total_notional"`
	MinActualRiskRatio float64 `json:"min_actual_risk_ratio"`
	FeeRateTaker       float64 `json:"fee_rate_taker"`
}

// TierSnapshot captures the entry TF -> (trendTF, structureTF, htfRegime) wiring
// used by buildEnsemble, so a backtest knows how the multi-TF engine was
// configured for each tier.
type TierSnapshot struct {
	EntryTF     models.Timeframe `json:"entry_tf"`
	TrendTF     models.Timeframe `json:"trend_tf"`
	StructureTF models.Timeframe `json:"structure_tf"`
	HTFRegime   models.Timeframe `json:"htf_regime"`
}

// BuildSnapshot constructs a ConfigSnapshot from the current runtime config.
// Strategy params are hardcoded here as the source-of-truth for fingerprinting;
// keep them in sync with the strategy code whenever tuning changes.
func BuildSnapshot(ensCfg EnsembleConfig, risk *RiskManager, pairs []string, tiers map[string]TierSnapshot) ConfigSnapshot {
	sortedPairs := append([]string(nil), pairs...)
	sort.Strings(sortedPairs)

	return ConfigSnapshot{
		Ensemble: EnsembleSnapshot{
			FullAgreement:   ensCfg.FullAgreement,
			FullRatio:       ensCfg.FullRatio,
			FullAvgConf:     ensCfg.FullAvgConf,
			HalfAgreement:   ensCfg.HalfAgreement,
			HalfRatio:       ensCfg.HalfRatio,
			HalfAvgConf:     ensCfg.HalfAvgConf,
			QuarterMinConf:  ensCfg.QuarterMinConf,
			DissentVetoConf: ensCfg.DissentVetoConf,
			MinNetRR:        ensCfg.MinNetRR,
			Regime:          ensCfg.Regime,
		},
		Risk: RiskSnapshot{
			RiskPerTradePct:    risk.RiskPerTradePct,
			MarginRatio:        risk.MarginRatio,
			MaxTotalNotional:   risk.MaxTotalNotional,
			MinActualRiskRatio: risk.MinActualRiskRatio,
			FeeRateTaker:       risk.FeeRateTaker,
		},
		PairCfgs:   risk.PairConfigs,
		Pairs:      sortedPairs,
		Strategies: defaultStrategyParams(),
		Tiers:      tiers,
	}
}

// defaultStrategyParams mirrors the magic numbers embedded in each strategy
// file. This is the single place to bump when a strategy changes behavior.
// If a v2 strategy is introduced (trend_follow_v2.go), add a new key such as
// "trend_follow_v2" with its params.
func defaultStrategyParams() map[string]map[string]any {
	return map[string]map[string]any{
		"trend_follow": {
			"local_adx_min":    20.0,
			"ema_fast":         20,
			"ema_mid":          50,
			"ema_slow":         200,
			"atr_mult_sl":      1.5,
			"atr_mult_tp":      3.0,
			"pullback_max_atr": 0.5,
			"conf_base":        60.0,
			"conf_adx_scale":   1.6,
			"conf_max":         90.0,
		},
		"mean_reversion": {
			"local_adx_max":    25.0,
			"bb_period":        20,
			"bb_std":           2.0,
			"rsi_period":       14,
			"rsi_oversold":     30.0,
			"rsi_overbought":   70.0,
			"atr_mult_sl":      1.0,
			"atr_mult_tp":      2.0,
		},
		"breakout": {
			"donchian_period": 20,
			"atr_period":      14,
			"atr_mult_sl":     1.5,
			"atr_mult_tp":     3.0,
		},
		"structure": {
			"swing_lookback": 3,
			"atr_mult_sl":    1.5,
			"atr_mult_tp":    3.0,
		},
	}
}

// CanonicalJSON serializes the snapshot with sorted keys so fingerprints are
// deterministic regardless of field-insertion order.
func (s ConfigSnapshot) CanonicalJSON() ([]byte, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return marshalSorted(generic)
}

// Fingerprint returns the sha256 hex of the canonical JSON.
func (s ConfigSnapshot) Fingerprint() (string, []byte, error) {
	canon, err := s.CanonicalJSON()
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), canon, nil
}

// DiffSnapshots compares two canonical snapshots field-by-field and returns a
// short human-readable changelog, suitable for the StrategyVersion.Notes column.
func DiffSnapshots(oldCanon, newCanon []byte) string {
	var oldMap, newMap map[string]any
	_ = json.Unmarshal(oldCanon, &oldMap)
	_ = json.Unmarshal(newCanon, &newMap)
	if oldMap == nil && newMap == nil {
		return ""
	}
	changes := diffMaps("", oldMap, newMap)
	if len(changes) == 0 {
		return "reactivated (no changes)"
	}
	sort.Strings(changes)
	return "changes: " + joinLines(changes)
}

// --- internal helpers ---

func marshalSorted(v any) ([]byte, error) {
	switch tv := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(tv))
		for k := range tv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := []byte{'{'}
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, _ := json.Marshal(k)
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := marshalSorted(tv[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return buf, nil
	case []any:
		buf := []byte{'['}
		for i, item := range tv {
			if i > 0 {
				buf = append(buf, ',')
			}
			vb, err := marshalSorted(item)
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, ']')
		return buf, nil
	default:
		return json.Marshal(tv)
	}
}

func diffMaps(prefix string, oldMap, newMap map[string]any) []string {
	out := []string{}
	seen := map[string]bool{}
	for k, oldVal := range oldMap {
		seen[k] = true
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		newVal, ok := newMap[k]
		if !ok {
			out = append(out, fmt.Sprintf("- %s", path))
			continue
		}
		out = append(out, diffValues(path, oldVal, newVal)...)
	}
	for k, newVal := range newMap {
		if seen[k] {
			continue
		}
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		out = append(out, fmt.Sprintf("+ %s=%v", path, newVal))
	}
	return out
}

func diffValues(path string, oldVal, newVal any) []string {
	oldM, oldIsMap := oldVal.(map[string]any)
	newM, newIsMap := newVal.(map[string]any)
	if oldIsMap && newIsMap {
		return diffMaps(path, oldM, newM)
	}
	ob, _ := json.Marshal(oldVal)
	nb, _ := json.Marshal(newVal)
	if string(ob) == string(nb) {
		return nil
	}
	return []string{fmt.Sprintf("~ %s: %s -> %s", path, string(ob), string(nb))}
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "; "
		}
		out += l
	}
	return out
}
