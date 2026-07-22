package reconstruction

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

// Apply applies one ordered pool event to the mutable state.
//
// The event cursor is committed only after the complete event transition
// succeeds. Failed events therefore do not advance reconstruction progress.
func (s *MutablePoolState) Apply(
	event domain.PoolEvent,
) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf(
			"apply pool event: %w",
			err,
		)
	}

	if event.PoolAddress() != s.pool.Address {
		return fmt.Errorf(
			"%w: event pool %s does not match state pool %s",
			ErrPoolMismatch,
			event.PoolAddress(),
			s.pool.Address,
		)
	}

	if err := s.validateEventOrder(
		event.Cursor(),
	); err != nil {
		return err
	}

	var err error

	switch payload := event.Payload().(type) {
	case domain.InitializeEvent:
		err = s.applyInitialize(payload)

	case domain.MintEvent:
		err = s.applyMint(payload)

	case domain.BurnEvent:
		err = s.applyBurn(payload)

	case domain.SwapEvent:
		err = s.applySwap(payload)

	default:
		err = fmt.Errorf(
			"unsupported pool event payload %T",
			event.Payload(),
		)
	}

	if err != nil {
		return fmt.Errorf(
			"apply %s event at %s: %w",
			event.Type(),
			event.Cursor(),
			err,
		)
	}

	s.setLastAppliedCursor(
		event.Cursor(),
	)

	return nil
}

func (s *MutablePoolState) applyInitialize(
	event domain.InitializeEvent,
) error {
	if s.initialized {
		return ErrAlreadyInitialized
	}

	if len(s.positions) != 0 ||
		len(s.ticks) != 0 ||
		s.activeLiquidity.Sign() != 0 {
		return fmt.Errorf(
			"uninitialized state contains existing liquidity data",
		)
	}

	derivedTick, err :=
		uniswapv3.GetTickAtSqrtRatio(
			event.SqrtPriceX96(),
		)
	if err != nil {
		return fmt.Errorf(
			"derive initialization tick: %w",
			err,
		)
	}

	if event.Tick() != derivedTick {
		return fmt.Errorf(
			"initialize tick %d does not match sqrt-price-derived tick %d",
			event.Tick(),
			derivedTick,
		)
	}

	s.initialized = true
	s.sqrtPriceX96 = event.SqrtPriceX96()
	s.currentTick = event.Tick()
	s.activeLiquidity = new(big.Int)

	return nil
}

func (s *MutablePoolState) applyMint(
	event domain.MintEvent,
) error {
	if !s.initialized {
		return ErrNotInitialized
	}

	if err := s.validatePositionRange(
		event.TickLower(),
		event.TickUpper(),
	); err != nil {
		return err
	}

	key := event.PositionKey(
		s.pool.Address,
	)

	delta, err := domain.NewMintLiquidityDelta(
		event.Amount(),
	)
	if err != nil {
		return err
	}

	var nextPosition domain.CorePosition

	currentPosition, exists :=
		s.positions[key]

	if exists {
		updated, err :=
			currentPosition.ApplyDelta(delta)
		if err != nil {
			return err
		}

		if updated == nil {
			return fmt.Errorf(
				"positive mint unexpectedly removed position %s",
				key,
			)
		}

		nextPosition = *updated
	} else {
		nextPosition, err =
			domain.NewCorePosition(
				key,
				event.Amount(),
			)
		if err != nil {
			return err
		}
	}

	lowerTick := s.tickOrEmpty(
		event.TickLower(),
	)

	nextLowerTick, err :=
		lowerTick.ApplyPositionDelta(
			delta,
			true,
		)
	if err != nil {
		return fmt.Errorf(
			"apply mint to lower tick %d: %w",
			event.TickLower(),
			err,
		)
	}

	upperTick := s.tickOrEmpty(
		event.TickUpper(),
	)

	nextUpperTick, err :=
		upperTick.ApplyPositionDelta(
			delta,
			false,
		)
	if err != nil {
		return fmt.Errorf(
			"apply mint to upper tick %d: %w",
			event.TickUpper(),
			err,
		)
	}

	nextActiveLiquidity :=
		cloneBigInt(s.activeLiquidity)

	if nextPosition.IsActiveAt(
		s.currentTick,
	) {
		nextActiveLiquidity, err =
			uniswapv3.AddLiquidityDelta(
				s.activeLiquidity,
				event.Amount(),
			)
		if err != nil {
			return fmt.Errorf(
				"apply active mint liquidity: %w",
				err,
			)
		}
	}

	// All computations above succeeded. The transition can now be committed.
	s.positions[key] = nextPosition
	s.setTick(nextLowerTick)
	s.setTick(nextUpperTick)
	s.activeLiquidity = nextActiveLiquidity

	return nil
}

