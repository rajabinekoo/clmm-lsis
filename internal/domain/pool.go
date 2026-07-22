package domain

import (
	"fmt"
	"strings"
)

// Token identifies one ERC-20 asset used by a pool.
type Token struct {
	Symbol   string
	Decimals uint8
}

func (t Token) Validate() error {
	symbol := strings.TrimSpace(t.Symbol)

	if symbol == "" {
		return fmt.Errorf("token symbol is required")
	}

	if symbol != strings.ToUpper(symbol) {
		return fmt.Errorf("token symbol %q must be upper-case", symbol)
	}

	if t.Decimals > 36 {
		return fmt.Errorf(
			"token %s decimals must not exceed 36",
			symbol,
		)
	}

	return nil
}

// Pool contains immutable metadata required by the simulator and studies.
type Pool struct {
	Name        string
	Address     Address
	FeePips     uint32
	TickSpacing int32
	Token0      Token
	Token1      Token
}

func (p Pool) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pool name is required")
	}

	if p.Address.IsZero() {
		return fmt.Errorf("pool address is required")
	}

	if p.FeePips == 0 {
		return fmt.Errorf("pool fee pips must be greater than zero")
	}

	if p.TickSpacing <= 0 {
		return fmt.Errorf("pool tick spacing must be greater than zero")
	}

	if err := p.Token0.Validate(); err != nil {
		return fmt.Errorf("pool token0: %w", err)
	}

	if err := p.Token1.Validate(); err != nil {
		return fmt.Errorf("pool token1: %w", err)
	}

	if p.Token0.Symbol == p.Token1.Symbol {
		return fmt.Errorf("pool tokens must have different symbols")
	}

	return nil
}
