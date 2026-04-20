package engine

// PairConfig defines per-symbol trading constraints.
type PairConfig struct {
	MaxLeverage  float64
	MinNotional  float64 // minimum order notional ($), skip if below
	Priority     int     // higher = preferred when capital is scarce
}

// DefaultPairConfigs returns sensible defaults per symbol.
// Add / tune per-pair overrides here; unknown symbols fall back to "default".
func DefaultPairConfigs() map[string]PairConfig {
	return map[string]PairConfig{
		"BTCUSDT":  {MaxLeverage: 25, MinNotional: 50, Priority: 10},
		"ETHUSDT":  {MaxLeverage: 20, MinNotional: 50, Priority: 9},
		"BNBUSDT":  {MaxLeverage: 15, MinNotional: 30, Priority: 7},
		"SOLUSDT":  {MaxLeverage: 15, MinNotional: 30, Priority: 7},
		"XAUUSDT":  {MaxLeverage: 20, MinNotional: 50, Priority: 8},
		"XRPUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"ADAUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"AVAXUSDT": {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"MATICUSDT":{MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"LINKUSDT": {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"DOTUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"ATOMUSDT": {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"NEARUSDT": {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"LTCUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"BCHUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 5},
		"TRXUSDT":  {MaxLeverage: 10, MinNotional: 20, Priority: 4},
		"SUIUSDT":  {MaxLeverage: 8, MinNotional: 20, Priority: 4},
		"DOGEUSDT": {MaxLeverage: 8, MinNotional: 20, Priority: 4},
		"default":  {MaxLeverage: 10, MinNotional: 20, Priority: 3},
	}
}

// LookupPairConfig returns the config for a symbol, falling back to "default".
func LookupPairConfig(configs map[string]PairConfig, symbol string) PairConfig {
	if cfg, ok := configs[symbol]; ok {
		return cfg
	}
	return configs["default"]
}
