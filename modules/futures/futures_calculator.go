package futures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FuturesLeverageRequest represents the input for the leverage calculation
type FuturesLeverageRequest struct {
	TargetProfit  float64 `json:"target_profit" example:"1.0"`
	Capital       float64 `json:"capital" example:"1000"`
	Symbol        string  `json:"symbol" example:"ADAUSDT"`
	PriceMovement float64 `json:"price_movement" example:"0.0001"`
}

// FuturesLeverageResponse represents the output of the leverage calculation
type FuturesLeverageResponse struct {
	Leverage     float64 `json:"leverage" example:"1.6"`
	PositionSize float64 `json:"position_size" example:"1634"`
	CurrentPrice float64 `json:"current_price" example:"0.612"`
}

// BaseApiResponse represents the structure for API responses
type BaseApiResponse struct {
	Success           bool        `json:"success"`
	HttpRequestStatus int         `json:"httpRequestStatus"`
	Message           string      `json:"message"`
	Data              interface{} `json:"data,omitempty"`
}

var defaultTickSize = map[string]float64{
	"BTCUSDT": 0.01,
	"ETHUSDT": 0.01,
	"ADAUSDT": 0.0001,
	// ... add more as needed
}

// fetchBinancePrice fetches the latest price for a symbol from Binance API
func fetchBinancePrice(symbol string) (float64, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(data.Price, 64)
}

// CalculateLeverageAPI godoc
// @Summary      Calculate required leverage for $1 profit per 0.001 price move
// @Description  Returns leverage and position size for given capital and symbol
// @Tags         futures
// @Accept       json
// @Produce      json
// @Param        request body FuturesLeverageRequest true "Capital, Symbol, and Price Movement"
// @Success      200 {object} futures.BaseApiResponse
// @Failure		400 {object} futures.BaseApiResponse "Bad Request"
// @Router       /v1/tool/futures/leverage [post]
func CalculateLeverageAPI() func(*gin.Context) {
	return func(c *gin.Context) {
		var req FuturesLeverageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, BaseApiResponse{
				Success:           false,
				HttpRequestStatus: http.StatusBadRequest,
				Message:           "Invalid request",
			})
			return
		}

		currentPrice, err := fetchBinancePrice(req.Symbol)
		if err != nil {
			c.JSON(http.StatusBadRequest, BaseApiResponse{
				Success:           false,
				HttpRequestStatus: http.StatusBadRequest,
				Message:           "Failed to fetch price",
			})
			return
		}

		priceMovement := req.PriceMovement
		if priceMovement <= 0 {
			if tick, ok := defaultTickSize[req.Symbol]; ok {
				priceMovement = tick
			} else {
				c.JSON(http.StatusBadRequest, BaseApiResponse{
					Success:           false,
					HttpRequestStatus: http.StatusBadRequest,
					Message:           "Invalid price movement and no default for symbol",
				})
				return
			}
		}
		positionSize := req.TargetProfit / (priceMovement * currentPrice)
		leverage := positionSize / req.Capital
		if leverage < 1 {
			leverage = 1
		}

		c.JSON(http.StatusOK, BaseApiResponse{
			Success:           true,
			HttpRequestStatus: http.StatusOK,
			Message:           "Calculation successful",
			Data: FuturesLeverageResponse{
				Leverage:     leverage,
				PositionSize: positionSize,
				CurrentPrice: currentPrice,
			},
		})
	}

}
