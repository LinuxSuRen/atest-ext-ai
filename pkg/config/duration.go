/*
Copyright 2023-2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Duration is a custom type that supports parsing from string in YAML/JSON/TOML
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		// Try to unmarshal as duration directly
		return unmarshal(&d.Duration)
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", s)
	}

	d.Duration = duration
	return nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try to unmarshal as number (nanoseconds)
		var ns int64
		if err := json.Unmarshal(data, &ns); err != nil {
			return err
		}
		d.Duration = time.Duration(ns)
		return nil
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", s)
	}

	d.Duration = duration
	return nil
}

// UnmarshalTOML implements toml.Unmarshaler interface
func (d *Duration) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case string:
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s", v)
		}
		d.Duration = duration
		return nil
	case int64:
		d.Duration = time.Duration(v)
		return nil
	case float64:
		d.Duration = time.Duration(int64(v))
		return nil
	default:
		return fmt.Errorf("cannot unmarshal %T into Duration", data)
	}
}

// MarshalYAML implements yaml.Marshaler interface
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

// MarshalJSON implements json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// MarshalTOML implements toml.Marshaler interface
func (d Duration) MarshalTOML() ([]byte, error) {
	return toml.Marshal(d.Duration.String())
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return d.Duration.String()
}

// Value returns the time.Duration value
func (d Duration) Value() time.Duration {
	return d.Duration
}

// NewDuration creates a Duration from time.Duration
func NewDuration(d time.Duration) Duration {
	return Duration{Duration: d}
}

// ParseDuration creates a Duration from string
func ParseDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return Duration{}, err
	}
	return Duration{Duration: d}, nil
}