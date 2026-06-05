"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Block,
  ChainEvent,
  ChainStatus,
  Transaction,
  WatchedContract,
  fetchBlocks,
  fetchContracts,
  fetchEvents,
  fetchStatus,
  fetchTx,
  fetchAddressTxs,
  fetchBalance,
  fetchBlockTxs,
} from "../lib/api";

type Tab = "blocks" | "events" | "search";

export default function ExplorerPage() {
  const [tab, setTab] = useState<Tab>("blocks");
  const [status, setStatus] = useState<ChainStatus | null>(null);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [events, setEvents] = useState<ChainEvent[]>([]);
  const [contracts, setContracts] = useState<WatchedContract[]>([]);
  const [selectedBlock, setSelectedBlock] = useState<number | null>(null);
  const [blockTxs, setBlockTxs] = useState<Transaction[]>([]);
  const [search, setSearch] = useState("");
  const [searchResult, setSearchResult] = useState<
    | { kind: "tx"; data: Transaction }
    | { kind: "addr"; txs: Transaction[]; balance?: string }
    | null
  >(null);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    try {
      const [st, bl, ev, ct] = await Promise.all([
        fetchStatus(),
        fetchBlocks(),
        fetchEvents(undefined, "Swap"),
        fetchContracts(),
      ]);
      setStatus(st);
      setBlocks(bl ?? []);
      setEvents(ev ?? []);
      setContracts(ct ?? []);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "load failed");
    }
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 8000);
    return () => clearInterval(id);
  }, [refresh]);

  useEffect(() => {
    if (selectedBlock == null) return;
    fetchBlockTxs("sepolia", selectedBlock)
      .then(setBlockTxs)
      .catch(() => setBlockTxs([]));
  }, [selectedBlock]);

  async function onSearch(e: React.FormEvent) {
    e.preventDefault();
    setSearchResult(null);
    setError("");
    const q = search.trim();
    if (!q) return;
    try {
      if (q.startsWith("0x") && q.length === 66) {
        const tx = await fetchTx("sepolia", q);
        setSearchResult({ kind: "tx", data: tx });
        return;
      }
      const addr = q.startsWith("0x") ? q : `0x${q}`;
      const [txs, bal] = await Promise.all([
        fetchAddressTxs("sepolia", addr),
        fetchBalance("sepolia", addr),
      ]);
      setSearchResult({ kind: "addr", txs, balance: bal.balance_wei });
    } catch (err) {
      setError(err instanceof Error ? err.message : "not found");
    }
  }

  return (
    <main>
      <span className="badge">Phase 5 — Explorer</span>
      <h1>Blockchain Explorer</h1>
      <p className="muted">
        Indexed Sepolia data from the lab AMM (LAB/LUSD). Requires indexer +
        rpc-gateway. Read-only — no real funds.
      </p>

      {status && (
        <section className="panel stats">
          <div>
            <span className="label">Chain</span>
            <strong>{status.chain.name}</strong>
          </div>
          <div>
            <span className="label">Indexed block</span>
            <strong>{status.last_indexed_block}</strong>
          </div>
          <div>
            <span className="label">Live head</span>
            <strong>{status.live_head ?? "—"}</strong>
          </div>
          <div>
            <span className="label">Lag</span>
            <strong>{status.lag_blocks} blocks</strong>
          </div>
        </section>
      )}

      {contracts.length > 0 && (
        <p className="muted small">
          Watching: {contracts.map((c) => c.label).join(", ")}
        </p>
      )}

      {error && <p className="error">{error}</p>}

      <div className="tabs">
        <button
          type="button"
          className={tab === "blocks" ? "active" : ""}
          onClick={() => setTab("blocks")}
        >
          Blocks
        </button>
        <button
          type="button"
          className={tab === "events" ? "active" : ""}
          onClick={() => setTab("events")}
        >
          AMM Swaps
        </button>
        <button
          type="button"
          className={tab === "search" ? "active" : ""}
          onClick={() => setTab("search")}
        >
          Search
        </button>
      </div>

      {tab === "blocks" && (
        <div className="grid">
          <section className="panel">
            <h2>Recent blocks</h2>
            <table>
              <thead>
                <tr>
                  <th>#</th>
                  <th>Hash</th>
                  <th>Txs</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {(blocks ?? []).map((b) => (
                  <tr
                    key={b.number}
                    className={selectedBlock === b.number ? "selected" : ""}
                    onClick={() => setSelectedBlock(b.number)}
                  >
                    <td>{b.number}</td>
                    <td className="mono">{short(b.hash)}</td>
                    <td>{b.tx_count}</td>
                    <td>{new Date(b.timestamp).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {blocks.length === 0 && (
              <p className="muted">No indexed blocks yet. Start the indexer service.</p>
            )}
          </section>
          <section className="panel">
            <h2>
              {selectedBlock != null
                ? `Transactions in block ${selectedBlock}`
                : "Select a block"}
            </h2>
            <ul className="tx-list">
              {blockTxs.map((t) => (
                <li key={t.hash}>
                  <span className="mono">{short(t.hash)}</span>
                  <span>
                    {short(t.from_addr)} → {short(t.to_addr || "—")}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        </div>
      )}

      {tab === "events" && (
        <section className="panel">
          <h2>AMM Swap events</h2>
          <table>
            <thead>
              <tr>
                <th>Block</th>
                <th>Tx</th>
                <th>Pool</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              {(events ?? []).map((ev) => (
                <tr key={ev.id}>
                  <td>{ev.block_number}</td>
                  <td className="mono">{short(ev.tx_hash)}</td>
                  <td className="mono">{short(ev.contract_address)}</td>
                  <td className="mono small">
                    {ev.payload
                      ? JSON.stringify(ev.payload)
                      : ev.event_type}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {events.length === 0 && (
            <p className="muted">
              No swap events indexed. Set INDEXER_AMM_PAIR and run indexer after
              AMM activity on Sepolia.
            </p>
          )}
        </section>
      )}

      {tab === "search" && (
        <section className="panel">
          <h2>Search</h2>
          <form onSubmit={onSearch} className="form">
            <label>
              Tx hash (0x…) or address
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="0x…"
              />
            </label>
            <button type="submit">Search</button>
          </form>
          {searchResult?.kind === "tx" && (
            <pre className="mono result">{JSON.stringify(searchResult.data, null, 2)}</pre>
          )}
          {searchResult?.kind === "addr" && (
            <div className="result">
              <p>Balance (wei): {searchResult.balance}</p>
              <ul className="tx-list">
                {(searchResult.txs ?? []).map((t) => (
                  <li key={t.hash}>
                    {short(t.hash)} — block {t.block_number}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>
      )}
    </main>
  );
}

function short(s: string) {
  if (s.length <= 14) return s;
  return `${s.slice(0, 8)}…${s.slice(-6)}`;
}
