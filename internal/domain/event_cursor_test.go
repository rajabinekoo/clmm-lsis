package domain_test

import (
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestEventCursorCompare(t *testing.T) {
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
			},
			right: domain.EventCursor{
				BlockNumber: 101,
			},
			expected: -1,
		},
		{
			name: "earlier transaction",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         20,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 3,
				LogIndex:         1,
			},
			expected: -1,
		},
		{
			name: "earlier log",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
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
			name: "equal",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         20,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         20,
			},
			expected: 0,
		},
		{
			name: "later log",
			left: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         22,
			},
			right: domain.EventCursor{
				BlockNumber:      100,
				TransactionIndex: 2,
				LogIndex:         21,
			},
			expected: 1,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := testCase.left.Compare(testCase.right)

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
