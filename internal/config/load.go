package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Load reads and validates one complete study configuration.
//
// Unknown JSON fields are rejected to prevent silent configuration mistakes.
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf(
			"open study configuration %q: %w",
			path,
			err,
		)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg Config

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf(
			"decode study configuration %q: %w",
			path,
			err,
		)
	}

	var trailing any

	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf(
				"decode study configuration %q: unexpected trailing JSON value",
				path,
			)
		}

		return Config{}, fmt.Errorf(
			"decode study configuration %q trailing content: %w",
			path,
			err,
		)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf(
			"validate study configuration %q: %w",
			path,
			err,
		)
	}

	return cfg, nil
}
