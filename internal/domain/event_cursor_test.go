package domain_test

import (
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestEventCursorCompareUsesBlockAndLogOrder(
	t *testing.T,
) {
	t.Parallel()

	testCases := []struct {
		name     string
		left     domain.EventCursor
		right    domain.EventCursor
		expected int
	}{
		{
			name: "earlier block",
			left: domain.EventCursor{
				BlockNumber: 100,
				LogIndex:    50,
			},
			right: domain.EventCursor{
				BlockNumber: 101,
				LogIndex:    1,
			},
			expected: -1,
		},
		{
			name: "earlier block log",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 10,
				LogIndex:         20,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         21,
			},
			expected: -1,
		},
		{
			name: "same Ethereum log",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         20,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 0,
				LogIndex:         20,
			},
			expected: 0,
		},
		{
			name: "later block log",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 1,
				LogIndex:         22,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 8,
				LogIndex:         21,
			},
			expected: 1,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := testCase.left.Compare(
				testCase.right,
			)

			if actual != testCase.expected {
				t.Fatalf(
					"Compare() = %d, want %d",
					actual,
					testCase.expected,
				)
			}
		})
	}
}

func TestEventCursorSameLogIgnoresTransactionIndex(
	t *testing.T,
) {
	t.Parallel()

	left := domain.EventCursor{
		BlockNumber:      100,
		TransactionIndex: 4,
		LogIndex:         25,
	}

	right := domain.EventCursor{
		BlockNumber:      100,
		TransactionIndex: 0,
		LogIndex:         25,
	}

	if !left.SameLog(right) {
		t.Fatal(
			"SameLog() = false, want true",
		)
	}
}
