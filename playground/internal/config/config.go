package config

import (
	"errors"
	"strings"
)

// Environment holds the parsed database connection profiles
type Environment struct {
	DSN        string
	MaxWorkers int
}

// ParseCredentials reads system data tokens and updates configuration references securely via pointers
func ParseCredentials(env *Environment, rawDSN string, workers int) error {
	cleanDSN := strings.TrimSpace(rawDSN)
	if cleanDSN == "" {
		return errors.New("database connection string cannot be empty")
	}

	if workers <= 0 {
		return errors.New("worker pool allocation must be a positive integer")
	}

	// Mutate the original configuration data directly via pointer references
	env.DSN = cleanDSN
	env.MaxWorkers = workers
	return nil
}
