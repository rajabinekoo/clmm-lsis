# CLMM-LSIS

CLMM-LSIS is a reproducible research implementation for measuring the
position-level liquidity dependence of concentrated-liquidity automated
market makers.

The central research question is:

> Does LSIS measured before a liquidity withdrawal explain the realized
> deterioration in market quality beyond the size and geometry of the
> withdrawn position?

## Research pipeline

The repository implements the following pipeline:

1. Ingest ordered Ethereum pool events.
2. Reconstruct a Uniswap v3 pool at an exact historical event cursor.
3. Validate the swap simulator against realized on-chain swaps.
4. Build directional price-impact curves.
5. Summarize each curve using PIAUC.
6. Remove one position counterfactually.
7. Compute directional and total LSIS.
8. Identify realized liquidity-withdrawal events.
9. Simulate the exact removed liquidity amount.
10. Compare predicted and realized changes in PIAUC.
11. Compare LSIS with static position baselines.
12. Run a limited, pre-specified robustness analysis.

## Scope

The primary metric is LSIS.

The primary realized outcome is the post-withdrawal change in PIAUC.

The static comparison variables are:

- active liquidity share;
- range width;
- distance to the nearest range edge;
- normalized liquidity density.

The framework measures the marginal effect of removing one position while
holding the remaining pre-event pool state fixed. Simultaneous multi-position
withdrawals and coalition effects are outside the primary scope.

## Requirements

- Go 1.23 or newer
- PostgreSQL
- An archive-capable Ethereum RPC endpoint
- Python 3.11 or newer for statistical analysis

## Initial setup

Copy the environment template:

```bash
cp .env.example .env
```

Start PostgreSQL:

```bash
make postgres-up
```

Validate the study configuration:

```bash
make config-check
```

Run the tests:

```bash
make test
```

Build the CLI:

```bash
make build
```

## CLI

The initial commands are:

```bash
clmm-lsis version
clmm-lsis config-check --config configs/study.example.json
```

Additional commands will be implemented incrementally:

```bash
clmm-lsis migrate
clmm-lsis ingest
clmm-lsis reconstruct
clmm-lsis validate-swaps
clmm-lsis structural
clmm-lsis withdrawals
clmm-lsis export
```

## Reproducibility

Every study execution will create one immutable run directory:

```bash
artifacts/<run-id>/
├── manifest.json
├── validation/
├── structural/
├── withdrawals/
└── analysis/
```

No analysis script may discover an input file through an unrestricted
wildcard. Every generated artifact must be referenced by the run manifest.

## Security

Never commit:

- .env
- Ethereum RPC credentials
- database credentials
- private keys
- local database volumes

## License

MIT