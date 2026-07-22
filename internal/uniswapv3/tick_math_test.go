package uniswapv3_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestGetSqrtRatioAtTickKnownValues(
	t *testing.T,
) {
	t.Parallel()

	testCases := []struct {
		name     string
		tick     int32
		expected string
	}{
		{
			name:     "minimum tick",
			tick:     uniswapv3.MinTick,
			expected: "4295128739",
		},
		{
			name:     "negative one",
			tick:     -1,
			expected: "79224201403219477170569942574",
		},
		{
			name:     "zero",
			tick:     0,
			expected: "79228162514264337593543950336",
		},
		{
			name:     "positive one",
			tick:     1,
			expected: "79232123823359799118286999568",
		},
		{
			name:     "maximum tick",
			tick:     uniswapv3.MaxTick,
			expected: "1461446703485210103287273052203988822378723970342",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual, err :=
				uniswapv3.GetSqrtRatioAtTick(
					testCase.tick,
				)
			if err != nil {
				t.Fatalf(
					"GetSqrtRatioAtTick() error = %v",
					err,
				)
			}

			expected := mustBigInt(
				t,
				testCase.expected,
			)

			if actual.Cmp(expected) != 0 {
				t.Fatalf(
					"GetSqrtRatioAtTick(%d) = %s, want %s",
					testCase.tick,
					actual,
					expected,
				)
			}
		})
	}
}

func TestTickMathRoundTrip(
	t *testing.T,
) {
	t.Parallel()

	ticks := []int32{
		uniswapv3.MinTick,
		-500000,
		-1000,
		-1,
		0,
		1,
		1000,
		500000,
		uniswapv3.MaxTick - 1,
	}

	for _, tick := range ticks {
		tick := tick

		t.Run(
			fmt.Sprintf("tick_%d", tick),
			func(t *testing.T) {
				t.Parallel()

				ratio, err :=
					uniswapv3.GetSqrtRatioAtTick(
						tick,
					)
				if err != nil {
					t.Fatalf(
						"GetSqrtRatioAtTick() error = %v",
						err,
					)
				}

				actualTick, err :=
					uniswapv3.GetTickAtSqrtRatio(
						ratio,
					)
				if err != nil {
					t.Fatalf(
						"GetTickAtSqrtRatio() error = %v",
						err,
					)
				}

				if actualTick != tick {
					t.Fatalf(
						"round trip tick = %d, want %d",
						actualTick,
						tick,
					)
				}
			},
		)
	}
}

func TestGetTickAtSqrtRatioReturnsLowerTickBetweenBoundaries(
	t *testing.T,
) {
	t.Parallel()

	const tick int32 = 12345

	lower, err := uniswapv3.GetSqrtRatioAtTick(
		tick,
	)
	if err != nil {
		t.Fatalf(
			"GetSqrtRatioAtTick(lower) error = %v",
			err,
		)
	}

	upper, err := uniswapv3.GetSqrtRatioAtTick(
		tick + 1,
	)
	if err != nil {
		t.Fatalf(
			"GetSqrtRatioAtTick(upper) error = %v",
			err,
		)
	}

	midpoint := new(big.Int).Add(
		lower,
		upper,
	)

	midpoint.Quo(
		midpoint,
		big.NewInt(2),
	)

	actual, err :=
		uniswapv3.GetTickAtSqrtRatio(
			midpoint,
		)
	if err != nil {
		t.Fatalf(
			"GetTickAtSqrtRatio() error = %v",
			err,
		)
	}

	if actual != tick {
		t.Fatalf(
			"GetTickAtSqrtRatio() = %d, want %d",
			actual,
			tick,
		)
	}
}

func TestGetTickAtSqrtRatioReturnsExactBoundaryTick(
	t *testing.T,
) {
	t.Parallel()

	const tick int32 = -202345

	boundary, err :=
		uniswapv3.GetSqrtRatioAtTick(
			tick,
		)
	if err != nil {
		t.Fatalf(
			"GetSqrtRatioAtTick() error = %v",
			err,
		)
	}

	actual, err :=
		uniswapv3.GetTickAtSqrtRatio(
			boundary,
		)
	if err != nil {
		t.Fatalf(
			"GetTickAtSqrtRatio() error = %v",
			err,
		)
	}

	if actual != tick {
		t.Fatalf(
			"GetTickAtSqrtRatio() = %d, want %d",
			actual,
			tick,
		)
	}
}

func TestGetSqrtRatioAtTickRejectsOutOfRangeTicks(
	t *testing.T,
) {
	t.Parallel()

	testCases := []int32{
		uniswapv3.MinTick - 1,
		uniswapv3.MaxTick + 1,
	}

	for _, tick := range testCases {
		tick := tick

		t.Run(
			fmt.Sprintf("tick_%d", tick),
			func(t *testing.T) {
				t.Parallel()

				_, err :=
					uniswapv3.GetSqrtRatioAtTick(
						tick,
					)
				if err == nil {
					t.Fatalf(
						"GetSqrtRatioAtTick(%d) expected error",
						tick,
					)
				}
			},
		)
	}
}

func TestGetTickAtSqrtRatioRejectsInvalidBounds(
	t *testing.T,
) {
	t.Parallel()

	belowMinimum := new(big.Int).Sub(
		uniswapv3.MinSqrtRatio(),
		big.NewInt(1),
	)

	if _, err :=
		uniswapv3.GetTickAtSqrtRatio(
			belowMinimum,
		); err == nil {
		t.Fatal(
			"GetTickAtSqrtRatio(below minimum) expected error",
		)
	}

	if _, err :=
		uniswapv3.GetTickAtSqrtRatio(
			uniswapv3.MaxSqrtRatio(),
		); err == nil {
		t.Fatal(
			"GetTickAtSqrtRatio(maximum) expected error",
		)
	}
}

func mustBigInt(
	t *testing.T,
	value string,
) *big.Int {
	t.Helper()

	parsed, ok := new(big.Int).SetString(
		value,
		10,
	)
	if !ok {
		t.Fatalf(
			"invalid integer test constant %q",
			value,
		)
	}

	return parsed
}
