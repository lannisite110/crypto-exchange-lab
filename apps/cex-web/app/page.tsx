"use client";

import { useCallback, useEffect, useState } from "react";
import {
  BalanceRow,
  Depth,
  Order,
  Trade,
  User,
  cancelOrder,
  fetchBalances,
  fetchDepth,
  fetchOrders,
  fetchSymbols,
  fetchTrades,
  fetchUsers,
  matchingWsUrl,
  placeOrder,
} from "../lib/api";
import { defaultPrice, defaultQty } from "../../shared-next/markets";

export default function CexPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [userId, setUserId] = useState("");
  const [symbol, setSymbol] = useState("BTC/USDT");
  const [symbols, setSymbols] = useState<string[]>([]);
  const [balances, setBalances] = useState<BalanceRow | null>(null);
  const [depth, setDepth] = useState<Depth | null>(null);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [side, setSide] = useState("BUY");
  const [price, setPrice] = useState("100000");
  const [qty, setQty] = useState("0.01");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    if (!userId) return;
    try {
      const [b, d, t, o] = await Promise.all([
        fetchBalances(userId),
        fetchDepth(symbol),
        fetchTrades(symbol),
        fetchOrders(userId),
      ]);
      setBalances(b);
      setDepth(d);
      setTrades(t);
      setOrders(o);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "refresh failed");
    }
  }, [userId, symbol]);

  useEffect(() => {
    fetchUsers().then((u) => {
      setUsers(u);
      if (u[0]) setUserId(u[0].id);
    });
    fetchSymbols().then(setSymbols);
  }, []);

  useEffect(() => {
    setPrice(defaultPrice(symbol));
    setQty(defaultQty(symbol));
  }, [symbol]);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 4000);
    return () => clearInterval(id);
  }, [refresh]);

  useEffect(() => {
    const ws = new WebSocket(matchingWsUrl(symbol));
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.symbol === symbol) {
          setDepth({ symbol, bids: msg.bids, asks: msg.asks });
        }
      } catch {
        /* ignore */
      }
    };
    return () => ws.close();
  }, [symbol]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await placeOrder({
        user_id: userId,
        symbol,
        side,
        type: "LIMIT",
        price,
        quantity: qty,
      });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "order failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <span className="badge">Phase 1 — Simulated CEX</span>
      <h1>CEX Web</h1>
      <p className="muted">
        Spot order book with double-entry ledger. No real funds.
      </p>

      {error && <p className="error">{error}</p>}

      <section className="panel">
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
          Symbol
          <select value={symbol} onChange={(e) => setSymbol(e.target.value)}>
            {(symbols.length ? symbols : [symbol]).map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
      </section>

      <div className="grid">
        <section className="panel">
          <h2>Balances</h2>
          <table>
            <thead>
              <tr>
                <th>Asset</th>
                <th>Available</th>
                <th>Frozen</th>
              </tr>
            </thead>
            <tbody>
              {balances?.balances.map((b) => (
                <tr key={b.asset}>
                  <td>{b.asset}</td>
                  <td>{b.available}</td>
                  <td>{b.frozen}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <section className="panel">
          <h2>Place limit order</h2>
          <form onSubmit={onSubmit} className="form">
            <div className="row">
              <button
                type="button"
                className={side === "BUY" ? "buy active" : "buy"}
                onClick={() => setSide("BUY")}
              >
                Buy
              </button>
              <button
                type="button"
                className={side === "SELL" ? "sell active" : "sell"}
                onClick={() => setSide("SELL")}
              >
                Sell
              </button>
            </div>
            <label>
              Price
              <input value={price} onChange={(e) => setPrice(e.target.value)} />
            </label>
            <label>
              Quantity
              <input value={qty} onChange={(e) => setQty(e.target.value)} />
            </label>
            <button type="submit" disabled={loading}>
              {loading ? "Submitting…" : "Submit"}
            </button>
          </form>
        </section>
      </div>

      <div className="grid">
        <section className="panel">
          <h2>Order book</h2>
          <div className="book">
            <div>
              <h3>Asks</h3>
              {(depth?.asks ?? []).slice(0, 8).map((a) => (
                <div key={a.price} className="ask row-line">
                  <span>{a.price}</span>
                  <span>{a.quantity}</span>
                </div>
              ))}
            </div>
            <div>
              <h3>Bids</h3>
              {(depth?.bids ?? []).slice(0, 8).map((b) => (
                <div key={b.price} className="bid row-line">
                  <span>{b.price}</span>
                  <span>{b.quantity}</span>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="panel">
          <h2>Recent trades</h2>
          <ul className="trades">
            {trades.map((t) => (
              <li key={t.id}>
                {t.price} × {t.quantity}
              </li>
            ))}
          </ul>
        </section>
      </div>

      <section className="panel">
        <h2>Your orders</h2>
        <table>
          <thead>
            <tr>
              <th>Side</th>
              <th>Price</th>
              <th>Qty</th>
              <th>Filled</th>
              <th>Status</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {orders.map((o) => (
              <tr key={o.id}>
                <td>{o.side}</td>
                <td>{o.price ?? "—"}</td>
                <td>{o.quantity}</td>
                <td>{o.filled_qty}</td>
                <td>{o.status}</td>
                <td>
                  {(o.status === "NEW" || o.status === "PARTIALLY_FILLED") && (
                    <button
                      type="button"
                      className="link"
                      onClick={() => cancelOrder(o.id).then(refresh)}
                    >
                      Cancel
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </main>
  );
}
