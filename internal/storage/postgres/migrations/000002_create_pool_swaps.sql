CREATE TABLE IF NOT EXISTS public.pool_swaps
(
    pool_address      CHAR(42)       NOT NULL,
    block_number      BIGINT         NOT NULL,
    block_hash        CHAR(66)       NOT NULL,
    transaction_hash  CHAR(66)       NOT NULL,
    transaction_index INTEGER        NOT NULL,
    log_index         INTEGER        NOT NULL,
    timestamp         TIMESTAMPTZ    NOT NULL,

    sender            CHAR(42)       NOT NULL,
    recipient         CHAR(42)       NOT NULL,

    amount0_raw       NUMERIC(78, 0) NOT NULL,
    amount1_raw       NUMERIC(78, 0) NOT NULL,

    sqrt_price_x96    NUMERIC(49, 0) NOT NULL,
    active_liquidity  NUMERIC(39, 0) NOT NULL,
    tick              INTEGER        NOT NULL,

    inserted_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT pool_swaps_pkey
        PRIMARY KEY
            (
             pool_address,
             block_number,
             log_index
                ),

    CONSTRAINT pool_swaps_transaction_log_unique
        UNIQUE
            (
             transaction_hash,
             log_index
                ),

    CONSTRAINT pool_swaps_pool_fkey
        FOREIGN KEY (pool_address)
            REFERENCES public.pools (address)
            ON UPDATE RESTRICT
            ON DELETE RESTRICT,

    CONSTRAINT pool_swaps_block_number_positive
        CHECK (block_number > 0),

    CONSTRAINT pool_swaps_transaction_index_non_negative
        CHECK (transaction_index >= 0),

    CONSTRAINT pool_swaps_log_index_non_negative
        CHECK (log_index >= 0),

    CONSTRAINT pool_swaps_amounts_opposite_signs
        CHECK
            (
            (
                amount0_raw > 0
                    AND amount1_raw < 0
                )
                OR
            (
                amount0_raw < 0
                    AND amount1_raw > 0
                )
            ),

    CONSTRAINT pool_swaps_sqrt_price_positive
        CHECK (sqrt_price_x96 > 0),

    CONSTRAINT pool_swaps_liquidity_non_negative
        CHECK (active_liquidity >= 0),

    CONSTRAINT pool_swaps_tick_range
        CHECK
            (
            tick >= -887272
                AND tick <= 887272
            )
);

-- statement-breakpoint

CREATE TABLE IF NOT EXISTS public.pool_swap_index_ranges
(
    pool_address              CHAR(42)    NOT NULL,
    from_block                BIGINT      NOT NULL,
    to_block                  BIGINT      NOT NULL,
    next_block                BIGINT      NOT NULL,

    status                    VARCHAR(16) NOT NULL,

    last_processed_block_hash CHAR(66),
    last_error                TEXT,

    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pool_swap_index_ranges_pkey
        PRIMARY KEY
            (
             pool_address,
             from_block,
             to_block
                ),

    CONSTRAINT pool_swap_index_ranges_pool_fkey
        FOREIGN KEY (pool_address)
            REFERENCES public.pools (address)
            ON UPDATE RESTRICT
            ON DELETE RESTRICT,

    CONSTRAINT pool_swap_index_ranges_valid_bounds
        CHECK
            (
            from_block > 0
                AND from_block <= to_block
                AND to_block < 9223372036854775807
            ),

    CONSTRAINT pool_swap_index_ranges_valid_next_block
        CHECK
            (
            next_block >= from_block
                AND next_block <= to_block + 1
            ),

    CONSTRAINT pool_swap_index_ranges_valid_status
        CHECK
            (
            status IN
            (
             'pending',
             'running',
             'complete',
             'failed'
                )
            ),

    CONSTRAINT pool_swap_index_ranges_complete_state
        CHECK
            (
            (
                status = 'complete'
                    AND next_block = to_block + 1
                )
                OR
            (
                status <> 'complete'
                    AND next_block <= to_block
                )
            ),

    CONSTRAINT pool_swap_index_ranges_processed_hash
        CHECK
            (
            (
                next_block = from_block
                    AND last_processed_block_hash IS NULL
                )
                OR
            (
                next_block > from_block
                    AND last_processed_block_hash IS NOT NULL
                )
            )
);