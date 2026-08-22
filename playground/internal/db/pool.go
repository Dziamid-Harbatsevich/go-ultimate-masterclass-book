package db

import (
	"errors"
	"fmt"
)

// LocalPool stub simulates a production thread connection pool container
type LocalPool struct {
	ConnectionString string
	IsConnected      bool
}

// BootPool validates parameters and sets up an isolated database connection plane
func BootPool(dsn string) (*LocalPool, error) {
	if dsn == "" {
		return nil, errors.New("cannot boot connection pool with an empty DSN profile")
	}

	fmt.Printf("[DB Internal] Connecting to database gateway via: %s...\n", dsn)

	return &LocalPool{
		ConnectionString: dsn,
		IsConnected:      true,
	}, nil
}