func (s *MutablePoolState) applyBurn(
	event domain.BurnEvent,
) error {
	if !s.initialized {
		return ErrNotInitialized
	}

	if err := s.validatePositionRange(
		event.TickLower(),
		event.TickUpper(),
	); err != nil {
		return err
	}

	key := event.PositionKey(
		s.pool.Address,
	)

	currentPosition, exists :=
		s.positions[key]

	if !exists {
		return fmt.Errorf(
			"%w: %s",
			ErrPositionNotFound,
			key,
		)
	}

	lowerTick, lowerExists :=
		s.ticks[event.TickLower()]

	if !lowerExists {
		return fmt.Errorf(
			"%w: lower tick %d",
			ErrTickNotFound,
			event.TickLower(),
		)
	}

	upperTick, upperExists :=
		s.ticks[event.TickUpper()]

	if !upperExists {
		return fmt.Errorf(
			"%w: upper tick %d",
			ErrTickNotFound,
			event.TickUpper(),
		)
	}

	delta, err := domain.NewBurnLiquidityDelta(
		event.Amount(),
	)
	if err != nil {
		return err
	}

	nextPosition, err :=
		currentPosition.ApplyDelta(delta)
	if err != nil {
		return err
	}

	nextLowerTick, err :=
		lowerTick.ApplyPositionDelta(
			delta,
			true,
		)
	if err != nil {
		return fmt.Errorf(
			"apply burn to lower tick %d: %w",
			event.TickLower(),
			err,
		)
	}

	nextUpperTick, err :=
		upperTick.ApplyPositionDelta(
			delta,
			false,
		)
	if err != nil {
		return fmt.Errorf(
			"apply burn to upper tick %d: %w",
			event.TickUpper(),
			err,
		)
	}

	nextActiveLiquidity :=
		cloneBigInt(s.activeLiquidity)

	if currentPosition.IsActiveAt(
		s.currentTick,
	) {
		negativeAmount := new(big.Int).Neg(
			event.Amount(),
		)

		nextActiveLiquidity, err =
			uniswapv3.AddLiquidityDelta(
				s.activeLiquidity,
				negativeAmount,
			)
		if err != nil {
			return fmt.Errorf(
				"apply active burn liquidity: %w",
				err,
			)
		}
	}

	// Commit the transition only after every invariant has succeeded.
	if nextPosition == nil {
		delete(s.positions, key)
	} else {
		s.positions[key] = *nextPosition
	}

	s.setTick(nextLowerTick)
	s.setTick(nextUpperTick)
	s.activeLiquidity = nextActiveLiquidity

	return nil
}

func (s *MutablePoolState) applySwap(
	event domain.SwapEvent,
) error {
	if !s.initialized {
		return ErrNotInitialized
	}

	if event.Tick() < uniswapv3.MinTick ||
		event.Tick() > uniswapv3.MaxTick {
		return fmt.Errorf(
			"swap tick %d is outside supported range [%d,%d]",
			event.Tick(),
			uniswapv3.MinTick,
			uniswapv3.MaxTick,
		)
	}

	if err := validateSwapTickAndPrice(event); err != nil {
		return err
	}

	if event.ZeroForOne() {
		if event.SqrtPriceX96().Cmp(
			s.sqrtPriceX96,
		) > 0 {
			return fmt.Errorf(
				"%w: zero-for-one swap increased sqrt price from %s to %s",
				ErrInconsistentSwap,
				s.sqrtPriceX96,
				event.SqrtPriceX96(),
			)
		}

		if event.Tick() > s.currentTick {
			return fmt.Errorf(
				"%w: zero-for-one swap increased tick from %d to %d",
				ErrInconsistentSwap,
				s.currentTick,
				event.Tick(),
			)
		}
	} else {
		if event.SqrtPriceX96().Cmp(
			s.sqrtPriceX96,
		) < 0 {
			return fmt.Errorf(
				"%w: one-for-zero swap decreased sqrt price from %s to %s",
				ErrInconsistentSwap,
				s.sqrtPriceX96,
				event.SqrtPriceX96(),
			)
		}

		if event.Tick() < s.currentTick {
			return fmt.Errorf(
				"%w: one-for-zero swap decreased tick from %d to %d",
				ErrInconsistentSwap,
				s.currentTick,
				event.Tick(),
			)
		}
	}

	expectedLiquidity, err :=
		s.activeLiquidityAfterTickMove(
			event.Tick(),
			event.ZeroForOne(),
		)
	if err != nil {
		return err
	}

	if expectedLiquidity.Cmp(
		event.ActiveLiquidity(),
	) != 0 {
		return fmt.Errorf(
			"%w: emitted active liquidity=%s reconstructed=%s",
			ErrInconsistentSwap,
			event.ActiveLiquidity(),
			expectedLiquidity,
		)
	}

	s.sqrtPriceX96 = event.SqrtPriceX96()
	s.currentTick = event.Tick()
	s.activeLiquidity = event.ActiveLiquidity()

	return nil
}

