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

func LoadRuntimeSecrets(cfg Config) (RuntimeSecrets, error) {
	rpcURL, err := requiredEnvironmentURL(
		cfg.Environment.EthereumRPCURLEnv,
		map[string]struct{}{
			"http":  {},
			"https": {},
			"ws":    {},
			"wss":   {},
		},
	)
	if err != nil {
		return RuntimeSecrets{}, fmt.Errorf("ethereum RPC URL: %w", err)
	}

	databaseURL, err := requiredEnvironmentURL(
		cfg.Environment.DatabaseURLEnv,
		map[string]struct{}{
			"postgres":   {},
			"postgresql": {},
		},
	)
	if err != nil {
		return RuntimeSecrets{}, fmt.Errorf("database URL: %w", err)
	}

	return RuntimeSecrets{
		EthereumRPCURL: rpcURL,
		DatabaseURL:    databaseURL,
	}, nil
}

func requiredEnvironmentURL(
	environmentVariable string,
	allowedSchemes map[string]struct{},
) (string, error) {
	value := strings.TrimSpace(os.Getenv(environmentVariable))

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
