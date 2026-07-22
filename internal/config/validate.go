package config

import (
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	environmentVariablePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	poolNamePattern            = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)
	decimalAmountPattern       = regexp.MustCompile(`^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)
)

func (c Config) Validate() error {
	var validationErrors []error

	if c.SchemaVersion != 1 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("schema_version must equal 1"),
		)
	}

	if c.ChainID == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("chain_id must be greater than zero"),
		)
	}

	validationErrors = append(
		validationErrors,
		validateEnvironment(c.Environment)...,
	)

	validationErrors = append(
		validationErrors,
		validateArtifacts(c.Artifacts)...,
	)

	validationErrors = append(
		validationErrors,
		validateStructuralStudy(c.StructuralStudy)...,
	)

	validationErrors = append(
		validationErrors,
		validateWithdrawalStudy(c.WithdrawalStudy)...,
	)

	validationErrors = append(
		validationErrors,
		validateStatisticalAnalysis(c.StatisticalAnalysis)...,
	)

	validationErrors = append(
		validationErrors,
		validatePools(c.Pools)...,
	)

	return errors.Join(validationErrors...)
}

func validateEnvironment(cfg EnvironmentConfig) []error {
	var validationErrors []error

	if !environmentVariablePattern.MatchString(cfg.EthereumRPCURLEnv) {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"environment.ethereum_rpc_url_env %q is not a valid environment variable name",
				cfg.EthereumRPCURLEnv,
			),
		)
	}

	if !environmentVariablePattern.MatchString(cfg.DatabaseURLEnv) {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"environment.database_url_env %q is not a valid environment variable name",
				cfg.DatabaseURLEnv,
			),
		)
	}

	if cfg.EthereumRPCURLEnv == cfg.DatabaseURLEnv {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"Ethereum RPC and database URLs must use different environment variables",
			),
		)
	}

	return validationErrors
}

func validateArtifacts(cfg ArtifactConfig) []error {
	baseDirectory := strings.TrimSpace(cfg.BaseDirectory)

	if baseDirectory == "" {
		return []error{
			fmt.Errorf("artifacts.base_directory is required"),
		}
	}

	if filepath.IsAbs(baseDirectory) {
		return []error{
			fmt.Errorf(
				"artifacts.base_directory must be relative to the repository",
			),
		}
	}

	cleaned := filepath.Clean(baseDirectory)

	if cleaned == "." || cleaned == ".." ||
		strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return []error{
			fmt.Errorf(
				"artifacts.base_directory must remain inside the repository",
			),
		}
	}

	return nil
}

func validateStructuralStudy(cfg StructuralStudyConfig) []error {
	var validationErrors []error

	if cfg.LookbackBlocks == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("structural_study.lookback_blocks must be greater than zero"),
		)
	}

	if cfg.StepBlocks == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("structural_study.step_blocks must be greater than zero"),
		)
	}

	if cfg.MaximumSnapshots <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("structural_study.maximum_snapshots must be greater than zero"),
		)
	}

	if cfg.PrimaryPositionLimit <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"structural_study.primary_position_limit must be greater than zero",
			),
		)
	}

	if len(cfg.RobustnessPositionLimits) == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"structural_study.robustness_position_limits must not be empty",
			),
		)
	} else {
		seen := make(map[int]struct{}, len(cfg.RobustnessPositionLimits))
		containsPrimary := false

		for index, limit := range cfg.RobustnessPositionLimits {
			if limit <= 0 {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"structural_study.robustness_position_limits[%d] must be greater than zero",
						index,
					),
				)
			}

			if _, exists := seen[limit]; exists {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"structural_study.robustness_position_limits contains duplicate value %d",
						limit,
					),
				)
			}

			seen[limit] = struct{}{}

			if limit == cfg.PrimaryPositionLimit {
				containsPrimary = true
			}
		}

		if !containsPrimary {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"structural_study.robustness_position_limits must include primary_position_limit",
				),
			)
		}
	}

	return validationErrors
}

func validateWithdrawalStudy(cfg WithdrawalStudyConfig) []error {
	var validationErrors []error

	minimumFraction, err := parsePositiveDecimal(
		cfg.MinimumRemovalFraction,
	)
	if err != nil {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"withdrawal_study.minimum_removal_fraction: %w",
				err,
			),
		)
	} else if minimumFraction.Cmp(big.NewRat(1, 1)) > 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"withdrawal_study.minimum_removal_fraction must not exceed 1",
			),
		)
	}

	if cfg.PilotEventLimit <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"withdrawal_study.pilot_event_limit must be greater than zero",
			),
		)
	}

	if cfg.PrimaryHorizonBlocks == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"withdrawal_study.primary_horizon_blocks must be greater than zero",
			),
		)
	}

	if cfg.ContaminationWindowBlocks == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"withdrawal_study.contamination_window_blocks must be greater than zero",
			),
		)
	}

	seenHorizons := make(map[uint64]struct{})

	for index, horizon := range cfg.RobustnessHorizonBlocks {
		if horizon == 0 {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"withdrawal_study.robustness_horizon_blocks[%d] must be greater than zero",
					index,
				),
			)
		}

		if horizon == cfg.PrimaryHorizonBlocks {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"withdrawal_study.robustness_horizon_blocks must not repeat primary_horizon_blocks",
				),
			)
		}

		if _, exists := seenHorizons[horizon]; exists {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"withdrawal_study.robustness_horizon_blocks contains duplicate value %d",
					horizon,
				),
			)
		}

		seenHorizons[horizon] = struct{}{}
	}

	return validationErrors
}

func validateStatisticalAnalysis(
	cfg StatisticalAnalysisConfig,
) []error {
	var validationErrors []error

	if cfg.TrainingFraction <= 0 || cfg.TrainingFraction >= 1 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"statistical_analysis.training_fraction must be between 0 and 1",
			),
		)
	}

	if cfg.BootstrapReplicates <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"statistical_analysis.bootstrap_replicates must be greater than zero",
			),
		)
	}

	return validationErrors
}

func validatePools(pools []PoolConfig) []error {
	if len(pools) == 0 {
		return []error{
			fmt.Errorf("pools must not be empty"),
		}
	}

	var validationErrors []error

	names := make(map[string]struct{}, len(pools))
	addresses := make(map[string]struct{}, len(pools))

	for index, pool := range pools {
		prefix := fmt.Sprintf("pools[%d]", index)

		if !poolNamePattern.MatchString(pool.Name) {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s.name %q must use lower-case snake_case",
					prefix,
					pool.Name,
				),
			)
		}

		if _, exists := names[pool.Name]; exists {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("duplicate pool name %q", pool.Name),
			)
		}

		names[pool.Name] = struct{}{}

		domainPool, err := pool.DomainPool()
		if err != nil {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("%s: %w", prefix, err),
			)
		} else {
			if _, exists := addresses[domainPool.Address.String()]; exists {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"duplicate pool address %s",
						domainPool.Address,
					),
				)
			}

			addresses[domainPool.Address.String()] = struct{}{}

			if err := domainPool.Validate(); err != nil {
				validationErrors = append(
					validationErrors,
					fmt.Errorf("%s: %w", prefix, err),
				)
			}
		}

		expectedTickSpacing, supported :=
			expectedTickSpacingForFee(pool.FeePips)

		if !supported {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s.fee_pips %d is not a supported Uniswap v3 fee tier",
					prefix,
					pool.FeePips,
				),
			)
		} else if pool.TickSpacing != expectedTickSpacing {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s.tick_spacing is %d, want %d for fee tier %d",
					prefix,
					pool.TickSpacing,
					expectedTickSpacing,
					pool.FeePips,
				),
			)
		}

		if pool.StructuralReferenceBlock == 0 {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s.structural_reference_block must be greater than zero",
					prefix,
				),
			)
		}

		validationErrors = append(
			validationErrors,
			validateTradeGrid(
				prefix+".trade_grid_token0",
				pool.TradeGridToken0,
				pool.Token0.Decimals,
			)...,
		)
	}

	return validationErrors
}

func validateTradeGrid(
	fieldName string,
	values []string,
	tokenDecimals uint8,
) []error {
	if len(values) < 2 {
		return []error{
			fmt.Errorf("%s must contain at least two amounts", fieldName),
		}
	}

	var validationErrors []error
	var previous *big.Rat

	for index, value := range values {
		if decimalPlaces(value) > int(tokenDecimals) {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s[%d] has more than %d decimal places",
					fieldName,
					index,
					tokenDecimals,
				),
			)
		}

		parsed, err := parsePositiveDecimal(value)
		if err != nil {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s[%d]: %w",
					fieldName,
					index,
					err,
				),
			)

			continue
		}

		if previous != nil && parsed.Cmp(previous) <= 0 {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"%s must be strictly increasing at index %d",
					fieldName,
					index,
				),
			)
		}

		previous = parsed
	}

	return validationErrors
}

func parsePositiveDecimal(value string) (*big.Rat, error) {
	trimmed := strings.TrimSpace(value)

	if !decimalAmountPattern.MatchString(trimmed) {
		return nil, fmt.Errorf(
			"%q is not a plain non-negative decimal number",
			value,
		)
	}

	parsed, ok := new(big.Rat).SetString(trimmed)
	if !ok {
		return nil, fmt.Errorf("cannot parse decimal value %q", value)
	}

	if parsed.Sign() <= 0 {
		return nil, fmt.Errorf("decimal value %q must be greater than zero", value)
	}

	return parsed, nil
}

func decimalPlaces(value string) int {
	trimmed := strings.TrimSpace(value)
	separatorIndex := strings.IndexByte(trimmed, '.')

	if separatorIndex < 0 {
		return 0
	}

	return len(trimmed) - separatorIndex - 1
}

func expectedTickSpacingForFee(feePips uint32) (int32, bool) {
	switch feePips {
	case 100:
		return 1, true
	case 500:
		return 10, true
	case 3000:
		return 60, true
	case 10000:
		return 200, true
	default:
		return 0, false
	}
}
