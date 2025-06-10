package storage

import (
	"context"
	"encoding/json"
	"fmt"
	dto "j-okx-ai/modules/okx/model/dto"
	"j-okx-ai/okx"
	"log"
	"net/http"
	"time"
)

// syncTimeWithOKX checks and logs the time difference between local time and OKX server time
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

	// Parse server timestamp
	serverTime, err := time.Parse(time.RFC3339, timeResp.Data[0].TS)
	if err != nil {
		return err
	}

	// Get local time
	localTime := time.Now().UTC()

	// Calculate time difference
	timeDiff := localTime.Sub(serverTime)
	log.Printf("Time difference between local and OKX server: %v", timeDiff)

	// If time difference is more than 1 second, log a warning
	if timeDiff > time.Second || timeDiff < -time.Second {
		log.Printf("WARNING: Local time is out of sync with OKX server by more than 1 second")
	}

	return nil
}

func (mongodbStore *mongodbStore) GetOkxInfo(ctx context.Context, cond map[string]interface{}) (*dto.OkxInfoResponse, error) {
	// Get the OKX service instance
	okxService := okx.GetInstance()

	// Get account information
	_, rawResponse, err := okxService.GetAccount("USDT")
	if err != nil {
		return nil, err
	}

	// Parse the raw response into OkxInfoResponse
	var response dto.OkxInfoResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}
