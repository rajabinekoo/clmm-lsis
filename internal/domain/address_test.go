package domain_test

import (
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestParseAddressCanonicalizesInput(t *testing.T) {
	t.Parallel()

	address, err := domain.ParseAddress(
		"  0x88E6A0C2DDD26FEEB64F039A2C41296FCB3F5640  ",
	)
	if err != nil {
		t.Fatalf("ParseAddress() error = %v", err)
	}

	const expected = "0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640"

	if address.String() != expected {
		t.Fatalf(
			"ParseAddress() = %q, want %q",
			address,
			expected,
		)
	}
}

func TestParseAddressRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
	}{
		{
			name:  "missing prefix",
			value: "88e6a0c2ddd26feeb64f039a2c41296fcb3f5640",
		},
		{
			name:  "too short",
			value: "0x1234",
		},
		{
			name:  "non hexadecimal",
			value: "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseAddress(testCase.value); err == nil {
				t.Fatalf(
					"ParseAddress(%q) expected error",
					testCase.value,
				)
			}
		})
	}
}
