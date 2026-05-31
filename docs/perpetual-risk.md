# Perpetual futures (Phase 3)

## Margin

- **Notional** = `size × mark_price`
- **Initial margin** = `notional / leverage` (isolated, frozen in USDT)
- **Maintenance margin** = `notional × maint_margin_rate` (default 0.5%)
- **Equity** = `margin + unrealized_pnl`
- **Margin ratio** = `equity / maintenance_margin` — liquidate when **&lt; 1**

## PnL

| Side  | Unrealized |
|-------|------------|
| LONG  | `(mark − entry) × size` |
| SHORT | `(entry − mark) × size` |

## Funding

Positive rate → **longs pay shorts**. Payment = `size × mark × rate` per interval.

Demo interval: 5 minutes (`FUNDING_INTERVAL`). Production would use 8h.

## Services

| Service | Port | Role |
|---------|------|------|
| hyperliquid-engine | 8085 | Open/close, mark feed from CEX mid |
| risk-engine | 8086 | Margin ratio API |
| liquidation-engine | 8087 | Scans & force-closes |
| funding-engine | 8088 | Periodic funding settlement |

## House account

`perp_house` balances PnL and funding legs (zero-sum ledger with users).
