"use client";

import { useState } from "react";
import { AmmPanel } from "../components/AmmPanel";
import { OrderbookPanel } from "../components/OrderbookPanel";

type Tab = "amm" | "orderbook";

export default function DexPage() {
  const [tab, setTab] = useState<Tab>("amm");

  return (
    <main>
      <span className="badge">Phase 4 — AMM + OrderBook</span>
      <h1>DEX Web</h1>

      <div className="tabs">
        <button
          type="button"
          className={tab === "amm" ? "active" : ""}
          onClick={() => setTab("amm")}
        >
          AMM (Sepolia)
        </button>
        <button
          type="button"
          className={tab === "orderbook" ? "active" : ""}
          onClick={() => setTab("orderbook")}
        >
          OrderBook (simulated)
        </button>
      </div>

      {tab === "amm" ? <AmmPanel /> : <OrderbookPanel />}
    </main>
  );
}
