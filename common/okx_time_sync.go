package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	timeOffset      time.Duration
	timeOffsetMutex sync.RWMutex
)

// SyncTimeWithOKX checks and logs the time difference between local time and OKX server time
func SyncTimeWithOKX() error {
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

	// Parse server timestamp (Unix millisecond format)
	serverTimeMs, err := strconv.ParseInt(timeResp.Data[0].TS, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse server timestamp: %v", err)
	}

	// Convert milliseconds to time.Time
	serverTime := time.Unix(0, serverTimeMs*int64(time.Millisecond))

	// Get local time
	localTime := time.Now().UTC()

	// Calculate time difference
	timeDiff := localTime.Sub(serverTime)

	// Store the time offset
	timeOffsetMutex.Lock()
	timeOffset = timeDiff
	timeOffsetMutex.Unlock()

	log.Printf("Time difference between local and OKX server: %v", timeDiff)

	// If time difference is more than 1 second, log a warning
	if timeDiff > time.Second || timeDiff < -time.Second {
		log.Printf("WARNING: Local time is out of sync with OKX server by more than 1 second")
	}

	return nil
}

// GetAdjustedTime returns the current time adjusted by the OKX server offset
func GetAdjustedTime() time.Time {
	timeOffsetMutex.RLock()
	defer timeOffsetMutex.RUnlock()
	return time.Now().UTC().Add(-timeOffset)
}
