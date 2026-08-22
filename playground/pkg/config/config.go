package config

import (
	"errors"
	"strings"
)

// TargetProfile stores configuration rules for target systems
type TargetProfile struct {
	DSN     string
	APIPort string
}

// LoadProfile reads configuration inputs and returns a formatted TargetProfile pointer
func LoadProfile(rawDSN, port string) (*TargetProfile, error) {
	cleanDSN := strings.TrimSpace(rawDSN)
	if cleanDSN == "" {
		return nil, errors.New("configuration validation failed: database connection string is empty")
	}

	cleanPort := strings.TrimSpace(port)
	if !strings.HasPrefix(cleanPort, ":") {
		cleanPort = ":" + cleanPort
	}

	return &TargetProfile{
		DSN:     cleanDSN,
		APIPort: cleanPort,
	}, nil
}
