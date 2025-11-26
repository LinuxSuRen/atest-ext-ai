package config

import "errors"

var (
	// ErrConfigNotFound indicates no configuration file could be located.
	ErrConfigNotFound = errors.New("configuration file not found")
	// ErrConfigWarning represents recoverable configuration issues.
	ErrConfigWarning = errors.New("configuration warning")
)
