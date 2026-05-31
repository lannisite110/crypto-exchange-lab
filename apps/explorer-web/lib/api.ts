const base = process.env.NEXT_PUBLIC_RPC_GATEWAY_URL ?? "http://localhost:8089";

type Envelope<T> = {
  ok: boolean;
  data?: T;
  error?: { code: string; message: string };
};

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`, { cache: "no-store" });
  const body: Envelope<T> = await res.json();
  if (!body.ok) throw new Error(body.error?.message ?? "request failed");
  return body.data as T;
}

export type Chain = {
  id: string;
  chain_id: number;
  name: string;
  rpc_url: string;
};

export type ChainStatus = {
  chain: Chain;
  last_indexed_block: number;
  live_head?: number;
  lag_blocks: number;
  updated_at: string;
};

export type Block = {
  chain_id: string;
  number: number;
  hash: string;
  parent_hash: string;
  timestamp: string;
  tx_count: number;
};

export type Transaction = {
  chain_id: string;
  hash: string;
  block_number: number;
  tx_index: number;
  from_addr: string;
  to_addr?: string;
  value_wei: string;
  gas_used?: number;
  status?: number;
};

export type ChainEvent = {
  id: number;
  chain_id: string;
  block_number: number;
  tx_hash: string;
  log_index: number;
  contract_address: string;
  event_type: string;
  payload?: Record<string, string>;
  created_at: string;
};

export type WatchedContract = {
  chain_id: string;
  address: string;
  label: string;
};

const CHAIN = "sepolia";

export function fetchChains() {
  return get<{ chains: Chain[] }>("/api/v1/chains").then((d) => d.chains);
}

export function fetchStatus(chain = CHAIN) {
  return get<ChainStatus>(`/api/v1/chains/${chain}/status`);
}

export function fetchBlocks(chain = CHAIN, limit = 15) {
  return get<{ blocks: Block[] }>(`/api/v1/chains/${chain}/blocks?limit=${limit}`).then(
    (d) => d.blocks
  );
}

export function fetchBlock(chain: string, number: number) {
  return get<Block>(`/api/v1/chains/${chain}/blocks/${number}`);
}

export function fetchBlockTxs(chain: string, number: number) {
  return get<{ transactions: Transaction[] }>(
    `/api/v1/chains/${chain}/blocks/${number}/transactions`
  ).then((d) => d.transactions);
}

export function fetchTx(chain: string, hash: string) {
  return get<Transaction>(`/api/v1/chains/${chain}/transactions/${hash}`);
}

export function fetchAddressTxs(chain: string, addr: string, limit = 20) {
  return get<{ transactions: Transaction[] }>(
    `/api/v1/chains/${chain}/addresses/${addr}/transactions?limit=${limit}`
  ).then((d) => d.transactions);
}

export function fetchBalance(chain: string, addr: string) {
  return get<{ balance_wei: string }>(`/api/v1/chains/${chain}/addresses/${addr}/balance`);
}

export function fetchEvents(chain = CHAIN, type?: string, limit = 25) {
  const q = new URLSearchParams({ limit: String(limit) });
  if (type) q.set("type", type);
  return get<{ events: ChainEvent[] }>(`/api/v1/chains/${chain}/events?${q}`).then(
    (d) => d.events
  );
}

export function fetchContracts(chain = CHAIN) {
  return get<{ contracts: WatchedContract[] }>(`/api/v1/chains/${chain}/contracts`).then(
    (d) => d.contracts
  );
}

export function fetchLiveHead(chain = CHAIN) {
  return get<{ number: number; hash: string; tx_count: number; source: string }>(
    `/api/v1/chains/${chain}/live/block/latest`
  );
}
