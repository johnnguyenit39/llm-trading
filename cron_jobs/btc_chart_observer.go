package cronjobs

import (
	"context"
	"fmt"
	"j-ai-trade/brokers/binance"
	openai "j-ai-trade/open_ai"
	quantitativetrading "j-ai-trade/quantitative_trading"
	"j-ai-trade/quantitative_trading/market_analyzer"
	strategies "j-ai-trade/quantitative_trading/strategies"
	"j-ai-trade/telegram"
	converter "j-ai-trade/utils/converter"
	"os"
	"strings"
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
	openAiService   *openai.UserService
}

func NewBtcChartObserver(service *binance.BinanceService) *BtcChartObserver {
	openAiRepo := openai.NewOpenAiRepository()
	openAiService := openai.NewOpenAiService(openAiRepo)

	return &BtcChartObserver{
		resultChan:      make(chan string),
		stopChan:        make(chan struct{}),
		service:         service,
		symbol:          "BTCUSDT",
		marketAnalyzer:  market_analyzer.NewMarketAnalyzer([]strategies.Strategy{}),
		strategyHandler: quantitativetrading.NewStrategyHandler(),
		telegramService: telegram.NewTelegramService(),
		openAiService:   openAiService,
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
				// Create new OpenAI thread
				threadID, err := o.openAiService.CreateNewChatThread()
				if err != nil {
					log.Error().Err(err).Msg("Failed to create OpenAI thread")
					continue
				}

				// Send the market analysis to OpenAI
				err = o.openAiService.CreateNewChatMessage(*threadID, "user", result)
				if err != nil {
					log.Error().Err(err).Msg("Failed to send market analysis to OpenAI")
					continue
				}

				// Create a string builder to collect the streamed response
				var responseBuilder strings.Builder

				// Stream the response from OpenAI
				err = o.openAiService.GetStreamMessageThread(
					*threadID,
					&responseBuilder,
					func(fullMessage string) {
						// Send AI response to Telegram
						err := o.telegramService.SendMessageToChannel(
							os.Getenv("JONNOZ_TOKEN"),
							os.Getenv("JONNOZ_MARKET_TREND_CHAN"),
							fullMessage)
						if err != nil {
							log.Error().Err(err).Msg("Failed to send signal to Telegram")
						}
					},
				)
				if err != nil {
					log.Error().Err(err).Msg("Failed to get OpenAI response")
					continue
				}

				// Clean up the thread
				err = o.openAiService.DeleteChatThread(*threadID)
				if err != nil {
					log.Error().Err(err).Msg("Failed to delete OpenAI thread")
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

	// Convert Binance candles to base candles
	baseCandles5m := converter.ConvertBinanceCandlesToBase(candles5m)
	baseCandles15m := converter.ConvertBinanceCandlesToBase(candles15m)
	baseCandles1h := converter.ConvertBinanceCandlesToBase(candles1h)

	// Analyze market conditions
	analysis, err := o.marketAnalyzer.AnalyzeMarket(baseCandles5m, baseCandles15m, baseCandles1h)
	if err != nil {
		return fmt.Errorf("failed to analyze market: %v", err)
	}

	// Construct detailed message
	msg := fmt.Sprintf("Detailed Market Analysis for %s:\n\n", symbol)

	// Primary Market Condition
	msg += fmt.Sprintf("Primary Market Condition: %s\n", analysis.PrimaryCondition)

	// Market Metrics
	msg += "\nMarket Metrics:\n"
	msg += fmt.Sprintf("- Volatility: %.2f\n", analysis.Volatility)
	msg += fmt.Sprintf("- Trend: %.2f\n", analysis.Trend)
	msg += fmt.Sprintf("- Volume: %.2f\n", analysis.Volume)

	// Market Conditions with Confidence
	msg += "\nMarket Conditions:\n"
	for _, condition := range analysis.Conditions {
		msg += fmt.Sprintf("- %s (Confidence: %.2f)\n", condition.Condition, condition.Confidence)
	}

	// Latest Price Data
	latest5m := baseCandles5m[len(baseCandles5m)-1]
	latest15m := baseCandles15m[len(baseCandles15m)-1]
	latest1h := baseCandles1h[len(baseCandles1h)-1]

	msg += "\nLatest Price Data:\n"
	msg += fmt.Sprintf("5m: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Volume=%.2f\n",
		latest5m.Open, latest5m.High, latest5m.Low, latest5m.Close, latest5m.Volume)
	msg += fmt.Sprintf("15m: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Volume=%.2f\n",
		latest15m.Open, latest15m.High, latest15m.Low, latest15m.Close, latest15m.Volume)
	msg += fmt.Sprintf("1h: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Volume=%.2f\n",
		latest1h.Open, latest1h.High, latest1h.Low, latest1h.Close, latest1h.Volume)

	// Price Changes
	msg += "\nPrice Changes:\n"
	msg += fmt.Sprintf("5m Change: %.2f%%\n", ((latest5m.Close-latest5m.Open)/latest5m.Open)*100)
	msg += fmt.Sprintf("15m Change: %.2f%%\n", ((latest15m.Close-latest15m.Open)/latest15m.Open)*100)
	msg += fmt.Sprintf("1h Change: %.2f%%\n", ((latest1h.Close-latest1h.Open)/latest1h.Open)*100)

	// Volume Analysis
	msg += "\nVolume Analysis:\n"
	msg += fmt.Sprintf("5m Volume Change: %.2f%%\n", ((latest5m.Volume-baseCandles5m[len(baseCandles5m)-2].Volume)/baseCandles5m[len(baseCandles5m)-2].Volume)*100)
	msg += fmt.Sprintf("15m Volume Change: %.2f%%\n", ((latest15m.Volume-baseCandles15m[len(baseCandles15m)-2].Volume)/baseCandles15m[len(baseCandles15m)-2].Volume)*100)
	msg += fmt.Sprintf("1h Volume Change: %.2f%%\n", ((latest1h.Volume-baseCandles1h[len(baseCandles1h)-2].Volume)/baseCandles1h[len(baseCandles1h)-2].Volume)*100)

	// Send message through result channel
	o.resultChan <- msg

	return nil
}
