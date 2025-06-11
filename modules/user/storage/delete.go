package storage

import (
	"context"
)

func (postgresStore *postgresStore) DeleteUser(ctx context.Context, cond map[string]interface{}) (bool, error) {
	return false, nil
}
