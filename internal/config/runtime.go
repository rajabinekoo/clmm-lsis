package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// RuntimeSecrets contains environment-specific values that must never be
// committed to the repository.
type RuntimeSecrets struct {
	EthereumRPCURL string
	DatabaseURL    string
}

// LoadRuntimeSecrets resolves every secret required by the complete research
// pipeline.
//
// Commands that require only one dependency should use LoadDatabaseURL or
// LoadEthereumRPCURL instead. This prevents a database-only command from
// unnecessarily requiring an RPC credential.
func LoadRuntimeSecrets(
	cfg Config,
) (RuntimeSecrets, error) {
	ethereumRPCURL, err :=
		LoadEthereumRPCURL(cfg)
	if err != nil {
		return RuntimeSecrets{}, err
	}

	databaseURL, err :=
		LoadDatabaseURL(cfg)
	if err != nil {
		return RuntimeSecrets{}, err
	}

	return RuntimeSecrets{
		EthereumRPCURL: ethereumRPCURL,
		DatabaseURL:    databaseURL,
	}, nil
}

// LoadEthereumRPCURL resolves and validates the configured RPC environment
// variable without reading any database secret.
func LoadEthereumRPCURL(
	cfg Config,
) (string, error) {
	value, err := requiredEnvironmentURL(
		cfg.Environment.EthereumRPCURLEnv,
		map[string]struct{}{
			"http":  {},
			"https": {},
			"ws":    {},
			"wss":   {},
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"ethereum RPC URL: %w",
			err,
		)
	}

	return value, nil
}

// LoadDatabaseURL resolves and validates the configured PostgreSQL environment
// variable without requiring an Ethereum RPC credential.
func LoadDatabaseURL(
	cfg Config,
) (string, error) {
	value, err := requiredEnvironmentURL(
		cfg.Environment.DatabaseURLEnv,
		map[string]struct{}{
			"postgres":   {},
			"postgresql": {},
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"database URL: %w",
			err,
		)
	}

	return value, nil
}

func requiredEnvironmentURL(
	environmentVariable string,
	allowedSchemes map[string]struct{},
) (string, error) {
	value := strings.TrimSpace(
		os.Getenv(environmentVariable),
	)

	if value == "" {
		return "", fmt.Errorf(
			"environment variable %s is not set",
			environmentVariable,
		)
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf(
			"parse environment variable %s: %w",
			environmentVariable,
			err,
		)
	}

	scheme := strings.ToLower(parsed.Scheme)

	if _, allowed := allowedSchemes[scheme]; !allowed {
		return "", fmt.Errorf(
			"environment variable %s uses unsupported URL scheme %q",
			environmentVariable,
			parsed.Scheme,
		)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf(
			"environment variable %s must contain a URL host",
			environmentVariable,
		)
	}

	return value, nil
}
