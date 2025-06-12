package common

import (
	"fmt"
	"time"

	"golang.org/x/exp/rand"
)

func GenerateRandomCode() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
