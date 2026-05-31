"use client";

import { useCallback, useEffect, useState } from "react";
import {
  BalanceRow,
  Market,
  Position,
  RiskRow,
  User,
  closePosition,
  fetchBalances,
  fetchFundingRate,
  fetchMarkPrices,
  fetchMarkets,
  fetchPositions,
  fetchRisk,
  fetchUsers,
  openPosition,
} from "../lib/api";

export default function HyperdexPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [userId, setUserId] = useState("");
  const [symbol, setSymbol] = useState("BTC-PERP");
  const [markets, setMarkets] = useState<Market[]>([]);
  const [marks, setMarks] = useState<Record<string, string>>({});
  const [balances, setBalances] = useState<BalanceRow | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [risks, setRisks] = useState<RiskRow[]>([]);
  const [fundingRate, setFundingRate] = useState("");
  const [side, setSide] = useState("LONG");
  const [size, setSize] = useState("0.01");
  const [leverage, setLeverage] = useState(10);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    if (!userId) return;
    try {
      const [b, p, r, m, fr] = await Promise.all([
        fetchBalances(userId),
        fetchPositions(userId),
        fetchRisk(userId),
        fetchMarkPrices(),
        fetchFundingRate(symbol),
      ]);
      setBalances(b);
      setPositions(p);
      setRisks(r.positions ?? []);
      setMarks(m);
      setFundingRate(fr.rate ?? "");
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "refresh failed");
    }
  }, [userId, symbol]);

  useEffect(() => {
    fetchUsers().then((u) => {
      setUsers(u.filter((x) => x.username !== "perp_house"));
      if (u[0]) setUserId(u.find((x) => x.username === "alice")?.id ?? u[0].id);
    });
    fetchMarkets().then(setMarkets);
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 5000);
    return () => clearInterval(id);
  }, [refresh]);

  async function onOpen(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await openPosition({ user_id: userId, symbol, side, size, leverage });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "open failed");
    } finally {
      setLoading(false);
    }
  }

  async function onClose(pos: Position) {
    setError("");
    try {
      await closePosition({ user_id: userId, symbol: pos.symbol });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "close failed");
    }
  }

  const mark = marks[symbol] ?? "—";

  return (
    <main>
      <span className="badge">Phase 3 — Perpetuals</span>
      <h1>HyperDEX</h1>
      <p className="muted">
        USDT-margined perpetuals with leverage, mark price, liquidation scan, and
        funding (simulated).
      </p>

      {error && <p className="error">{error}</p>}

      <section className="panel row-controls">
        <label>
          Trader
          <select value={userId} onChange={(e) => setUserId(e.target.value)}>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.username}
              </option>
            ))}
          </select>
        </label>
        <label>
          Market
          <select value={symbol} onChange={(e) => setSymbol(e.target.value)}>
            {markets.map((m) => (
              <option key={m.symbol} value={m.symbol}>
                {m.symbol}
              </option>
            ))}
          </select>
        </label>
        <div className="stat">
          <span className="label">Mark</span>
          <span className="value">{mark}</span>
        </div>
        <div className="stat">
          <span className="label">Funding rate</span>
          <span className="value">{fundingRate || "—"}</span>
        </div>
      </section>

      <div className="grid">
        <section className="panel">
          <h2>USDT balance</h2>
          <table>
            <tbody>
              {balances?.balances
                .filter((b) => b.asset === "USDT")
                .map((b) => (
                  <tr key={b.asset}>
                    <td>Available</td>
                    <td>{b.available}</td>
                  </tr>
                ))}
              {balances?.balances
                .filter((b) => b.asset === "USDT")
                .map((b) => (
                  <tr key={b.asset + "-f"}>
                    <td>Frozen (margin)</td>
                    <td>{b.frozen}</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </section>

        <section className="panel">
          <h2>Open position</h2>
          <form onSubmit={onOpen} className="form">
            <div className="row">
              <button
                type="button"
                className={side === "LONG" ? "long active" : "long"}
                onClick={() => setSide("LONG")}
              >
                Long
              </button>
              <button
                type="button"
                className={side === "SHORT" ? "short active" : "short"}
                onClick={() => setSide("SHORT")}
              >
                Short
              </button>
            </div>
            <label>
              Size (BTC/ETH)
              <input value={size} onChange={(e) => setSize(e.target.value)} />
            </label>
            <label>
              Leverage
              <input
                type="number"
                min={1}
                max={20}
                value={leverage}
                onChange={(e) => setLeverage(Number(e.target.value))}
              />
            </label>
            <button type="submit" disabled={loading}>
              {loading ? "Opening…" : "Open / Add"}
            </button>
          </form>
        </section>
      </div>

      <section className="panel">
        <h2>Positions</h2>
        <table>
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Side</th>
              <th>Size</th>
              <th>Entry</th>
              <th>Leverage</th>
              <th>uPnL</th>
              <th>Margin ratio</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {positions.map((p) => (
              <tr key={p.id} className={p.liquidation_risk ? "danger" : ""}>
                <td>{p.symbol}</td>
                <td>{p.side}</td>
                <td>{p.size}</td>
                <td>{p.entry_price}</td>
                <td>{p.leverage}x</td>
                <td>{p.unrealized_pnl ?? "—"}</td>
                <td>{p.margin_ratio ?? "—"}</td>
                <td>
                  <button type="button" className="link" onClick={() => onClose(p)}>
                    Close
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="panel">
        <h2>Risk engine</h2>
        <ul className="risk-list">
          {risks.map((r) => (
            <li key={r.symbol} className={r.liquidation_risk ? "danger" : ""}>
              {r.symbol} {r.side} — ratio {r.margin_ratio}
              {r.liquidation_risk ? " ⚠ liquidation risk" : ""}
            </li>
          ))}
          {risks.length === 0 && <li className="muted">No open positions</li>}
        </ul>
      </section>
    </main>
  );
}
