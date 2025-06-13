package repository

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

	"j-ai-trade/brokers/okx/types"

	"github.com/rs/zerolog/log"
)

type okxRepositoryImpl struct {
	apiKey     string
	apiSecret  string
	passphrase string
	client     *http.Client
	baseURL    string
	timeOffset time.Duration
	timeMutex  sync.RWMutex
}

func NewOKXRepository() OKXRepository {
	return &okxRepositoryImpl{
		apiKey:     os.Getenv("API_KEY"),
		apiSecret:  os.Getenv("API_SECRET_KEY"),
		passphrase: os.Getenv("API_PASSPHRASE"),
		client:     &http.Client{Timeout: 10 * time.Second},
		baseURL:    "https://www.okx.com",
	}
}

func (r *okxRepositoryImpl) GetAccount(currency string) (map[string]types.Account, []byte, error) {
	if err := r.SyncTimeWithOKX(); err != nil {
		return nil, nil, err
	}

	timestamp := r.GetAdjustedTime().Format("2006-01-02T15:04:05.000Z")
	url := "https://www.okx.com/api/v5/account/balance"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("OK-ACCESS-KEY", r.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", r.GenerateSign(timestamp, "GET", "/api/v5/account/balance", ""))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", r.passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

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

	if result.Code != "0" {
		return nil, rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	accounts := make(map[string]types.Account)
	for _, data := range result.Data {
		for _, detail := range data.Details {
			eq, _ := strconv.ParseFloat(detail.Eq, 64)
			frozenBal, _ := strconv.ParseFloat(detail.FrozenBal, 64)
			availBal, _ := strconv.ParseFloat(detail.AvailBal, 64)

			accounts[detail.Ccy] = types.Account{
				Coin:             detail.Ccy,
				Balance:          eq,
				AvailableBalance: availBal,
				FrozenBalance:    frozenBal,
			}
		}
	}

	return accounts, rawResponse, nil
}

func (r *okxRepositoryImpl) CreateSpotOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType) ([]byte, error) {
	if err := r.SyncTimeWithOKX(); err != nil {
		return nil, err
	}

	timestamp := r.GetAdjustedTime().Format("2006-01-02T15:04:05.000Z")
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

	url := "https://www.okx.com/api/v5/trade/order"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("OK-ACCESS-KEY", r.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", r.GenerateSign(timestamp, "POST", "/api/v5/trade/order", string(body)))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", r.passphrase)
	req.Header.Set("x-simulated-trading", "1")
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(rawResponse, &result); err != nil {
		return rawResponse, fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Code != "0" {
		return rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	return rawResponse, nil
}

func (r *okxRepositoryImpl) CancelSpotOrder(orderID string, instId string) ([]byte, error) {
	if err := r.SyncTimeWithOKX(); err != nil {
		return nil, err
	}

	timestamp := r.GetAdjustedTime().Format("2006-01-02T15:04:05.000Z")
	orderData := map[string]string{
		"ordId":  orderID,
		"instId": instId,
	}

	body, err := json.Marshal(orderData)
	if err != nil {
		return nil, err
	}

	url := "https://www.okx.com/api/v5/trade/cancel-order"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("OK-ACCESS-KEY", r.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", r.GenerateSign(timestamp, "POST", "/api/v5/trade/cancel-order", string(body)))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", r.passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(rawResponse, &result); err != nil {
		return rawResponse, fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Code != "0" {
		return rawResponse, fmt.Errorf("API error: %s", result.Msg)
	}

	return rawResponse, nil
}

func (r *okxRepositoryImpl) SyncTimeWithOKX() error {
	resp, err := http.Get("https://www.okx.com/api/v5/public/time")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	ts, err := strconv.ParseInt(timeResp.Data[0].TS, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse timestamp: %v", err)
	}

	serverTime := time.Unix(0, ts*int64(time.Millisecond))
	localTime := time.Now().UTC()
	timeDiff := localTime.Sub(serverTime)

	r.timeMutex.Lock()
	r.timeOffset = timeDiff
	r.timeMutex.Unlock()

	log.Info().Dur("time_diff", timeDiff).Msg("Time difference between local and OKX server")

	if timeDiff > time.Second || timeDiff < -time.Second {
		log.Warn().Dur("time_diff", timeDiff).Msg("Local time is out of sync with OKX server by more than 1 second")
	}

	return nil
}

func (r *okxRepositoryImpl) GetAdjustedTime() time.Time {
	r.timeMutex.RLock()
	defer r.timeMutex.RUnlock()
	return time.Now().UTC().Add(-r.timeOffset)
}

func (r *okxRepositoryImpl) GenerateSign(timestamp, method, requestPath, body string) string {
	message := timestamp + method + requestPath + body
	mac := hmac.New(sha256.New, []byte(r.apiSecret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
