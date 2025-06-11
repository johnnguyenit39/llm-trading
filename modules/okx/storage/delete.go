package storage

import (
	"context"
)

func (postgresStore *postgresStore) DeleteSubscription(ctx context.Context, cond map[string]interface{}) (bool, error) {
	return false, nil
}
