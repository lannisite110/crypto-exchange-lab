const hyperBase =
  process.env.NEXT_PUBLIC_HYPERLIQUID_API_URL ?? "http://localhost:8085";
const riskBase =
  process.env.NEXT_PUBLIC_RISK_API_URL ?? "http://localhost:8086";
const fundingBase =
  process.env.NEXT_PUBLIC_FUNDING_API_URL ?? "http://localhost:8088";
const accountBase =
  process.env.NEXT_PUBLIC_ACCOUNT_API_URL ?? "http://localhost:8081";

type Envelope<T> = {
  ok: boolean;
  data?: T;
  error?: { code: string; message: string };
};

async function get<T>(url: string): Promise<T> {
  const res = await fetch(url, { cache: "no-store" });
  const body: Envelope<T> = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "request failed");
  return body.data as T;
}

async function post<T>(url: string, payload: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body: Envelope<T> = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "request failed");
  return body.data as T;
}

export type User = { id: string; username: string };
export type BalanceRow = {
  user_id: string;
  balances: { asset: string; available: string; frozen: string }[];
};
export type Market = {
  symbol: string;
  spot_symbol: string;
  max_leverage: number;
  maint_margin_rate: string;
};
export type Position = {
  id: string;
  symbol: string;
  side: string;
  size: string;
  entry_price: string;
  leverage: number;
  margin: string;
  mark_price?: string;
  unrealized_pnl?: string;
  margin_ratio?: string;
  liquidation_risk?: boolean;
};
export type RiskRow = {
  symbol: string;
  side: string;
  margin_ratio: string;
  liquidation_risk: boolean;
  unrealized_pnl: string;
  equity: string;
};

export function fetchUsers() {
  return get<User[]>(`${accountBase}/api/v1/users`);
}
export function fetchBalances(userId: string) {
  return get<BalanceRow>(`${accountBase}/api/v1/users/${userId}/balances`);
}
export function fetchMarkets() {
  return get<Market[]>(`${hyperBase}/api/v1/markets`);
}
export function fetchMarkPrices() {
  return get<Record<string, string>>(`${hyperBase}/api/v1/mark-prices`);
}
export function fetchPositions(userId: string) {
  return get<Position[]>(`${hyperBase}/api/v1/positions?user_id=${userId}`);
}
export function fetchRisk(userId: string) {
  return get<{ positions: RiskRow[] }>(`${riskBase}/api/v1/users/${userId}/risk`);
}
export function fetchFundingRate(symbol: string) {
  return get<{ rate: string }>(`${fundingBase}/api/v1/funding/rates?symbol=${symbol}`);
}
export function openPosition(payload: {
  user_id: string;
  symbol: string;
  side: string;
  size: string;
  leverage: number;
}) {
  return post<Position>(`${hyperBase}/api/v1/positions/open`, payload);
}
export function closePosition(payload: {
  user_id: string;
  symbol: string;
  size?: string;
}) {
  return post<{ position: Position | null; realized_pnl: string }>(
    `${hyperBase}/api/v1/positions/close`,
    payload
  );
}
