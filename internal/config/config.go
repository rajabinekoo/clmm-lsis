package config

import (
	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// Config is the complete, non-secret research configuration.
//
// Every parameter capable of changing a scientific result belongs here rather
// than in environment variables or command-specific constants.
type Config struct {
	SchemaVersion       int                       `json:"schema_version"`
	ChainID             uint64                    `json:"chain_id"`
	Environment         EnvironmentConfig         `json:"environment"`
	Artifacts           ArtifactConfig            `json:"artifacts"`
	StructuralStudy     StructuralStudyConfig     `json:"structural_study"`
	WithdrawalStudy     WithdrawalStudyConfig     `json:"withdrawal_study"`
	StatisticalAnalysis StatisticalAnalysisConfig `json:"statistical_analysis"`
	Pools               []PoolConfig              `json:"pools"`
}

type EnvironmentConfig struct {
	EthereumRPCURLEnv string `json:"ethereum_rpc_url_env"`
	DatabaseURLEnv    string `json:"database_url_env"`
}

type ArtifactConfig struct {
	BaseDirectory string `json:"base_directory"`
}

type StructuralStudyConfig struct {
	LookbackBlocks           uint64 `json:"lookback_blocks"`
	StepBlocks               uint64 `json:"step_blocks"`
	MaximumSnapshots         int    `json:"maximum_snapshots"`
	PrimaryPositionLimit     int    `json:"primary_position_limit"`
	RobustnessPositionLimits []int  `json:"robustness_position_limits"`
}

type WithdrawalStudyConfig struct {
	MinimumRemovalFraction    string   `json:"minimum_removal_fraction"`
	PilotEventLimit           int      `json:"pilot_event_limit"`
	PrimaryHorizonBlocks      uint64   `json:"primary_horizon_blocks"`
	RobustnessHorizonBlocks   []uint64 `json:"robustness_horizon_blocks"`
	ContaminationWindowBlocks uint64   `json:"contamination_window_blocks"`
}

type StatisticalAnalysisConfig struct {
	RandomSeed          int64   `json:"random_seed"`
	TrainingFraction    float64 `json:"training_fraction"`
	BootstrapReplicates int     `json:"bootstrap_replicates"`
}

type PoolConfig struct {
	Name                     string      `json:"name"`
	Address                  string      `json:"address"`
	FeePips                  uint32      `json:"fee_pips"`
	TickSpacing              int32       `json:"tick_spacing"`
	StructuralReferenceBlock uint64      `json:"structural_reference_block"`
	Token0                   TokenConfig `json:"token0"`
	Token1                   TokenConfig `json:"token1"`
	TradeGridToken0          []string    `json:"trade_grid_token0"`
}

type TokenConfig struct {
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

func (p PoolConfig) DomainPool() (domain.Pool, error) {
	address, err := domain.ParseAddress(p.Address)
	if err != nil {
		return domain.Pool{}, err
	}

	return domain.Pool{
		Name:        p.Name,
		Address:     address,
		FeePips:     p.FeePips,
		TickSpacing: p.TickSpacing,
		Token0: domain.Token{
			Symbol:   p.Token0.Symbol,
			Decimals: p.Token0.Decimals,
		},
		Token1: domain.Token{
			Symbol:   p.Token1.Symbol,
			Decimals: p.Token1.Decimals,
		},
	}, nil
}
