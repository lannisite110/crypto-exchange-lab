const dexBase =
  process.env.NEXT_PUBLIC_ORDERBOOK_DEX_API_URL ?? "http://localhost:8084";
const accountBase =
  process.env.NEXT_PUBLIC_ACCOUNT_API_URL ?? "http://localhost:8081";
const matchingBase =
  process.env.NEXT_PUBLIC_MATCHING_API_URL ?? "http://localhost:8083";
const VENUE = "DEX";

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
export type Order = {
  id: string;
  venue?: string;
  user_id: string;
  symbol: string;
  side: string;
  type: string;
  status: string;
  price?: string;
  quantity: string;
  filled_qty: string;
};
export type Trade = {
  id: string;
  symbol: string;
  price: string;
  quantity: string;
};
export type Depth = {
  venue: string;
  symbol: string;
  bids: { price: string; quantity: string }[];
  asks: { price: string; quantity: string }[];
};

export function fetchUsers() {
  return get<User[]>(`${accountBase}/api/v1/users`);
}

export function fetchBalances(userId: string) {
  return get<BalanceRow>(`${accountBase}/api/v1/users/${userId}/balances`);
}

export function fetchSymbols() {
  return get<string[]>(`${dexBase}/api/v1/symbols`);
}

export function fetchVenueInfo() {
  return get<Record<string, string>>(`${dexBase}/api/v1/venue`);
}

export function fetchDepth(symbol: string) {
  return get<Depth>(
    `${matchingBase}/api/v1/markets/${encodeURIComponent(symbol)}/depth?venue=${VENUE}`
  );
}

export function fetchTrades(symbol: string) {
  return get<Trade[]>(
    `${dexBase}/api/v1/markets/${encodeURIComponent(symbol)}/trades`
  );
}

export function fetchOrders(userId: string) {
  return get<Order[]>(`${dexBase}/api/v1/orders?user_id=${userId}`);
}

export function placeOrder(payload: {
  user_id: string;
  symbol: string;
  side: string;
  type: string;
  price: string;
  quantity: string;
}) {
  return post<{ order: Order; trades: Trade[] }>(`${dexBase}/api/v1/orders`, payload);
}

export function cancelOrder(orderId: string) {
  return fetch(`${dexBase}/api/v1/orders/${orderId}`, { method: "DELETE" }).then(
    async (res) => {
      const body = await res.json();
      if (!body.ok) throw new Error(body.error?.message ?? "cancel failed");
      return body.data as Order;
    }
  );
}

export function matchingWsUrl(symbol: string) {
  const base =
    process.env.NEXT_PUBLIC_MATCHING_WS_URL ?? "ws://localhost:8083/ws/v1/market";
  return `${base}?venue=${VENUE}&symbol=${encodeURIComponent(symbol)}`;
}
