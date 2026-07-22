package storage

import "errors"

var (
	ErrRecordNotFound = errors.New(
		"storage record not found",
	)

	ErrInvalidLegacyRecord = errors.New(
		"invalid legacy storage record",
	)

	ErrMissingPositionOwner = errors.New(
		"liquidity action position owner is missing",
	)

	ErrInvalidLiquiditySign = errors.New(
		"liquidity action has invalid delta sign",
	)

	ErrDuplicateEventCursor = errors.New(
		"duplicate event cursor",
	)

	ErrCheckpointMismatch = errors.New(
		"checkpoint state mismatch",
	)

	ErrSchemaIncompatible = errors.New(
		"database schema is incompatible",
	)

	ErrSwapTableUnavailable = errors.New(
		"pool swaps table is unavailable",
	)

	ErrSwapIndexRangeNotFound = errors.New(
		"swap index range not found",
	)

	ErrSwapIndexProgressConflict = errors.New(
		"swap index progress conflict",
	)

	ErrStoredSwapConflict = errors.New(
		"stored swap conflicts with existing event",
	)

	ErrInvalidSwapBatch = errors.New(
		"invalid swap batch",
	)

	ErrChainReorganization = errors.New(
		"chain reorganization detected",
	)
)
