package okx

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Account represents an OKX account balance
type Account struct {
	Coin             string  `json:"coin,omitempty"`
	Balance          float64 `json:"balance,omitempty"`
	AvailableBalance float64 `json:"available_balance,omitempty"`
	FrozenBalance    float64 `json:"frozen_balance,omitempty"`
}

// OrderSide represents the side of an order (buy/sell)
type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

// CurrencyPair represents a trading pair
type CurrencyPair struct {
	BaseSymbol  string
	QuoteSymbol string
	Symbol      string
}

var (
	okxClient       *OKXService
	once            sync.Once
	timeOffset      time.Duration
	timeOffsetMutex sync.RWMutex
)

// OKXService represents the OKX API service
type OKXService struct {
	apiKey     string
	apiSecret  string
	passphrase string
	client     *http.Client
	baseURL    string
}

// syncTimeWithOKX synchronizes the local time with OKX server time
func syncTimeWithOKX() error {
	// Make a request to OKX's time endpoint
	resp, err := http.Get("https://www.okx.com/api/v5/public/time")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse the response to get server time
	var timeResp struct {
		Code string `json:"code"`
		Data []struct {
			TS string `json:"ts"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&timeResp); err != nil {
		return err
	}

	if len(timeResp.Data) == 0 {
		return fmt.Errorf("no time data in response")
	}

	// Parse server timestamp (Unix milliseconds)
	ts, err := strconv.ParseInt(timeResp.Data[0].TS, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse timestamp: %v", err)
	}

	// Convert Unix milliseconds to time.Time
	serverTime := time.Unix(0, ts*int64(time.Millisecond))

	// Get local time
	localTime := time.Now().UTC()

	// Calculate time difference
	timeDiff := localTime.Sub(serverTime)

	// Store the time offset
	timeOffsetMutex.Lock()
	timeOffset = timeDiff
	timeOffsetMutex.Unlock()

	log.Info().Dur("time_diff", timeDiff).Msg("Time difference between local and OKX server")

	// If time difference is more than 1 second, log a warning
	if timeDiff > time.Second || timeDiff < -time.Second {
		log.Warn().Dur("time_diff", timeDiff).Msg("Local time is out of sync with OKX server by more than 1 second")
	}

	return nil
}

// getAdjustedTime returns the current time adjusted by the OKX server offset
func getAdjustedTime() time.Time {
	timeOffsetMutex.RLock()
	defer timeOffsetMutex.RUnlock()
	return time.Now().UTC().Add(-timeOffset)
}

// GetInstance returns the singleton instance of OKXService
func GetInstance() *OKXService {
	once.Do(func() {
		apiKey := os.Getenv("API_KEY")
		apiSecret := os.Getenv("API_SECRET_KEY")
		passphrase := os.Getenv("API_PASSPHRASE")

		if apiKey == "" || apiSecret == "" || passphrase == "" {
			log.Info().Msg("OKX API credentials not found in environment variables")
		}

		okxClient = &OKXService{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			passphrase: passphrase,
			client:     &http.Client{Timeout: 10 * time.Second},
			baseURL:    "https://www.okx.com",
		}

		// Initial time sync
		if err := syncTimeWithOKX(); err != nil {
			log.Error().Err(err).Msg("Failed to sync time with OKX")
		}

		// Start periodic time synchronization
		go func() {
			for {
				if err := syncTimeWithOKX(); err != nil {
					log.Error().Err(err).Msg("Failed to sync time with OKX")
				}
				time.Sleep(30 * time.Second) // Sync every 30 seconds
			}
		}()
	})
	return okxClient
}

// GetAccount retrieves the account information for a specific currency
func (s *OKXService) GetAccount(currency string) (map[string]Account, []byte, error) {
	// Sync time with OKX before making API call
	if err := syncTimeWithOKX(); err != nil {
		return nil, nil, err
	}

	// Get the current timestamp in ISO 8601 format with milliseconds
	timestamp := getAdjustedTime().Format("2006-01-02T15:04:05.000Z")

	// Prepare the request
	url := "https://www.okx.com/api/v5/account/balance"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	// Add required headers
	req.Header.Set("OK-ACCESS-KEY", s.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", s.generateSign(timestamp, "GET", "/api/v5/account/balance", ""))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", s.passphrase)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Read the raw response for logging
	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	// Parse the response
	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Details []struct {
				Ccy       string `json:"ccy"`
				Eq        string `json:"eq"`
				CashBal   string `json:"cashBal"`
				FrozenBal string `json:"frozenBal"`
				AvailBal  string `json:"availBal"`
			} `json:"details"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rawResponse, &result); err != nil {
		return nil, rawResponse, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for API error
	if result.Code != "0" {
		return nil, rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	// Convert the response to the expected format
	accounts := make(map[string]Account)
	for _, data := range result.Data {
		for _, detail := range data.Details {
			// Convert string balances to float64
			eq, _ := strconv.ParseFloat(detail.Eq, 64)
			frozenBal, _ := strconv.ParseFloat(detail.FrozenBal, 64)
			availBal, _ := strconv.ParseFloat(detail.AvailBal, 64)

			accounts[detail.Ccy] = Account{
				Coin:             detail.Ccy,
				Balance:          eq,
				AvailableBalance: availBal,
				FrozenBalance:    frozenBal,
			}
		}
	}

	return accounts, rawResponse, nil
}

// CreateOrder creates a new order
func (s *OKXService) CreateOrder(pair CurrencyPair, amount, price float64, side OrderSide, orderType OrderType) ([]byte, error) {
	// Sync time with OKX before making API call
	if err := syncTimeWithOKX(); err != nil {
		return nil, err
	}

	// Get the current timestamp in ISO 8601 format with milliseconds
	timestamp := getAdjustedTime().Format("2006-01-02T15:04:05.000Z")

	// Prepare the request body
	orderData := map[string]interface{}{
		"instId":  pair.Symbol,
		"tdMode":  "cash",
		"side":    string(side),
		"ordType": string(orderType),
		"sz":      fmt.Sprintf("%f", amount),
		"px":      fmt.Sprintf("%f", price),
	}

	body, err := json.Marshal(orderData)
	if err != nil {
		return nil, err
	}

	// Prepare the request
	url := "https://www.okx.com/api/v5/trade/order"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Add required headers
	req.Header.Set("OK-ACCESS-KEY", s.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", s.generateSign(timestamp, "POST", "/api/v5/trade/order", string(body)))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", s.passphrase)
	req.Header.Set("x-simulated-trading", "1")
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(rawResponse, &result); err != nil {
		return rawResponse, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for API error
	if result.Code != "0" {
		return rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	return rawResponse, nil
}

// NewCurrencyPair creates a new currency pair
func (s *OKXService) NewCurrencyPair(base, quote string) CurrencyPair {
	return CurrencyPair{
		BaseSymbol:  base,
		QuoteSymbol: quote,
		Symbol:      base + "-" + quote,
	}
}

// generateSign generates the signature for OKX API requests
func (s *OKXService) generateSign(timestamp, method, requestPath, body string) string {
	message := timestamp + method + requestPath + body
	mac := hmac.New(sha256.New, []byte(s.apiSecret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (s *OKXService) CancelOrder(orderID string, instId string) ([]byte, error) {
	// Sync time with OKX before making API call
	if err := syncTimeWithOKX(); err != nil {
		return nil, err
	}

	// Get the current timestamp in ISO 8601 format with milliseconds
	timestamp := getAdjustedTime().Format("2006-01-02T15:04:05.000Z")

	// Prepare the request body
	orderData := map[string]string{
		"ordId":  orderID,
		"instId": instId,
	}

	body, err := json.Marshal(orderData)
	if err != nil {
		return nil, err
	}

	// Prepare the request
	url := "https://www.okx.com/api/v5/trade/cancel-order"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Add required headers
	req.Header.Set("OK-ACCESS-KEY", s.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", s.generateSign(timestamp, "POST", "/api/v5/trade/cancel-order", string(body)))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", s.passphrase)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(rawResponse, &result); err != nil {
		return rawResponse, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for API error
	if result.Code != "0" {
		return rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	return rawResponse, nil
}
