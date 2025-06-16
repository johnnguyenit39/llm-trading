package cronjobs

import (
	"context"
	"fmt"
	"j-ai-trade/brokers/binance"
	quantitativetrading "j-ai-trade/quantitative_trading"
	"j-ai-trade/quantitative_trading/market_analyzer"
	strategies "j-ai-trade/quantitative_trading/strategies"
	"j-ai-trade/telegram"
	converter "j-ai-trade/utils/converter"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type BtcChartObserver struct {
	resultChan      chan string
	stopChan        chan struct{}
	service         *binance.BinanceService
	symbol          string
	marketAnalyzer  *market_analyzer.MarketAnalyzer
	strategyHandler *quantitativetrading.StrategyHandler
	telegramService *telegram.TelegramService
}

func NewBtcChartObserver(service *binance.BinanceService) *BtcChartObserver {
	return &BtcChartObserver{
		resultChan:      make(chan string),
		stopChan:        make(chan struct{}),
		service:         service,
		symbol:          "BTCUSDT",
		marketAnalyzer:  market_analyzer.NewMarketAnalyzer([]strategies.Strategy{}),
		strategyHandler: quantitativetrading.NewStrategyHandler(),
		telegramService: telegram.NewTelegramService(),
	}
}

func (o *BtcChartObserver) StartBtcChartObserver() {
	go o.run()
}

func (o *BtcChartObserver) StopBtcChartObserver() {
	close(o.stopChan)
}

func (o *BtcChartObserver) run() {
	ticker := time.NewTicker(1800 * time.Second)
	defer ticker.Stop()

	// Start a goroutine to listen for results
	go func() {
		for {
			select {
			case result := <-o.resultChan:
				err := o.telegramService.SendMessageToChannel(
					os.Getenv("JONNOZ_TOKEN"),
					os.Getenv("JONNOZ_MARKET_TREND_CHAN"),
					result)
				if err != nil {
					log.Error().Err(err).Msg("Failed to send signal to Telegram")
				}
			case <-o.stopChan:
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			err := o.analyzeBtcMarket(context.Background(), o.symbol, o.service)
			if err != nil {
				log.Error().Err(err).Msg("Failed to analyze market")
			}
		case <-o.stopChan:
			return
		}
	}
}

func (o *BtcChartObserver) analyzeBtcMarket(ctx context.Context, symbol string, service *binance.BinanceService) error {
	// Fetch candle data for different timeframes
	candles5m, err := service.Fetch5mCandles(ctx, symbol, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch 5m candles: %v", err)
	}

	candles15m, err := service.Fetch15mCandles(ctx, symbol, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch 15m candles: %v", err)
	}

	candles1h, err := service.Fetch1hCandles(ctx, symbol, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch 1h candles: %v", err)
	}

	// Log the latest candles
	if len(candles5m) > 0 {
		latest := candles5m[len(candles5m)-1]
		log.Info().
			Str("symbol", symbol).
			Str("timeframe", "5m").
			Float64("open", latest.Open).
			Float64("high", latest.High).
			Float64("low", latest.Low).
			Float64("close", latest.Close).
			Float64("volume", latest.Volume).
			Msg("Latest 5m candle")
	}

	// Convert Binance candles to base candles
	baseCandles5m := converter.ConvertBinanceCandlesToBase(candles5m)
	baseCandles15m := converter.ConvertBinanceCandlesToBase(candles15m)
	baseCandles1h := converter.ConvertBinanceCandlesToBase(candles1h)

	// Analyze market conditions
	analysis, err := o.marketAnalyzer.AnalyzeMarket(baseCandles5m, baseCandles15m, baseCandles1h)
	if err != nil {
		return fmt.Errorf("failed to analyze market: %v", err)
	}

	// Construct message
	msg := fmt.Sprintf("Market Analysis for %s:\n", symbol)
	msg += fmt.Sprintf("Primary Condition: %s\n", analysis.PrimaryCondition)
	msg += fmt.Sprintf("Volatility: %.2f\n", analysis.Volatility)
	msg += fmt.Sprintf("Trend: %.2f\n", analysis.Trend)
	msg += fmt.Sprintf("Volume: %.2f\n", analysis.Volume)
	msg += "Conditions:\n"
	for _, condition := range analysis.Conditions {
		msg += fmt.Sprintf("- %s (Confidence: %.2f)\n", condition.Condition, condition.Confidence)
	}

	// Send message through result channel
	o.resultChan <- msg

	return nil
}