func (s *MutablePoolState) validatePositionRange(
	tickLower int32,
	tickUpper int32,
) error {
	if tickLower < uniswapv3.MinTick {
		return fmt.Errorf(
			"tick lower %d is smaller than minimum tick %d",
			tickLower,
			uniswapv3.MinTick,
		)
	}

	if tickUpper > uniswapv3.MaxTick {
		return fmt.Errorf(
			"tick upper %d exceeds maximum tick %d",
			tickUpper,
			uniswapv3.MaxTick,
		)
	}

	if tickLower >= tickUpper {
		return fmt.Errorf(
			"tick lower %d must be smaller than tick upper %d",
			tickLower,
			tickUpper,
		)
	}

	if tickLower%s.pool.TickSpacing != 0 {
		return fmt.Errorf(
			"tick lower %d is not aligned with tick spacing %d",
			tickLower,
			s.pool.TickSpacing,
		)
	}

	if tickUpper%s.pool.TickSpacing != 0 {
		return fmt.Errorf(
			"tick upper %d is not aligned with tick spacing %d",
			tickUpper,
			s.pool.TickSpacing,
		)
	}

	return nil
}

func validateSwapTickAndPrice(
	event domain.SwapEvent,
) error {
	derivedTick, err :=
		uniswapv3.GetTickAtSqrtRatio(
			event.SqrtPriceX96(),
		)
	if err != nil {
		return fmt.Errorf(
			"derive swap tick from sqrt price: %w",
			err,
		)
	}

	if event.Tick() == derivedTick {
		return nil
	}

	// Immediately after a right-to-left crossing, slot0.tick is tickNext-1
	// while the sqrt price is exactly located on tickNext's boundary.
	if event.Tick() == derivedTick-1 {
		boundaryPrice, err :=
			uniswapv3.GetSqrtRatioAtTick(
				derivedTick,
			)
		if err != nil {
			return err
		}

		if boundaryPrice.Cmp(
			event.SqrtPriceX96(),
		) == 0 {
			return nil
		}
	}

	return fmt.Errorf(
		"%w: swap tick %d does not match sqrt-price-derived tick %d",
		ErrInconsistentSwap,
		event.Tick(),
		derivedTick,
	)
}

// activeLiquidityAfterTickMove derives post-swap liquidity from initialized
// tick crossings rather than scanning every position.
//
// This is critical for replay performance because pools may contain many
// positions and millions of swap events.
func (s *MutablePoolState) activeLiquidityAfterTickMove(
	nextTick int32,
	zeroForOne bool,
) (*big.Int, error) {
	result := cloneBigInt(
		s.activeLiquidity,
	)

	if nextTick == s.currentTick {
		return result, nil
	}

	if zeroForOne {
		if nextTick > s.currentTick {
			return nil, fmt.Errorf(
				"%w: zero-for-one next tick %d exceeds current tick %d",
				ErrInconsistentSwap,
				nextTick,
				s.currentTick,
			)
		}

		firstIndex := sort.Search(
			len(s.tickIndexes),
			func(index int) bool {
				return s.tickIndexes[index] > nextTick
			},
		)

		endIndex := sort.Search(
			len(s.tickIndexes),
			func(index int) bool {
				return s.tickIndexes[index] >
					s.currentTick
			},
		)

		for index := endIndex - 1; index >= firstIndex; index-- {
			tickIndex := s.tickIndexes[index]
			tick := s.ticks[tickIndex]

			nextLiquidity, err :=
				uniswapv3.ApplyLiquidityNet(
					result,
					tick.LiquidityNet(),
					true,
				)
			if err != nil {
				return nil, fmt.Errorf(
					"apply right-to-left crossing at tick %d: %w",
					tickIndex,
					err,
				)
			}

			result = nextLiquidity
		}

		return result, nil
	}

	if nextTick < s.currentTick {
		return nil, fmt.Errorf(
			"%w: one-for-zero next tick %d is below current tick %d",
			ErrInconsistentSwap,
			nextTick,
			s.currentTick,
		)
	}

	firstIndex := sort.Search(
		len(s.tickIndexes),
		func(index int) bool {
			return s.tickIndexes[index] >
				s.currentTick
		},
	)

	endIndex := sort.Search(
		len(s.tickIndexes),
		func(index int) bool {
			return s.tickIndexes[index] > nextTick
		},
	)

	for index := firstIndex; index < endIndex; index++ {
		tickIndex := s.tickIndexes[index]
		tick := s.ticks[tickIndex]

		nextLiquidity, err :=
			uniswapv3.ApplyLiquidityNet(
				result,
				tick.LiquidityNet(),
				false,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"apply left-to-right crossing at tick %d: %w",
				tickIndex,
				err,
			)
		}

		result = nextLiquidity
	}

	return result, nil
}
