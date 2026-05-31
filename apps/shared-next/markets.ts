/** Demo reference mids for limit-order defaults (path A, not live prices). */
export const REF_MID: Record<string, string> = {
  "BTC/USDT": "100000",
  "ETH/USDT": "3500",
  "SOL/USDT": "180",
  "BNB/USDT": "600",
  "XRP/USDT": "2.2",
  "DOGE/USDT": "0.15",
  "LINK/USDT": "15",
  "AVAX/USDT": "35",
  "ADA/USDT": "0.55",
  "DOT/USDT": "7",
};

export function defaultPrice(symbol: string): string {
  return REF_MID[symbol] ?? "100";
}

export function defaultQty(symbol: string): string {
  if (symbol.startsWith("BTC")) return "0.01";
  if (symbol.startsWith("ETH") || symbol.startsWith("SOL") || symbol.startsWith("BNB"))
    return "1";
  if (symbol.includes("DOGE") || symbol.includes("XRP") || symbol.includes("ADA"))
    return "100";
  return "10";
}
