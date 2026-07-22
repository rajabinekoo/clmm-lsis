package reconstruction

import "errors"

var (
	// ErrPoolMismatch indicates that an event or snapshot belongs to a
	// different pool than the reconstruction state.
	ErrPoolMismatch = errors.New("pool mismatch")

	// ErrOutOfOrder indicates that an event does not occur strictly after the
	// last event already applied to the state.
	ErrOutOfOrder = errors.New("event out of order")

	// ErrNotInitialized indicates that a liquidity or swap event was observed
	// before the pool initialization event.
	ErrNotInitialized = errors.New("pool is not initialized")

	// ErrAlreadyInitialized indicates that Initialize was applied more than
	// once.
	ErrAlreadyInitialized = errors.New("pool is already initialized")

	// ErrPositionNotFound indicates that a Burn refers to an unknown core
	// position.
	ErrPositionNotFound = errors.New("position not found")

	// ErrTickNotFound indicates that a Burn refers to a boundary that is not
	// present in reconstructed tick state.
	ErrTickNotFound = errors.New("tick not found")

	// ErrInconsistentSwap indicates that the post-swap state emitted on-chain
	// is inconsistent with the reconstructed liquidity geometry.
	ErrInconsistentSwap = errors.New("inconsistent swap state")

	// ErrInvalidSnapshotReference indicates that a requested snapshot
	// reference is earlier than the current reconstructed state.
	ErrInvalidSnapshotReference = errors.New(
		"invalid snapshot reference",
	)
)
